package handlers

import (
	"database/sql"
	"encoding/json"
	"errors"
	"log/slog"
	"net/http"

	"github.com/levionstudio/fintech/internal/models"
	"github.com/levionstudio/fintech/internal/store"
	"github.com/levionstudio/fintech/internal/utils"
)

type ElectricityBillHandler struct {
	billStore store.ElectricityBillStore
	logger    *slog.Logger
}

func NewElectricityBillHandler(billStore store.ElectricityBillStore, logger *slog.Logger) *ElectricityBillHandler {
	return &ElectricityBillHandler{
		billStore: billStore,
		logger:    logger,
	}
}

func (eh *ElectricityBillHandler) HandleCreateElectricityBill(w http.ResponseWriter, r *http.Request) {
	var eb models.ElectricityBillModel
	if err := json.NewDecoder(r.Body).Decode(&eb); err != nil {
		utils.BadRequest(w, eh.logger, "create electricity bill", err)
		return
	}

	if err := eb.Validate(); err != nil {
		utils.BadRequest(w, eh.logger, "create electricity bill", err)
		return
	}

	if len(eb.RetailerID) == 0 || eb.RetailerID[0] != 'R' {
		utils.BadRequest(w, eh.logger, "create electricity bill", errors.New("retailer_id must start with R"))
		return
	}

	if err := eh.billStore.InitializeElectricityBill(&eb); err != nil {
		if isElectricityClientErr(err) {
			utils.BadRequest(w, eh.logger, "create electricity bill", err)
			return
		}
		utils.ServerError(w, eh.logger, "create electricity bill", err)
		return
	}

	apiResp, finalStatus, orderID, operatorTxnID := callElectricityBillAPI(eh.logger, &eb)

	if err := eh.billStore.FinalizeElectricityBill(eb.ElectricityBillTransactionID, operatorTxnID, orderID, finalStatus); err != nil {
		eh.logger.Error("failed to finalize electricity bill", "error", err, "id", eb.ElectricityBillTransactionID)
	}

	eb.TransactionStatus = finalStatus
	eb.OrderID = &orderID
	eb.OperatorTransactionID = &operatorTxnID

	utils.WriteJSON(w, http.StatusCreated, utils.Envelope{
		"message":      "electricity bill payment processed",
		"bill":         eb,
		"api_response": apiResp,
	})
}

func callElectricityBillAPI(logger *slog.Logger, eb *models.ElectricityBillModel) (resp *models.APIResponseModel, finalStatus, orderID, operatorTxnID string) {
	finalStatus = "FAILED"

	if utils.RechargeKitAPI1 == "" || utils.RechargeKitAPIToken == "" {
		logger.Error("electricity bill api not configured", "id", eb.ElectricityBillTransactionID)
		return
	}

	var apiResp models.APIResponseModel
	err := utils.PostRequest(
		utils.RechargeKitAPI1+utils.ElectricityBill,
		"Authorization",
		"Bearer "+utils.RechargeKitAPIToken,
		map[string]any{
			"p1":                 eb.CustomerID,
			"p2":                 "",
			"p3":                 "",
			"operator_code":      eb.OperatorCode,
			"amount":             eb.Amount,
			"customer_email":     eb.CustomerEmail,
			"partner_request_id": eb.PartnerRequestID,
		},
		&apiResp,
	)
	if err != nil {
		logger.Error("electricity bill api call failed", "error", err, "id", eb.ElectricityBillTransactionID)
		return
	}

	resp = &apiResp
	orderID = apiResp.OrderID
	operatorTxnID = apiResp.OperatorTransactionID

	if apiResp.Error != 0 {
		logger.Error("electricity bill api error", "msg", apiResp.Message, "id", eb.ElectricityBillTransactionID)
		return
	}

	switch apiResp.Status {
	case 1:
		finalStatus = "SUCCESS"
	case 2:
		finalStatus = "PENDING"
	default:
		finalStatus = "FAILED"
	}
	return
}

func (eh *ElectricityBillHandler) HandleCheckElectricityBillStatus(w http.ResponseWriter, r *http.Request) {
	id, err := utils.ReadParamIDInt(r)
	if err != nil {
		utils.BadRequest(w, eh.logger, "check electricity bill status", err)
		return
	}

	eb, err := eh.billStore.GetElectricityBillByID(id)
	if err != nil {
		if err.Error() == "electricity bill not found" {
			utils.BadRequest(w, eh.logger, "check electricity bill status", err)
			return
		}
		utils.ServerError(w, eh.logger, "check electricity bill status", err)
		return
	}

	if eb.TransactionStatus != "PENDING" {
		utils.WriteJSON(w, http.StatusOK, utils.Envelope{
			"message": "bill already finalized",
			"bill":    eb,
		})
		return
	}

	apiResp, finalStatus, orderID, operatorTxnID := callElectricityBillStatusAPI(eh.logger, eb.PartnerRequestID, eb.ElectricityBillTransactionID)

	if err = eh.billStore.FinalizeElectricityBill(eb.ElectricityBillTransactionID, operatorTxnID, orderID, finalStatus); err != nil {
		if err.Error() == "electricity bill not found or already finalized" {
			utils.BadRequest(w, eh.logger, "check electricity bill status", err)
			return
		}
		utils.ServerError(w, eh.logger, "check electricity bill status finalize", err)
		return
	}

	eb.TransactionStatus = finalStatus
	eb.OrderID = &orderID
	eb.OperatorTransactionID = &operatorTxnID

	msg := "bill status updated"
	if finalStatus == "PENDING" {
		msg = "bill still pending"
	}

	utils.WriteJSON(w, http.StatusOK, utils.Envelope{
		"message":      msg,
		"bill":         eb,
		"api_response": apiResp,
	})
}

func callElectricityBillStatusAPI(logger *slog.Logger, partnerRequestID string, id int64) (resp *models.APIResponseModel, finalStatus, orderID, operatorTxnID string) {
	finalStatus = "PENDING"

	if utils.RechargeKitAPI1 == "" || utils.RechargeKitAPIToken == "" {
		logger.Error("electricity bill status api not configured", "id", id)
		return
	}

	var apiResp models.APIResponseModel
	err := utils.PostRequest(
		utils.RechargeKitAPI1+utils.PayoutStatus,
		"Authorization",
		"Bearer "+utils.RechargeKitAPIToken,
		map[string]any{
			"partner_request_id": partnerRequestID,
		},
		&apiResp,
	)
	if err != nil {
		logger.Error("electricity bill status api call failed", "error", err, "id", id)
		return
	}

	resp = &apiResp
	orderID = apiResp.OrderID
	operatorTxnID = apiResp.OperatorTransactionID

	if apiResp.Error != 0 {
		logger.Error("electricity bill status api error", "msg", apiResp.Message, "id", id)
		return
	}

	switch apiResp.Status {
	case 1:
		finalStatus = "SUCCESS"
	case 3:
		finalStatus = "FAILED"
	default:
		finalStatus = "PENDING"
	}
	return
}

func (eh *ElectricityBillHandler) HandleRefundElectricityBill(w http.ResponseWriter, r *http.Request) {
	id, err := utils.ReadParamIDInt(r)
	if err != nil {
		utils.BadRequest(w, eh.logger, "refund electricity bill", err)
		return
	}

	if err = eh.billStore.RefundElectricityBill(id); err != nil {
		if isElectricityClientErr(err) {
			utils.BadRequest(w, eh.logger, "refund electricity bill", err)
			return
		}
		utils.ServerError(w, eh.logger, "refund electricity bill", err)
		return
	}

	utils.WriteJSON(w, http.StatusOK, utils.Envelope{"message": "electricity bill refunded"})
}

func (eh *ElectricityBillHandler) HandleGetAllElectricityBills(w http.ResponseWriter, r *http.Request) {
	p := utils.ReadQueryParams(r)
	results, err := eh.billStore.GetAllElectricityBills(p)
	if err != nil {
		utils.ServerError(w, eh.logger, "get all electricity bills", err)
		return
	}
	utils.WriteJSON(w, http.StatusOK, utils.Envelope{"message": "electricity bills fetched", "electricity_bills": results})
}

func (eh *ElectricityBillHandler) HandleGetElectricityBillsByRetailerID(w http.ResponseWriter, r *http.Request) {
	id, err := utils.ReadParamID(r)
	if err != nil {
		utils.BadRequest(w, eh.logger, "get electricity bills by retailer", err)
		return
	}
	p := utils.ReadQueryParams(r)
	results, err := eh.billStore.GetElectricityBillsByRetailerID(id, p)
	if err != nil {
		utils.ServerError(w, eh.logger, "get electricity bills by retailer", err)
		return
	}
	utils.WriteJSON(w, http.StatusOK, utils.Envelope{"message": "electricity bills fetched", "electricity_bills": results})
}

func (eh *ElectricityBillHandler) HandleGetElectricityBillsByDistributorID(w http.ResponseWriter, r *http.Request) {
	id, err := utils.ReadParamID(r)
	if err != nil {
		utils.BadRequest(w, eh.logger, "get electricity bills by distributor", err)
		return
	}
	p := utils.ReadQueryParams(r)
	results, err := eh.billStore.GetElectricityBillsByDistributorID(id, p)
	if err != nil {
		utils.ServerError(w, eh.logger, "get electricity bills by distributor", err)
		return
	}
	utils.WriteJSON(w, http.StatusOK, utils.Envelope{"message": "electricity bills fetched", "electricity_bills": results})
}

func (eh *ElectricityBillHandler) HandleGetElectricityBillsByMasterDistributorID(w http.ResponseWriter, r *http.Request) {
	id, err := utils.ReadParamID(r)
	if err != nil {
		utils.BadRequest(w, eh.logger, "get electricity bills by md", err)
		return
	}
	p := utils.ReadQueryParams(r)
	results, err := eh.billStore.GetElectricityBillsByMasterDistributorID(id, p)
	if err != nil {
		utils.ServerError(w, eh.logger, "get electricity bills by md", err)
		return
	}
	utils.WriteJSON(w, http.StatusOK, utils.Envelope{"message": "electricity bills fetched", "electricity_bills": results})
}

func (eh *ElectricityBillHandler) HandleFetchElectricityBill(w http.ResponseWriter, r *http.Request) {
	var req models.ElectricityBillFetchRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		utils.BadRequest(w, eh.logger, "fetch electricity bill", err)
		return
	}
	if req.ConsumerID == "" {
		utils.BadRequest(w, eh.logger, "fetch electricity bill", errors.New("consumer_id is required"))
		return
	}
	if req.OperatorCode == 0 {
		utils.BadRequest(w, eh.logger, "fetch electricity bill", errors.New("operator_code is required"))
		return
	}
	if utils.RechargeKitAPI1 == "" || utils.RechargeKitAPIToken == "" {
		utils.ServerError(w, eh.logger, "fetch electricity bill", errors.New("recharge api not configured"))
		return
	}

	var resp models.ElectricityBillFetchResponse
	if err := utils.PostRequest(
		utils.RechargeKitAPI1+utils.ElectricityBillFetch,
		"Authorization",
		"Bearer "+utils.RechargeKitAPIToken,
		map[string]any{
			"consumer_id":   req.ConsumerID,
			"operator_code": req.OperatorCode,
		},
		&resp,
	); err != nil {
		utils.ServerError(w, eh.logger, "fetch electricity bill", err)
		return
	}

	utils.WriteJSON(w, http.StatusOK, utils.Envelope{"bill": resp})
}

func (eh *ElectricityBillHandler) HandleCreateElectricityOperator(w http.ResponseWriter, r *http.Request) {
	var op models.ElectricityOperatorModel
	if err := json.NewDecoder(r.Body).Decode(&op); err != nil {
		utils.BadRequest(w, eh.logger, "create electricity operator", err)
		return
	}
	if op.OperatorCode == 0 || op.OperatorName == "" {
		utils.BadRequest(w, eh.logger, "create electricity operator", errors.New("operator_code and operator_name are required"))
		return
	}
	if err := eh.billStore.CreateElectricityOperator(op); err != nil {
		utils.ServerError(w, eh.logger, "create electricity operator", err)
		return
	}
	utils.WriteJSON(w, http.StatusCreated, utils.Envelope{"message": "operator created"})
}

func (eh *ElectricityBillHandler) HandleUpdateElectricityOperator(w http.ResponseWriter, r *http.Request) {
	code, err := utils.ReadParamIDInt(r)
	if err != nil {
		utils.BadRequest(w, eh.logger, "update electricity operator", err)
		return
	}
	var op models.ElectricityOperatorModel
	if err := json.NewDecoder(r.Body).Decode(&op); err != nil {
		utils.BadRequest(w, eh.logger, "update electricity operator", err)
		return
	}
	if op.OperatorName == "" {
		utils.BadRequest(w, eh.logger, "update electricity operator", errors.New("operator_name is required"))
		return
	}
	op.OperatorCode = int(code)
	if err := eh.billStore.UpdateElectricityOperator(op); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			utils.BadRequest(w, eh.logger, "update electricity operator", errors.New("operator not found"))
			return
		}
		utils.ServerError(w, eh.logger, "update electricity operator", err)
		return
	}
	utils.WriteJSON(w, http.StatusOK, utils.Envelope{"message": "operator updated"})
}

func (eh *ElectricityBillHandler) HandleDeleteElectricityOperator(w http.ResponseWriter, r *http.Request) {
	code, err := utils.ReadParamIDInt(r)
	if err != nil {
		utils.BadRequest(w, eh.logger, "delete electricity operator", err)
		return
	}
	if err := eh.billStore.DeleteElectricityOperator(int(code)); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			utils.BadRequest(w, eh.logger, "delete electricity operator", errors.New("operator not found"))
			return
		}
		utils.ServerError(w, eh.logger, "delete electricity operator", err)
		return
	}
	utils.WriteJSON(w, http.StatusOK, utils.Envelope{"message": "operator deleted"})
}

func (eh *ElectricityBillHandler) HandleGetElectricityOperators(w http.ResponseWriter, r *http.Request) {
	operators, err := eh.billStore.GetElectricityOperators()
	if err != nil {
		utils.ServerError(w, eh.logger, "get electricity operators", err)
		return
	}
	utils.WriteJSON(w, http.StatusOK, utils.Envelope{"message": "operators fetched", "operators": operators})
}

func isElectricityClientErr(err error) bool {
	msg := err.Error()
	return msg == "retailer not found" ||
		msg == "retailer KYC is not verified" ||
		msg == "retailer is blocked" ||
		msg == "insufficient wallet balance" ||
		msg == "operator not found" ||
		msg == "electricity bill not found or already finalized" ||
		msg == "electricity bill not found or already refunded" ||
		msg == "only FAILED electricity bills can be refunded"
}
