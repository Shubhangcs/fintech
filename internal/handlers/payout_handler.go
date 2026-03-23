package handlers

import (
	"encoding/json"
	"errors"
	"fmt"
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
	case 3:
		return "FAILED"
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
func callPayoutAPI(logger *slog.Logger, pt *models.PayoutTransactionModel) (resp *models.PayoutAPIResponseModel, finalStatus, orderID, operatorTxnID string) {
	finalStatus = "PENDING"

	if utils.RechargeKitAPI2 == "" || utils.RechargeKitAPIToken == "" {
		logger.Error("payout api not configured", "payout_transaction_id", pt.PayoutTransactionID)
		return
	}

	var transactionType int
	var apiResp models.PayoutAPIResponseModel
	if pt.TransferType == "IMPS" {
		transactionType = 5
	} else {
		transactionType = 6
	}
	err := utils.PostRequest(
		utils.RechargeKitAPI2+utils.Payout,
		"Authorization",
		"Bearer "+utils.RechargeKitAPIToken,
		map[string]any{
			"mobile_no":          pt.MobileNumber,
			"beneficiary_name":   pt.BeneficiaryName,
			"account_no":         pt.AccountNumber,
			"ifsc":               pt.IFSCCode,
			"bank_name":          pt.BankName,
			"amount":             pt.Amount,
			"transfer_type":      transactionType,
			"partner_request_id": pt.PartnerRequestID,
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

	fmt.Println(apiResp)

	if apiResp.Error != 0 {
		logger.Error("payout api error", "msg", apiResp.Message, "payout_transaction_id", pt.PayoutTransactionID)
		return // finalStatus stays FAILED, but we captured the IDs
	}

	finalStatus = mapAPIStatus(apiResp.Status)
	return
}

// HandleCheckPayoutStatus fetches the current payout record, calls the operator status
// check API using partner_request_id, and auto-finalizes based on the response.
// Only PENDING transactions are re-checked; already finalized ones are returned as-is.
func (ph *PayoutHandler) HandleCheckPayoutStatus(w http.ResponseWriter, r *http.Request) {
	payoutID, err := utils.ReadParamID(r)
	if err != nil {
		utils.BadRequest(w, ph.logger, "check payout status", err)
		return
	}

	pt, err := ph.payoutStore.GetPayoutTransactionByID(payoutID)
	if err != nil {
		if err.Error() == "payout transaction not found" {
			utils.BadRequest(w, ph.logger, "check payout status", err)
			return
		}
		utils.ServerError(w, ph.logger, "check payout status", err)
		return
	}

	// If already finalized, return current record without calling the API
	if pt.PayoutTransactionStatus != "PENDING" {
		utils.WriteJSON(w, http.StatusOK, utils.Envelope{
			"message":            "payout already finalized",
			"payout_transaction": pt,
		})
		return
	}

	// Call status check API with partner_request_id
	apiResp, finalStatus, orderID, operatorTxnID := callPayoutStatusAPI(ph.logger, pt.PartnerRequestID, pt.PayoutTransactionID)

	if err = ph.payoutStore.FinalizePayout(pt.PayoutTransactionID, orderID, operatorTxnID, finalStatus); err != nil {
		utils.ServerError(w, ph.logger, "check payout status finalize", err)
		return
	}

	pt.PayoutTransactionStatus = finalStatus
	pt.OrderID = orderID
	pt.OperatorTransactionID = operatorTxnID

	utils.WriteJSON(w, http.StatusOK, utils.Envelope{
		"message":            "payout status updated",
		"payout_transaction": pt,
		"api_response":       apiResp,
	})
}

// callPayoutStatusAPI hits the operator status check endpoint and returns the resolved
// finalStatus, orderID, and operatorTxnID.
//
// Status resolution:
//   - Network / parse failure → PENDING (retry later)
//   - API error (error != 0)  → PENDING (retry later)
//   - status 1                → SUCCESS
//   - status 2                → PENDING
//   - status 3                → FAILED
//   - others (hold)           → PENDING
func callPayoutStatusAPI(logger *slog.Logger, partnerRequestID, payoutTransactionID string) (resp *models.PayoutAPIResponseModel, finalStatus, orderID, operatorTxnID string) {
	finalStatus = "PENDING"

	if utils.RechargeKitAPI2 == "" || utils.RechargeKitAPIToken == "" {
		logger.Error("payout status api not configured", "payout_transaction_id", payoutTransactionID)
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
		logger.Error("payout status api call failed", "error", err, "payout_transaction_id", payoutTransactionID)
		return
	}

	resp = &apiResp
	orderID = apiResp.OrderID
	operatorTxnID = apiResp.OperatorTransactionID

	if apiResp.Error != 0 {
		logger.Error("payout status api error", "msg", apiResp.Message, "payout_transaction_id", payoutTransactionID)
		return // stays PENDING
	}

	switch apiResp.Status {
	case 1:
		finalStatus = "SUCCESS"
	case 3:
		finalStatus = "FAILED"
	default:
		finalStatus = "PENDING" // status 2 or any hold code
	}
	return
}

// HandleRefundPayout reverses a FAILED payout: deducts commissions from each
// recipient wallet and credits the full amount back to the retailer.
func (ph *PayoutHandler) HandleRefundPayout(w http.ResponseWriter, r *http.Request) {
	payoutID, err := utils.ReadParamID(r)
	if err != nil {
		utils.BadRequest(w, ph.logger, "refund payout", err)
		return
	}

	if err = ph.payoutStore.RefundPayout(payoutID); err != nil {
		if isPayoutClientErr(err) {
			utils.BadRequest(w, ph.logger, "refund payout", err)
			return
		}
		utils.ServerError(w, ph.logger, "refund payout", err)
		return
	}

	utils.WriteJSON(w, http.StatusOK, utils.Envelope{"message": "payout refunded successfully"})
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
		msg == "transaction limit exceded" ||
		msg == "payout transaction not found or already finalized" ||
		msg == "payout transaction not found or already refunded" ||
		msg == "only FAILED payout transactions can be refunded" ||
		msg == "invalid payout_transaction_status"
}
