package handlers

import (
	"encoding/json"
	"log/slog"
	"net/http"

	"github.com/levionstudio/fintech/internal/models"
	"github.com/levionstudio/fintech/internal/store"
	"github.com/levionstudio/fintech/internal/utils"
)

type WalletTransactionHandler struct {
	walletStore store.WalletTransactionStore
	logger      *slog.Logger
}

func NewWalletTransactionHandler(walletStore store.WalletTransactionStore, logger *slog.Logger) *WalletTransactionHandler {
	return &WalletTransactionHandler{
		walletStore: walletStore,
		logger:      logger,
	}
}

func (wh *WalletTransactionHandler) HandleCreateWalletTransaction(w http.ResponseWriter, r *http.Request) {
	var req models.WalletTransactionModel
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		utils.BadRequest(w, wh.logger, "create wallet transaction", err)
		return
	}

	if err := req.ValidateCreateWalletTransaction(); err != nil {
		utils.BadRequest(w, wh.logger, "create wallet transaction", err)
		return
	}

	if err := wh.walletStore.CreateWalletTransaction(&req); err != nil {
		utils.ServerError(w, wh.logger, "create wallet transaction", err)
		return
	}

	utils.WriteJSON(w, http.StatusCreated, utils.Envelope{"message": "wallet transaction created successfully", "wallet_transaction": req})
}

func (wh *WalletTransactionHandler) HandleGetWalletTransactionsByUserID(w http.ResponseWriter, r *http.Request) {
	userID, err := utils.ReadParamID(r)
	if err != nil {
		utils.BadRequest(w, wh.logger, "get wallet transactions", err)
		return
	}

	p := utils.ReadPaginationParams(r)
	startDate := utils.ParseDateParam(r, "start_date")
	endDate := utils.ParseDateParam(r, "end_date")

	transactions, err := wh.walletStore.GetWalletTransactionsByUserID(userID, p.Limit, p.Offset, startDate, endDate)
	if err != nil {
		utils.ServerError(w, wh.logger, "get wallet transactions", err)
		return
	}

	utils.WriteJSON(w, http.StatusOK, utils.Envelope{"message": "wallet transactions fetched successfully", "wallet_transactions": transactions})
}

