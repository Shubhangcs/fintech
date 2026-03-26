package handlers

import (
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"net/http"

	"github.com/levionstudio/fintech/internal/models"
	"github.com/levionstudio/fintech/internal/store"
	"github.com/levionstudio/fintech/internal/utils"
)

type MobileRechargeHandler struct {
	rechargeStore store.MobileRechargeStore
	logger        *slog.Logger
}

func NewMobileRechargeHandler(rechargeStore store.MobileRechargeStore, logger *slog.Logger) *MobileRechargeHandler {
	return &MobileRechargeHandler{
		rechargeStore: rechargeStore,
		logger:        logger,
	}
}

func (mh *MobileRechargeHandler) HandleCreateMobileRecharge(w http.ResponseWriter, r *http.Request) {
	var mr models.MobileRechargeModel
	if err := json.NewDecoder(r.Body).Decode(&mr); err != nil {
		utils.BadRequest(w, mh.logger, "create mobile recharge", err)
		return
	}

	if err := mr.ValidateInitializeMobileRecharge(); err != nil {
		utils.BadRequest(w, mh.logger, "create mobile recharge", err)
		return
	}

	if len(mr.RetailerID) == 0 || mr.RetailerID[0] != 'R' {
		utils.BadRequest(w, mh.logger, "create mobile recharge", errors.New("retailer_id must start with R"))
		return
	}

	if err := mh.rechargeStore.InitializeMobileRecharge(&mr); err != nil {
		if isRechargeClientErr(err) {
			utils.BadRequest(w, mh.logger, "create mobile recharge", err)
			return
		}
		utils.ServerError(w, mh.logger, "create mobile recharge", err)
		return
	}

	apiResp, finalStatus, orderID, operatorTxnID := callMobileRechargeAPI(mh.logger, &mr)

	if err := mh.rechargeStore.FinalizeMobileRecharge(mr.MobileRechargeTransactionID, operatorTxnID, orderID, finalStatus); err != nil {
		mh.logger.Error("failed to finalize mobile recharge", "error", err, "id", mr.MobileRechargeTransactionID)
	}

	mr.RechargeStatus = finalStatus
	mr.OrderID = orderID
	mr.OperatorTransactionID = operatorTxnID
	utils.WriteJSON(w, http.StatusCreated, utils.Envelope{
		"message":      "mobile recharge processed",
		"recharge":     mr,
		"api_response": apiResp,
	})
}

func callMobileRechargeAPI(logger *slog.Logger, mr *models.MobileRechargeModel) (resp *models.APIResponseModel, finalStatus, orderID, operatorTxnID string) {
	finalStatus = "FAILED"

	if utils.RechargeKitAPI1 == "" || utils.RechargeKitAPIToken == "" {
		logger.Error("mobile recharge api not configured", "id", mr.MobileRechargeTransactionID)
		return
	}

	endpoint := utils.MobileRecharge
	if mr.RechargeType == "POSTPAID" {
		endpoint = utils.PostpaidMobileRecharge
	}

	rechargeType := 1
	if mr.RechargeType == "POSTPAID" {
		rechargeType = 2
	}

	var apiResp models.APIResponseModel
	err := utils.PostRequest(
		utils.RechargeKitAPI1+endpoint,
		"Authorization",
		"Bearer "+utils.RechargeKitAPIToken,
		map[string]any{
			"mobile_no":          mr.MobileNumber,
			"operator_code":      mr.OperatorCode,
			"circle":             mr.CircleCode,
			"amount":             mr.Amount,
			"recharge_type":      rechargeType,
			"partner_request_id": mr.PartnerRequestID,
		},
		&apiResp,
	)
	if err != nil {
		logger.Error("mobile recharge api call failed", "error", err, "id", mr.MobileRechargeTransactionID)
		return
	}

	resp = &apiResp
	orderID = apiResp.OrderID
	operatorTxnID = apiResp.OperatorTransactionID

	if apiResp.Error != 0 {
		logger.Error("mobile recharge api error", "msg", apiResp.Message, "id", mr.MobileRechargeTransactionID)
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

func (mh *MobileRechargeHandler) HandleCheckMobileRechargeStatus(w http.ResponseWriter, r *http.Request) {
	id, err := utils.ReadParamIDInt(r)
	if err != nil {
		utils.BadRequest(w, mh.logger, "check mobile recharge status", err)
		return
	}

	mr, err := mh.rechargeStore.GetMobileRechargeByID(id)
	if err != nil {
		if err.Error() == "recharge not found" {
			utils.BadRequest(w, mh.logger, "check mobile recharge status", err)
			return
		}
		utils.ServerError(w, mh.logger, "check mobile recharge status", err)
		return
	}

	if mr.RechargeStatus != "PENDING" {
		utils.WriteJSON(w, http.StatusOK, utils.Envelope{
			"message":  "recharge already finalized",
			"recharge": mr,
		})
		return
	}

	apiResp, finalStatus, orderID, operatorTxnID := callMobileRechargeStatusAPI(mh.logger, mr.PartnerRequestID, mr.MobileRechargeTransactionID)

	if err = mh.rechargeStore.FinalizeMobileRecharge(mr.MobileRechargeTransactionID, operatorTxnID, orderID, finalStatus); err != nil {
		if err.Error() == "recharge not found or already finalized" {
			utils.BadRequest(w, mh.logger, "check mobile recharge status", err)
			return
		}
		utils.ServerError(w, mh.logger, "check mobile recharge status finalize", err)
		return
	}

	mr.RechargeStatus = finalStatus
	mr.OrderID = orderID
	mr.OperatorTransactionID = operatorTxnID

	msg := "recharge status updated"
	if finalStatus == "PENDING" {
		msg = "recharge still pending"
	}

	utils.WriteJSON(w, http.StatusOK, utils.Envelope{
		"message":      msg,
		"recharge":     mr,
		"api_response": apiResp,
	})
}

func callMobileRechargeStatusAPI(logger *slog.Logger, partnerRequestID string, id int64) (resp *models.APIResponseModel, finalStatus, orderID, operatorTxnID string) {
	finalStatus = "PENDING"

	if utils.RechargeKitAPI1 == "" || utils.RechargeKitAPIToken == "" {
		logger.Error("mobile recharge status api not configured", "id", id)
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
		logger.Error("mobile recharge status api call failed", "error", err, "id", id)
		return
	}

	resp = &apiResp
	orderID = apiResp.OrderID
	operatorTxnID = apiResp.OperatorTransactionID

	if apiResp.Error != 0 {
		logger.Error("mobile recharge status api error", "msg", apiResp.Message, "id", id)
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

func (mh *MobileRechargeHandler) HandleRefundMobileRecharge(w http.ResponseWriter, r *http.Request) {
	id, err := utils.ReadParamIDInt(r)
	if err != nil {
		utils.BadRequest(w, mh.logger, "refund mobile recharge", err)
		return
	}

	if err = mh.rechargeStore.RefundMobileRecharge(id); err != nil {
		if isRechargeClientErr(err) {
			utils.BadRequest(w, mh.logger, "refund mobile recharge", err)
			return
		}
		utils.ServerError(w, mh.logger, "refund mobile recharge", err)
		return
	}

	utils.WriteJSON(w, http.StatusOK, utils.Envelope{"message": "mobile recharge refunded"})
}

func (mh *MobileRechargeHandler) HandleGetAllMobileRecharge(w http.ResponseWriter, r *http.Request) {
	p := utils.ReadQueryParams(r)
	results, err := mh.rechargeStore.GetAllMobileRecharge(p)
	if err != nil {
		utils.ServerError(w, mh.logger, "get all mobile recharge", err)
		return
	}
	utils.WriteJSON(w, http.StatusOK, utils.Envelope{"message": "mobile recharges fetched", "mobile_recharges": results})
}

func (mh *MobileRechargeHandler) HandleGetMobileRechargeByRetailerID(w http.ResponseWriter, r *http.Request) {
	id, err := utils.ReadParamID(r)
	if err != nil {
		utils.BadRequest(w, mh.logger, "get mobile recharge by retailer", err)
		return
	}
	p := utils.ReadQueryParams(r)
	results, err := mh.rechargeStore.GetMobileRechargeByRetailerID(id, p)
	if err != nil {
		utils.ServerError(w, mh.logger, "get mobile recharge by retailer", err)
		return
	}
	utils.WriteJSON(w, http.StatusOK, utils.Envelope{"message": "mobile recharges fetched", "mobile_recharges": results})
}

func (mh *MobileRechargeHandler) HandleGetMobileRechargeByDistributorID(w http.ResponseWriter, r *http.Request) {
	id, err := utils.ReadParamID(r)
	if err != nil {
		utils.BadRequest(w, mh.logger, "get mobile recharge by distributor", err)
		return
	}
	p := utils.ReadQueryParams(r)
	results, err := mh.rechargeStore.GetMobileRechargeByDistributorID(id, p)
	if err != nil {
		utils.ServerError(w, mh.logger, "get mobile recharge by distributor", err)
		return
	}
	utils.WriteJSON(w, http.StatusOK, utils.Envelope{"message": "mobile recharges fetched", "mobile_recharges": results})
}

func (mh *MobileRechargeHandler) HandleGetMobileRechargeByMasterDistributorID(w http.ResponseWriter, r *http.Request) {
	id, err := utils.ReadParamID(r)
	if err != nil {
		utils.BadRequest(w, mh.logger, "get mobile recharge by md", err)
		return
	}
	p := utils.ReadQueryParams(r)
	results, err := mh.rechargeStore.GetMobileRechargeByMasterDistributorID(id, p)
	if err != nil {
		utils.ServerError(w, mh.logger, "get mobile recharge by md", err)
		return
	}
	utils.WriteJSON(w, http.StatusOK, utils.Envelope{"message": "mobile recharges fetched", "mobile_recharges": results})
}

func (mh *MobileRechargeHandler) HandleCreateMobileRechargeCircle(w http.ResponseWriter, r *http.Request) {
	var circle models.MobileRechargeCircleModel
	if err := json.NewDecoder(r.Body).Decode(&circle); err != nil {
		utils.BadRequest(w, mh.logger, "create circle", err)
		return
	}
	if circle.CircleCode == 0 || circle.CircleName == "" {
		utils.BadRequest(w, mh.logger, "create circle", errors.New("circle_code and circle_name are required"))
		return
	}
	if err := mh.rechargeStore.CreateMobileRechargeCircle(circle); err != nil {
		utils.ServerError(w, mh.logger, "create circle", err)
		return
	}
	utils.WriteJSON(w, http.StatusCreated, utils.Envelope{"message": "circle created"})
}

func (mh *MobileRechargeHandler) HandleUpdateMobileRechargeCircle(w http.ResponseWriter, r *http.Request) {
	code, err := utils.ReadParamIDInt(r)
	if err != nil {
		utils.BadRequest(w, mh.logger, "update circle", err)
		return
	}
	var circle models.MobileRechargeCircleModel
	if err := json.NewDecoder(r.Body).Decode(&circle); err != nil {
		utils.BadRequest(w, mh.logger, "update circle", err)
		return
	}
	if circle.CircleName == "" {
		utils.BadRequest(w, mh.logger, "update circle", errors.New("circle_name is required"))
		return
	}
	circle.CircleCode = int(code)
	if err := mh.rechargeStore.UpdateMobileRechargeCircle(circle); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			utils.BadRequest(w, mh.logger, "update circle", errors.New("circle not found"))
			return
		}
		utils.ServerError(w, mh.logger, "update circle", err)
		return
	}
	utils.WriteJSON(w, http.StatusOK, utils.Envelope{"message": "circle updated"})
}

func (mh *MobileRechargeHandler) HandleDeleteMobileRechargeCircle(w http.ResponseWriter, r *http.Request) {
	code, err := utils.ReadParamIDInt(r)
	if err != nil {
		utils.BadRequest(w, mh.logger, "delete circle", err)
		return
	}
	if err := mh.rechargeStore.DeleteMobileRechargeCircle(int(code)); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			utils.BadRequest(w, mh.logger, "delete circle", errors.New("circle not found"))
			return
		}
		utils.ServerError(w, mh.logger, "delete circle", err)
		return
	}
	utils.WriteJSON(w, http.StatusOK, utils.Envelope{"message": "circle deleted"})
}

func (mh *MobileRechargeHandler) HandleGetMobileRechargeCircles(w http.ResponseWriter, r *http.Request) {
	circles, err := mh.rechargeStore.GetMobileRechargeCircles()
	if err != nil {
		utils.ServerError(w, mh.logger, "get circles", err)
		return
	}
	utils.WriteJSON(w, http.StatusOK, utils.Envelope{"message": "circles fetched", "circles": circles})
}

func (mh *MobileRechargeHandler) HandleCreateMobileRechargeOperator(w http.ResponseWriter, r *http.Request) {
	var op models.MobileRechargeOperatorModel
	if err := json.NewDecoder(r.Body).Decode(&op); err != nil {
		utils.BadRequest(w, mh.logger, "create operator", err)
		return
	}
	if op.OperatorCode == 0 || op.OperatorName == "" {
		utils.BadRequest(w, mh.logger, "create operator", errors.New("operator_code and operator_name are required"))
		return
	}
	if err := mh.rechargeStore.CreateMobileRechargeOperator(op); err != nil {
		utils.ServerError(w, mh.logger, "create operator", err)
		return
	}
	utils.WriteJSON(w, http.StatusCreated, utils.Envelope{"message": "operator created"})
}

func (mh *MobileRechargeHandler) HandleUpdateMobileRechargeOperator(w http.ResponseWriter, r *http.Request) {
	code, err := utils.ReadParamIDInt(r)
	if err != nil {
		utils.BadRequest(w, mh.logger, "update operator", err)
		return
	}
	var op models.MobileRechargeOperatorModel
	if err := json.NewDecoder(r.Body).Decode(&op); err != nil {
		utils.BadRequest(w, mh.logger, "update operator", err)
		return
	}
	if op.OperatorName == "" {
		utils.BadRequest(w, mh.logger, "update operator", errors.New("operator_name is required"))
		return
	}
	op.OperatorCode = int(code)
	if err := mh.rechargeStore.UpdateMobileRechargeOperator(op); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			utils.BadRequest(w, mh.logger, "update operator", errors.New("operator not found"))
			return
		}
		utils.ServerError(w, mh.logger, "update operator", err)
		return
	}
	utils.WriteJSON(w, http.StatusOK, utils.Envelope{"message": "operator updated"})
}

func (mh *MobileRechargeHandler) HandleDeleteMobileRechargeOperator(w http.ResponseWriter, r *http.Request) {
	code, err := utils.ReadParamIDInt(r)
	if err != nil {
		utils.BadRequest(w, mh.logger, "delete operator", err)
		return
	}
	if err := mh.rechargeStore.DeleteMobileRechargeOperator(int(code)); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			utils.BadRequest(w, mh.logger, "delete operator", errors.New("operator not found"))
			return
		}
		utils.ServerError(w, mh.logger, "delete operator", err)
		return
	}
	utils.WriteJSON(w, http.StatusOK, utils.Envelope{"message": "operator deleted"})
}

func (mh *MobileRechargeHandler) HandleGetMobileRechargeOperators(w http.ResponseWriter, r *http.Request) {
	operators, err := mh.rechargeStore.GetMobileRechargeOperators()
	if err != nil {
		utils.ServerError(w, mh.logger, "get operators", err)
		return
	}
	utils.WriteJSON(w, http.StatusOK, utils.Envelope{"message": "operators fetched", "operators": operators})
}

func (mh *MobileRechargeHandler) HandleFetchPrepaidPlans(w http.ResponseWriter, r *http.Request) {
	operatorCode := r.URL.Query().Get("operator_code")
	circle := r.URL.Query().Get("circle")

	if operatorCode == "" || circle == "" {
		utils.BadRequest(w, mh.logger, "fetch prepaid plans", errors.New("operator_code and circle are required"))
		return
	}
	if utils.RechargeKitAPI1 == "" || utils.RechargeKitAPIToken == "" {
		utils.ServerError(w, mh.logger, "fetch prepaid plans", errors.New("recharge api not configured"))
		return
	}

	url := fmt.Sprintf("%s%s?circle=%s&operator_code=%s", utils.RechargeKitAPI1, utils.PrepaidPlanFetch, circle, operatorCode)
	var resp models.PrepaidPlanFetchResponseModel
	if err := utils.GetRequest(url, "Authorization", "Bearer "+utils.RechargeKitAPIToken, &resp); err != nil {
		utils.ServerError(w, mh.logger, "fetch prepaid plans", err)
		return
	}

	utils.WriteJSON(w, http.StatusOK, utils.Envelope{"plans": resp})
}

func (mh *MobileRechargeHandler) HandleFetchPostpaidBill(w http.ResponseWriter, r *http.Request) {
	var req models.PostpaidBillFetchRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		utils.BadRequest(w, mh.logger, "fetch postpaid bill", err)
		return
	}
	if req.MobileNumber == "" {
		utils.BadRequest(w, mh.logger, "fetch postpaid bill", errors.New("mobile_no is required"))
		return
	}
	if req.OperatorCode == 0 {
		utils.BadRequest(w, mh.logger, "fetch postpaid bill", errors.New("operator_code is required"))
		return
	}
	if utils.RechargeKitAPI1 == "" || utils.RechargeKitAPIToken == "" {
		utils.ServerError(w, mh.logger, "fetch postpaid bill", errors.New("recharge api not configured"))
		return
	}

	var resp models.PostpaidBillFetchResponse
	if err := utils.PostRequest(
		utils.RechargeKitAPI1+utils.PostpaidBillFetch,
		"Authorization",
		"Bearer "+utils.RechargeKitAPIToken,
		map[string]any{
			"mobile_no":     req.MobileNumber,
			"operator_code": req.OperatorCode,
		},
		&resp,
	); err != nil {
		utils.ServerError(w, mh.logger, "fetch postpaid bill", err)
		return
	}

	utils.WriteJSON(w, http.StatusOK, utils.Envelope{"bill": resp})
}

func isRechargeClientErr(err error) bool {
	msg := err.Error()
	return msg == "retailer not found" ||
		msg == "retailer KYC is not verified" ||
		msg == "retailer is blocked" ||
		msg == "insufficient wallet balance" ||
		msg == "operator not found" ||
		msg == "circle not found" ||
		msg == "recharge not found or already finalized" ||
		msg == "recharge not found or already refunded" ||
		msg == "only FAILED recharges can be refunded"
}
