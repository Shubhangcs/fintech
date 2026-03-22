package handlers

import (
	"encoding/json"
	"errors"
	"log/slog"
	"net/http"

	"github.com/levionstudio/fintech/internal/models"
	"github.com/levionstudio/fintech/internal/store"
	"github.com/levionstudio/fintech/internal/utils"
)

type PayoutHandler struct {
	payoutStore store.PayoutTransactionStore
	logger      *slog.Logger
}

func NewPayoutHandler(payoutStore store.PayoutTransactionStore, logger *slog.Logger) *PayoutHandler {
	return &PayoutHandler{payoutStore: payoutStore, logger: logger}
}

func mapAPIStatus(status int) string {
	switch status {
	case 1:
		return "SUCCESS"
	case 2:
		return "PENDING"
	default:
		return "FAILED"
	}
}

func (ph *PayoutHandler) HandleCreatePayoutTransaction(w http.ResponseWriter, r *http.Request) {
	var req models.PayoutTransactionModel
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		utils.BadRequest(w, ph.logger, "create payout transaction", err)
		return
	}

	if err := req.ValidateInitilizePayout(); err != nil {
		utils.BadRequest(w, ph.logger, "create payout transaction", err)
		return
	}

	if len(req.RetailerID) == 0 || string(req.RetailerID[0]) != "R" {
		utils.BadRequest(w, ph.logger, "create payout transaction", errors.New("retailer_id must belong to a retailer"))
		return
	}

	if err := ph.payoutStore.InitializePayoutTransaction(&req); err != nil {
		if isPayoutClientErr(err) {
			utils.BadRequest(w, ph.logger, "create payout transaction", err)
			return
		}
		utils.ServerError(w, ph.logger, "create payout transaction", err)
		return
	}

	// Hit the external payout API and auto-finalize based on the response.
	apiResp, finalStatus, orderID, operatorTxnID := callPayoutAPI(ph.logger, &req)

	if err := ph.payoutStore.FinalizePayout(req.PayoutTransactionID, orderID, operatorTxnID, finalStatus); err != nil {
		utils.ServerError(w, ph.logger, "finalize payout transaction", err)
		return
	}

	req.PayoutTransactionStatus = finalStatus
	req.OrderID = orderID
	req.OperatorTransactionID = operatorTxnID

	utils.WriteJSON(w, http.StatusCreated, utils.Envelope{
		"message":            "payout transaction processed",
		"payout_transaction": req,
		"api_response":       apiResp,
	})
}

// callPayoutAPI sends the payout request to the external API and returns the response
// plus the resolved finalStatus, orderID, and operatorTxnID ready to pass to FinalizePayout.
//
// Status resolution:
//   - Network / parse failure         → FAILED, empty IDs
//   - API error (error != 0)          → FAILED, IDs from response (may be empty)
//   - API status 1                    → SUCCESS
//   - API status 2                    → PENDING
//   - API status 3 (or anything else) → FAILED
func callPayoutAPI(logger *slog.Logger, pt *models.PayoutTransactionModel) (resp *apiPayoutResponse, finalStatus, orderID, operatorTxnID string) {
	finalStatus = "FAILED"

	if utils.RechargeKitAPI1 == "" || utils.RechargeKitAPIToken == "" {
		logger.Error("payout api not configured", "payout_transaction_id", pt.PayoutTransactionID)
		return
	}

	var apiResp apiPayoutResponse
	err := utils.PostRequest(
		utils.RechargeKitAPI2+utils.Payout,
		"Authorization",
		"Bearer "+utils.RechargeKitAPIToken,
		map[string]any{
			"mobile":       pt.MobileNumber,
			"name":         pt.BeneficiaryName,
			"account":      pt.AccountNumber,
			"ifsc":         pt.IFSCCode,
			"bankname":     pt.BankName,
			"amount":       pt.Amount,
			"txntype":      pt.TransferType,
			"partnerreqid": pt.PartnerRequestID,
		},
		&apiResp,
	)
	if err != nil {
		logger.Error("payout api call failed", "error", err, "payout_transaction_id", pt.PayoutTransactionID)
		return
	}

	resp = &apiResp
	orderID = apiResp.OrderID
	operatorTxnID = apiResp.OperatorTransactionID

	if apiResp.Error != 0 {
		logger.Error("payout api error", "msg", apiResp.Message, "payout_transaction_id", pt.PayoutTransactionID)
		return // finalStatus stays FAILED, but we captured the IDs
	}

	finalStatus = mapAPIStatus(apiResp.Status)
	return
}

// HandleUpdatePayoutTransaction manually finalizes a payout — used for callbacks or
// manual status corrections when the API response was not received.
func (ph *PayoutHandler) HandleUpdatePayoutTransaction(w http.ResponseWriter, r *http.Request) {
	payoutID, err := utils.ReadParamID(r)
	if err != nil {
		utils.BadRequest(w, ph.logger, "update payout transaction", err)
		return
	}

	var body struct {
		OrderID               string `json:"order_id"`
		OperatorTransactionID string `json:"operator_transaction_id"`
		Status                string `json:"payout_transaction_status"`
	}
	if err = json.NewDecoder(r.Body).Decode(&body); err != nil {
		utils.BadRequest(w, ph.logger, "update payout transaction", err)
		return
	}
	if body.Status == "" {
		utils.BadRequest(w, ph.logger, "update payout transaction", errors.New("payout_transaction_status is required"))
		return
	}

	if err = ph.payoutStore.FinalizePayout(payoutID, body.OrderID, body.OperatorTransactionID, body.Status); err != nil {
		if isPayoutClientErr(err) {
			utils.BadRequest(w, ph.logger, "update payout transaction", err)
			return
		}
		utils.ServerError(w, ph.logger, "update payout transaction", err)
		return
	}

	utils.WriteJSON(w, http.StatusOK, utils.Envelope{"message": "payout transaction updated successfully"})
}

// --- GET handlers ---

func (ph *PayoutHandler) HandleGetAllPayoutTransactions(w http.ResponseWriter, r *http.Request) {
	results, err := ph.payoutStore.GetAllPayoutTransactions(utils.ReadQueryParams(r))
	if err != nil {
		utils.ServerError(w, ph.logger, "get all payout transactions", err)
		return
	}
	utils.WriteJSON(w, http.StatusOK, utils.Envelope{"message": "payout transactions fetched successfully", "payout_transactions": results})
}

func (ph *PayoutHandler) HandleGetPayoutTransactionsByRetailerID(w http.ResponseWriter, r *http.Request) {
	id, err := utils.ReadParamID(r)
	if err != nil {
		utils.BadRequest(w, ph.logger, "get payout transactions by retailer id", err)
		return
	}
	results, err := ph.payoutStore.GetPayoutTransactionsByRetailerID(id, utils.ReadQueryParams(r))
	if err != nil {
		utils.ServerError(w, ph.logger, "get payout transactions by retailer id", err)
		return
	}
	utils.WriteJSON(w, http.StatusOK, utils.Envelope{"message": "payout transactions fetched successfully", "payout_transactions": results})
}

func (ph *PayoutHandler) HandleGetPayoutTransactionsByDistributorID(w http.ResponseWriter, r *http.Request) {
	id, err := utils.ReadParamID(r)
	if err != nil {
		utils.BadRequest(w, ph.logger, "get payout transactions by distributor id", err)
		return
	}
	results, err := ph.payoutStore.GetPayoutTransactionsByDistributorID(id, utils.ReadQueryParams(r))
	if err != nil {
		utils.ServerError(w, ph.logger, "get payout transactions by distributor id", err)
		return
	}
	utils.WriteJSON(w, http.StatusOK, utils.Envelope{"message": "payout transactions fetched successfully", "payout_transactions": results})
}

func (ph *PayoutHandler) HandleGetPayoutTransactionsByMasterDistributorID(w http.ResponseWriter, r *http.Request) {
	id, err := utils.ReadParamID(r)
	if err != nil {
		utils.BadRequest(w, ph.logger, "get payout transactions by md id", err)
		return
	}
	results, err := ph.payoutStore.GetPayoutTransactionsByMasterDistributorID(id, utils.ReadQueryParams(r))
	if err != nil {
		utils.ServerError(w, ph.logger, "get payout transactions by md id", err)
		return
	}
	utils.WriteJSON(w, http.StatusOK, utils.Envelope{"message": "payout transactions fetched successfully", "payout_transactions": results})
}

func isPayoutClientErr(err error) bool {
	msg := err.Error()
	return msg == "retailer not found" ||
		msg == "retailer KYC is not verified" ||
		msg == "retailer is blocked" ||
		msg == "insufficient wallet balance" ||
		msg == "payout transaction not found or already finalized" ||
		msg == "invalid payout_transaction_status"
}
