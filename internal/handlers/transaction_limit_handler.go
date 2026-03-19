package handlers

import (
	"database/sql"
	"encoding/json"
	"errors"
	"log/slog"
	"net/http"
	"strconv"

	"github.com/go-chi/chi/v5"
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

	utils.WriteJSON(w, http.StatusCreated, utils.Envelope{"transaction_limit": t})
}

func (th *TransactionLimitHandler) HandleUpdateTransactionLimit(w http.ResponseWriter, r *http.Request) {
	id, err := readLimitID(r)
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

	if err := th.transactionLimitStore.UpdateTransactionLimit(id, &t); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			utils.BadRequest(w, th.logger, "update transaction limit", errors.New("transaction limit not found"))
			return
		}
		utils.ServerError(w, th.logger, "update transaction limit", err)
		return
	}

	utils.WriteJSON(w, http.StatusOK, utils.Envelope{"message": "transaction limit updated successfully"})
}

func (th *TransactionLimitHandler) HandleDeleteTransactionLimit(w http.ResponseWriter, r *http.Request) {
	id, err := readLimitID(r)
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

func (th *TransactionLimitHandler) HandleGetAllTransactionLimits(w http.ResponseWriter, r *http.Request) {
	p := utils.ReadPaginationParams(r)

	limits, err := th.transactionLimitStore.GetAllTransactionLimits(p.Limit, p.Offset)
	if err != nil {
		utils.ServerError(w, th.logger, "get all transaction limits", err)
		return
	}

	utils.WriteJSON(w, http.StatusOK, utils.Envelope{"transaction_limits": limits})
}

func readLimitID(r *http.Request) (int64, error) {
	id, err := strconv.ParseInt(chi.URLParam(r, "id"), 10, 64)
	if err != nil {
		return 0, errors.New("invalid limit id")
	}
	return id, nil
}
