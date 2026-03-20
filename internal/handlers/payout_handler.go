package handlers

import (
	"database/sql"
	"encoding/json"
	"errors"
	"log/slog"
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/levionstudio/fintech/internal/models"
	"github.com/levionstudio/fintech/internal/store"
	"github.com/levionstudio/fintech/internal/utils"
)

const paysprintPayoutPath = "/api/v1/payout/transfer"

// paysprintPayoutResponse is the response from Paysprint's payout API.
type paysprintPayoutResponse struct {
	Status       bool   `json:"status"`
	ResponseCode int    `json:"response_code"`
	Message      string `json:"message"`
	TxnID        string `json:"txnid"`
	OrderID      string `json:"orderid"`
	UTR          string `json:"utr"`
}

type PayoutHandler struct {
	payoutStore store.PayoutStore
	logger      *slog.Logger
}

func NewPayoutHandler(payoutStore store.PayoutStore, logger *slog.Logger) *PayoutHandler {
	return &PayoutHandler{payoutStore: payoutStore, logger: logger}
}

func (ph *PayoutHandler) HandleCreatePayoutTransaction(w http.ResponseWriter, r *http.Request) {
	var req models.CreatePayoutRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		utils.BadRequest(w, ph.logger, "create payout", err)
		return
	}
	if err := req.Validate(); err != nil {
		utils.BadRequest(w, ph.logger, "create payout", err)
		return
	}

	// Look up commision before initiating — so we know the total cost upfront
	commision, err := ph.payoutStore.GetPayoutCommision(req.RetailerID, req.Amount)
	if err != nil {
		utils.ServerError(w, ph.logger, "create payout: get commision", err)
		return
	}

	// Phase 1: debit retailer + write PENDING record atomically
	reqID := utils.GenerateReqID()
	transactionID, err := ph.payoutStore.InitiatePayoutTransaction(&req, reqID, commision)
	if err != nil {
		utils.BadRequest(w, ph.logger, "create payout: initiate", err)
		return
	}

	// Phase 2: call Paysprint API (PENDING record already committed)
	token, err := utils.GeneratePaysprintToken(reqID)
	if err != nil {
		// API token generation failed — record stays PENDING for manual recovery
		ph.logger.Error("create payout: generate token", "error", err, "transaction_id", transactionID)
		utils.WriteJSON(w, http.StatusAccepted, utils.Envelope{
			"message":        "payout is pending",
			"transaction_id": transactionID,
		})
		return
	}

	var apiResp paysprintPayoutResponse
	err = utils.PostRequest(utils.PaysprintAPI+paysprintPayoutPath, "Token", token, map[string]any{
		"refid":        reqID,
		"mobile":       req.MobileNumber,
		"bankname":     req.BankName,
		"bene_name":    req.BeneficiaryName,
		"accno":        req.AccountNumber,
		"ifsccode":     req.IFSCCode,
		"amount":       req.Amount,
		"transfertype": req.TransferType,
	}, &apiResp)
	if err != nil {
		// Network/decode error — record stays PENDING for manual recovery
		ph.logger.Error("create payout: paysprint call", "error", err, "transaction_id", transactionID)
		utils.WriteJSON(w, http.StatusAccepted, utils.Envelope{
			"message":        "payout is pending",
			"transaction_id": transactionID,
		})
		return
	}

	// Determine outcome from API response
	// response_code 1 = success, 2 = pending, others = failure
	switch apiResp.ResponseCode {
	case 1: // SUCCESS
		if finalErr := ph.payoutStore.FinalizePayoutTransaction(transactionID, apiResp.OrderID, apiResp.TxnID, "SUCCESS", commision, req.RetailerID); finalErr != nil {
			ph.logger.Error("create payout: finalize success", "error", finalErr, "transaction_id", transactionID)
		}
		utils.WriteJSON(w, http.StatusOK, utils.Envelope{
			"message":        "payout successful",
			"transaction_id": transactionID,
			"utr":            apiResp.UTR,
		})

	case 2: // PENDING
		if finalErr := ph.payoutStore.FinalizePayoutTransaction(transactionID, apiResp.OrderID, apiResp.TxnID, "PENDING", commision, req.RetailerID); finalErr != nil {
			ph.logger.Error("create payout: finalize pending", "error", finalErr, "transaction_id", transactionID)
		}
		utils.WriteJSON(w, http.StatusAccepted, utils.Envelope{
			"message":        "payout is pending",
			"transaction_id": transactionID,
		})

	default: // FAILED
		if failErr := ph.payoutStore.FailPayoutTransaction(transactionID, apiResp.OrderID, apiResp.TxnID); failErr != nil {
			ph.logger.Error("create payout: fail transaction", "error", failErr, "transaction_id", transactionID)
		}
		utils.WriteJSON(w, http.StatusUnprocessableEntity, utils.Envelope{
			"message":        apiResp.Message,
			"transaction_id": transactionID,
		})
	}
}

func (ph *PayoutHandler) HandleGetAllPayoutTransactions(w http.ResponseWriter, r *http.Request) {
	p := utils.ReadPaginationParams(r)

	transactions, err := ph.payoutStore.GetAllPayoutTransactions(p.Limit, p.Offset)
	if err != nil {
		utils.ServerError(w, ph.logger, "get all payout transactions", err)
		return
	}

	utils.WriteJSON(w, http.StatusOK, utils.Envelope{"payout_transactions": transactions})
}

func (ph *PayoutHandler) HandleGetPayoutTransactionsByRetailerID(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	if id == "" {
		utils.BadRequest(w, ph.logger, "get payout transactions by retailer", errors.New("id is required"))
		return
	}

	p := utils.ReadPaginationParams(r)

	transactions, err := ph.payoutStore.GetPayoutTransactionsByRetailerID(id, p.Limit, p.Offset)
	if err != nil {
		utils.ServerError(w, ph.logger, "get payout transactions by retailer", err)
		return
	}

	utils.WriteJSON(w, http.StatusOK, utils.Envelope{"payout_transactions": transactions})
}

func (ph *PayoutHandler) HandlePayoutRefund(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	if id == "" {
		utils.BadRequest(w, ph.logger, "payout refund", errors.New("id is required"))
		return
	}

	if err := ph.payoutStore.PayoutRefund(id); err != nil {
		if errors.Is(err, sql.ErrNoRows) || err.Error() == "transaction not found" || err.Error() == "only successful transactions can be refunded" {
			utils.BadRequest(w, ph.logger, "payout refund", err)
			return
		}
		utils.ServerError(w, ph.logger, "payout refund", err)
		return
	}

	utils.WriteJSON(w, http.StatusOK, utils.Envelope{"message": "payout refunded successfully"})
}

func (ph *PayoutHandler) HandleUpdatePayoutTransaction(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	if id == "" {
		utils.BadRequest(w, ph.logger, "update payout transaction", errors.New("id is required"))
		return
	}

	var req models.UpdatePayoutTransactionRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		utils.BadRequest(w, ph.logger, "update payout transaction", err)
		return
	}

	if err := ph.payoutStore.UpdatePayoutTransaction(id, &req); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			utils.BadRequest(w, ph.logger, "update payout transaction", errors.New("transaction not found"))
			return
		}
		utils.ServerError(w, ph.logger, "update payout transaction", err)
		return
	}

	utils.WriteJSON(w, http.StatusOK, utils.Envelope{"message": "payout transaction updated successfully"})
}
