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

type TransactionLimitHandler struct {
	transactionLimitStore store.TransactionLimitStore
	logger                *slog.Logger
}

func NewTransactionLimitHandler(transactionLimitStore store.TransactionLimitStore, logger *slog.Logger) *TransactionLimitHandler {
	return &TransactionLimitHandler{transactionLimitStore: transactionLimitStore, logger: logger}
}

// Create Transaction Limit Handler
func (th *TransactionLimitHandler) HandleCreateTransactionLimit(w http.ResponseWriter, r *http.Request) {
	var t models.TransactionLimitModel
	if err := json.NewDecoder(r.Body).Decode(&t); err != nil {
		utils.BadRequest(w, th.logger, "create transaction limit", err)
		return
	}

	if err := t.Validate(); err != nil {
		utils.BadRequest(w, th.logger, "create transaction limit", err)
		return
	}

	if err := th.transactionLimitStore.CreateTransactionLimit(&t); err != nil {
		utils.ServerError(w, th.logger, "create transaction limit", err)
		return
	}

	utils.WriteJSON(w, http.StatusCreated, utils.Envelope{"message": "transaction limit created successfully", "transaction_limit": t})
}

// Update Transaction Limit Handler
func (th *TransactionLimitHandler) HandleUpdateTransactionLimit(w http.ResponseWriter, r *http.Request) {
	id, err := utils.ReadParamIDInt(r)
	if err != nil {
		utils.BadRequest(w, th.logger, "update transaction limit", err)
		return
	}

	var t models.TransactionLimitModel
	if err := json.NewDecoder(r.Body).Decode(&t); err != nil {
		utils.BadRequest(w, th.logger, "update transaction limit", err)
		return
	}

	if t.LimitAmount < 0 {
		utils.BadRequest(w, th.logger, "update transaction limit", errors.New("limit_amount must be >= 0"))
		return
	}

	t.LimitID = id
	if err := th.transactionLimitStore.UpdateTransactionLimit(&t); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			utils.BadRequest(w, th.logger, "update transaction limit", errors.New("transaction limit not found"))
			return
		}
		utils.ServerError(w, th.logger, "update transaction limit", err)
		return
	}

	utils.WriteJSON(w, http.StatusOK, utils.Envelope{"message": "transaction limit updated successfully"})
}

// Delete Transaction Limit Handler
func (th *TransactionLimitHandler) HandleDeleteTransactionLimit(w http.ResponseWriter, r *http.Request) {
	id, err := utils.ReadParamIDInt(r)
	if err != nil {
		utils.BadRequest(w, th.logger, "delete transaction limit", err)
		return
	}

	if err := th.transactionLimitStore.DeleteTransactionLimit(id); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			utils.BadRequest(w, th.logger, "delete transaction limit", errors.New("transaction limit not found"))
			return
		}
		utils.ServerError(w, th.logger, "delete transaction limit", err)
		return
	}

	utils.WriteJSON(w, http.StatusOK, utils.Envelope{"message": "transaction limit deleted successfully"})
}

// Get All Transaction Limits Handler
func (th *TransactionLimitHandler) HandleGetAllTransactionLimits(w http.ResponseWriter, r *http.Request) {
	p := utils.ReadPaginationParams(r)

	limits, err := th.transactionLimitStore.GetAllTransactionLimits(p.Limit, p.Offset)
	if err != nil {
		utils.ServerError(w, th.logger, "get all transaction limits", err)
		return
	}

	utils.WriteJSON(w, http.StatusOK, utils.Envelope{"message": "transaction limits fetched successfully", "transaction_limits": limits})
}

// Get Transaction Limit By Retailer ID and Service Handler
func (th *TransactionLimitHandler) HandleGetTransactionLimitByRetailerIDAndService(w http.ResponseWriter, r *http.Request) {
	var req models.TransactionLimitModel
	err := json.NewDecoder(r.Body).Decode(&req)
	if err != nil {
		utils.BadRequest(w, th.logger, "get transaction limit by retailer id and service", err)
		return
	}

	if req.RetailerID == "" || req.Service == "" {
		utils.BadRequest(w, th.logger, "get transaction limit by retailer id and service", errors.New("invalid request format retailer id and service is required"))
		return
	}

	limit, isDefault, err := th.transactionLimitStore.GetTransactionLimitByRetailerIDAndService(&req)
	if err != nil {
		utils.ServerError(w, th.logger, "get transaction limit by retailer id and service", err)
		return
	}

	utils.WriteJSON(w, http.StatusOK, utils.Envelope{"message": "transaction limit fetched successfully", "limit": limit, "is_default": isDefault})
}
