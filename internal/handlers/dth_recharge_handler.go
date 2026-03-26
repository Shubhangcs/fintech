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

type DTHRechargeHandler struct {
	rechargeStore store.DTHRechargeStore
	logger        *slog.Logger
}

func NewDTHRechargeHandler(rechargeStore store.DTHRechargeStore, logger *slog.Logger) *DTHRechargeHandler {
	return &DTHRechargeHandler{
		rechargeStore: rechargeStore,
		logger:        logger,
	}
}

func (dh *DTHRechargeHandler) HandleCreateDTHRecharge(w http.ResponseWriter, r *http.Request) {
	var dr models.DTHRechargeModel
	if err := json.NewDecoder(r.Body).Decode(&dr); err != nil {
		utils.BadRequest(w, dh.logger, "create dth recharge", err)
		return
	}

	if err := dr.Validate(); err != nil {
		utils.BadRequest(w, dh.logger, "create dth recharge", err)
		return
	}

	if len(dr.RetailerID) == 0 || dr.RetailerID[0] != 'R' {
		utils.BadRequest(w, dh.logger, "create dth recharge", errors.New("retailer_id must start with R"))
		return
	}

	if err := dh.rechargeStore.InitializeDTHRecharge(&dr); err != nil {
		if isDTHClientErr(err) {
			utils.BadRequest(w, dh.logger, "create dth recharge", err)
			return
		}
		utils.ServerError(w, dh.logger, "create dth recharge", err)
		return
	}

	apiResp, finalStatus := callDTHRechargeAPI(dh.logger, &dr)

	if err := dh.rechargeStore.FinalizeDTHRecharge(dr.DTHTransactionID, finalStatus); err != nil {
		dh.logger.Error("failed to finalize dth recharge", "error", err, "id", dr.DTHTransactionID)
	}

	dr.Status = finalStatus
	utils.WriteJSON(w, http.StatusCreated, utils.Envelope{
		"message":      "dth recharge processed",
		"recharge":     dr,
		"api_response": apiResp,
	})
}

func callDTHRechargeAPI(logger *slog.Logger, dr *models.DTHRechargeModel) (resp *models.PayoutAPIResponseModel, finalStatus string) {
	finalStatus = "FAILED"

	if utils.RechargeKitAPI1 == "" || utils.RechargeKitAPIToken == "" {
		logger.Error("dth recharge api not configured", "id", dr.DTHTransactionID)
		return
	}

	var apiResp models.PayoutAPIResponseModel
	err := utils.PostRequest(
		utils.RechargeKitAPI1+utils.DTHRecharge,
		"Authorization",
		"Bearer "+utils.RechargeKitAPIToken,
		map[string]any{
			"customer_id":        dr.CustomerID,
			"operator_code":      dr.OperatorCode,
			"amount":             dr.Amount,
			"partner_request_id": dr.PartnerRequestID,
		},
		&apiResp,
	)
	if err != nil {
		logger.Error("dth recharge api call failed", "error", err, "id", dr.DTHTransactionID)
		return
	}

	resp = &apiResp

	if apiResp.Error != 0 {
		logger.Error("dth recharge api error", "msg", apiResp.Message, "id", dr.DTHTransactionID)
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

func (dh *DTHRechargeHandler) HandleCheckDTHRechargeStatus(w http.ResponseWriter, r *http.Request) {
	id, err := utils.ReadParamIDInt(r)
	if err != nil {
		utils.BadRequest(w, dh.logger, "check dth recharge status", err)
		return
	}

	dr, err := dh.rechargeStore.GetDTHRechargeByID(id)
	if err != nil {
		if err.Error() == "dth recharge not found" {
			utils.BadRequest(w, dh.logger, "check dth recharge status", err)
			return
		}
		utils.ServerError(w, dh.logger, "check dth recharge status", err)
		return
	}

	if dr.Status != "PENDING" {
		utils.WriteJSON(w, http.StatusOK, utils.Envelope{
			"message":  "recharge already finalized",
			"recharge": dr,
		})
		return
	}

	apiResp, finalStatus := callDTHRechargeStatusAPI(dh.logger, dr.PartnerRequestID, dr.DTHTransactionID)

	if err = dh.rechargeStore.FinalizeDTHRecharge(dr.DTHTransactionID, finalStatus); err != nil {
		if err.Error() == "dth recharge not found or already finalized" {
			utils.BadRequest(w, dh.logger, "check dth recharge status", err)
			return
		}
		utils.ServerError(w, dh.logger, "check dth recharge status finalize", err)
		return
	}

	dr.Status = finalStatus

	msg := "recharge status updated"
	if finalStatus == "PENDING" {
		msg = "recharge still pending"
	}

	utils.WriteJSON(w, http.StatusOK, utils.Envelope{
		"message":      msg,
		"recharge":     dr,
		"api_response": apiResp,
	})
}

func callDTHRechargeStatusAPI(logger *slog.Logger, partnerRequestID string, id int64) (resp *models.PayoutAPIResponseModel, finalStatus string) {
	finalStatus = "PENDING"

	if utils.RechargeKitAPI1 == "" || utils.RechargeKitAPIToken == "" {
		logger.Error("dth recharge status api not configured", "id", id)
		return
	}

	var apiResp models.PayoutAPIResponseModel
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
		logger.Error("dth recharge status api call failed", "error", err, "id", id)
		return
	}

	resp = &apiResp

	if apiResp.Error != 0 {
		logger.Error("dth recharge status api error", "msg", apiResp.Message, "id", id)
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

func (dh *DTHRechargeHandler) HandleGetAllDTHRecharge(w http.ResponseWriter, r *http.Request) {
	p := utils.ReadQueryParams(r)
	results, err := dh.rechargeStore.GetAllDTHRecharge(p)
	if err != nil {
		utils.ServerError(w, dh.logger, "get all dth recharge", err)
		return
	}
	utils.WriteJSON(w, http.StatusOK, utils.Envelope{"message": "dth recharges fetched", "dth_recharges": results})
}

func (dh *DTHRechargeHandler) HandleGetDTHRechargeByRetailerID(w http.ResponseWriter, r *http.Request) {
	id, err := utils.ReadParamID(r)
	if err != nil {
		utils.BadRequest(w, dh.logger, "get dth recharge by retailer", err)
		return
	}
	p := utils.ReadQueryParams(r)
	results, err := dh.rechargeStore.GetDTHRechargeByRetailerID(id, p)
	if err != nil {
		utils.ServerError(w, dh.logger, "get dth recharge by retailer", err)
		return
	}
	utils.WriteJSON(w, http.StatusOK, utils.Envelope{"message": "dth recharges fetched", "dth_recharges": results})
}

func (dh *DTHRechargeHandler) HandleGetDTHRechargeByDistributorID(w http.ResponseWriter, r *http.Request) {
	id, err := utils.ReadParamID(r)
	if err != nil {
		utils.BadRequest(w, dh.logger, "get dth recharge by distributor", err)
		return
	}
	p := utils.ReadQueryParams(r)
	results, err := dh.rechargeStore.GetDTHRechargeByDistributorID(id, p)
	if err != nil {
		utils.ServerError(w, dh.logger, "get dth recharge by distributor", err)
		return
	}
	utils.WriteJSON(w, http.StatusOK, utils.Envelope{"message": "dth recharges fetched", "dth_recharges": results})
}

func (dh *DTHRechargeHandler) HandleGetDTHRechargeByMasterDistributorID(w http.ResponseWriter, r *http.Request) {
	id, err := utils.ReadParamID(r)
	if err != nil {
		utils.BadRequest(w, dh.logger, "get dth recharge by md", err)
		return
	}
	p := utils.ReadQueryParams(r)
	results, err := dh.rechargeStore.GetDTHRechargeByMasterDistributorID(id, p)
	if err != nil {
		utils.ServerError(w, dh.logger, "get dth recharge by md", err)
		return
	}
	utils.WriteJSON(w, http.StatusOK, utils.Envelope{"message": "dth recharges fetched", "dth_recharges": results})
}

func (dh *DTHRechargeHandler) HandleCreateDTHRechargeOperator(w http.ResponseWriter, r *http.Request) {
	var op models.DTHRechargeOperatorModel
	if err := json.NewDecoder(r.Body).Decode(&op); err != nil {
		utils.BadRequest(w, dh.logger, "create dth operator", err)
		return
	}
	if op.OperatorCode == 0 || op.OperatorName == "" {
		utils.BadRequest(w, dh.logger, "create dth operator", errors.New("operator_code and operator_name are required"))
		return
	}
	if err := dh.rechargeStore.CreateDTHRechargeOperator(op); err != nil {
		utils.ServerError(w, dh.logger, "create dth operator", err)
		return
	}
	utils.WriteJSON(w, http.StatusCreated, utils.Envelope{"message": "operator created"})
}

func (dh *DTHRechargeHandler) HandleUpdateDTHRechargeOperator(w http.ResponseWriter, r *http.Request) {
	code, err := utils.ReadParamIDInt(r)
	if err != nil {
		utils.BadRequest(w, dh.logger, "update dth operator", err)
		return
	}
	var op models.DTHRechargeOperatorModel
	if err := json.NewDecoder(r.Body).Decode(&op); err != nil {
		utils.BadRequest(w, dh.logger, "update dth operator", err)
		return
	}
	if op.OperatorName == "" {
		utils.BadRequest(w, dh.logger, "update dth operator", errors.New("operator_name is required"))
		return
	}
	op.OperatorCode = int(code)
	if err := dh.rechargeStore.UpdateDTHRechargeOperator(op); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			utils.BadRequest(w, dh.logger, "update dth operator", errors.New("operator not found"))
			return
		}
		utils.ServerError(w, dh.logger, "update dth operator", err)
		return
	}
	utils.WriteJSON(w, http.StatusOK, utils.Envelope{"message": "operator updated"})
}

func (dh *DTHRechargeHandler) HandleDeleteDTHRechargeOperator(w http.ResponseWriter, r *http.Request) {
	code, err := utils.ReadParamIDInt(r)
	if err != nil {
		utils.BadRequest(w, dh.logger, "delete dth operator", err)
		return
	}
	if err := dh.rechargeStore.DeleteDTHRechargeOperator(int(code)); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			utils.BadRequest(w, dh.logger, "delete dth operator", errors.New("operator not found"))
			return
		}
		utils.ServerError(w, dh.logger, "delete dth operator", err)
		return
	}
	utils.WriteJSON(w, http.StatusOK, utils.Envelope{"message": "operator deleted"})
}

func (dh *DTHRechargeHandler) HandleGetDTHRechargeOperators(w http.ResponseWriter, r *http.Request) {
	operators, err := dh.rechargeStore.GetDTHRechargeOperators()
	if err != nil {
		utils.ServerError(w, dh.logger, "get dth operators", err)
		return
	}
	utils.WriteJSON(w, http.StatusOK, utils.Envelope{"message": "operators fetched", "operators": operators})
}

func isDTHClientErr(err error) bool {
	msg := err.Error()
	return msg == "retailer not found" ||
		msg == "retailer KYC is not verified" ||
		msg == "retailer is blocked" ||
		msg == "insufficient wallet balance" ||
		msg == "operator not found" ||
		msg == "dth recharge not found or already finalized"
}
