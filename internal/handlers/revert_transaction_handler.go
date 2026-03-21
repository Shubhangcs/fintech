package handlers

import (
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"
	"strings"

	"github.com/levionstudio/fintech/internal/models"
	"github.com/levionstudio/fintech/internal/store"
	"github.com/levionstudio/fintech/internal/utils"
)

type RevertTransactionHandler struct {
	revertStore store.RevertTransactionStore
	logger      *slog.Logger
}

func NewRevertTransactionHandler(revertStore store.RevertTransactionStore, logger *slog.Logger) *RevertTransactionHandler {
	return &RevertTransactionHandler{revertStore: revertStore, logger: logger}
}

// --- create handlers ---

func (rh *RevertTransactionHandler) HandleAdminRevertOnMD(w http.ResponseWriter, r *http.Request) {
	rh.handleCreate(w, r, "admin revert on md", "A", "M")
}

func (rh *RevertTransactionHandler) HandleAdminRevertOnDistributor(w http.ResponseWriter, r *http.Request) {
	rh.handleCreate(w, r, "admin revert on distributor", "A", "D")
}

func (rh *RevertTransactionHandler) HandleAdminRevertOnRetailer(w http.ResponseWriter, r *http.Request) {
	rh.handleCreate(w, r, "admin revert on retailer", "A", "R")
}

func (rh *RevertTransactionHandler) HandleMDRevertOnDistributor(w http.ResponseWriter, r *http.Request) {
	rh.handleCreate(w, r, "md revert on distributor", "M", "D")
}

func (rh *RevertTransactionHandler) HandleMDRevertOnRetailer(w http.ResponseWriter, r *http.Request) {
	rh.handleCreate(w, r, "md revert on retailer", "M", "R")
}

func (rh *RevertTransactionHandler) HandleDistributorRevertOnRetailer(w http.ResponseWriter, r *http.Request) {
	rh.handleCreate(w, r, "distributor revert on retailer", "D", "R")
}

// --- get handlers ---

func (rh *RevertTransactionHandler) HandleGetRevertTransactionsByRevertByID(w http.ResponseWriter, r *http.Request) {
	id, err := utils.ReadParamID(r)
	if err != nil {
		utils.BadRequest(w, rh.logger, "get revert transactions by revert_by id", err)
		return
	}

	results, err := rh.revertStore.GetRevertTransactionsByRevertByID(id, utils.ReadQueryParams(r))
	if err != nil {
		utils.ServerError(w, rh.logger, "get revert transactions by revert_by id", err)
		return
	}

	utils.WriteJSON(w, http.StatusOK, utils.Envelope{"message": "revert transactions fetched successfully", "revert_transactions": results})
}

func (rh *RevertTransactionHandler) HandleGetRevertTransactionsByRevertOnID(w http.ResponseWriter, r *http.Request) {
	id, err := utils.ReadParamID(r)
	if err != nil {
		utils.BadRequest(w, rh.logger, "get revert transactions by revert_on id", err)
		return
	}

	results, err := rh.revertStore.GetRevertTransactionsByRevertOnID(id, utils.ReadQueryParams(r))
	if err != nil {
		utils.ServerError(w, rh.logger, "get revert transactions by revert_on id", err)
		return
	}

	utils.WriteJSON(w, http.StatusOK, utils.Envelope{"message": "revert transactions fetched successfully", "revert_transactions": results})
}

func (rh *RevertTransactionHandler) HandleGetAllRevertTransactions(w http.ResponseWriter, r *http.Request) {
	results, err := rh.revertStore.GetAllRevertTransactions(utils.ReadQueryParams(r))
	if err != nil {
		utils.ServerError(w, rh.logger, "get all revert transactions", err)
		return
	}

	utils.WriteJSON(w, http.StatusOK, utils.Envelope{"message": "revert transactions fetched successfully", "revert_transactions": results})
}

// --- private helpers ---

func (rh *RevertTransactionHandler) handleCreate(
	w http.ResponseWriter,
	r *http.Request,
	op, revertByPrefix, revertOnPrefix string,
) {
	var req models.RevertTransactionModel
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		utils.BadRequest(w, rh.logger, op, err)
		return
	}

	if err := req.Validate(); err != nil {
		utils.BadRequest(w, rh.logger, op, err)
		return
	}

	if len(req.RevertByID) == 0 || string(req.RevertByID[0]) != revertByPrefix {
		utils.BadRequest(w, rh.logger, op, fmt.Errorf("revert_by_id must belong to a %s user", prefixToRole(revertByPrefix)))
		return
	}

	if len(req.RevertOnID) == 0 || string(req.RevertOnID[0]) != revertOnPrefix {
		utils.BadRequest(w, rh.logger, op, fmt.Errorf("revert_on_id must belong to a %s user", prefixToRole(revertOnPrefix)))
		return
	}

	if err := rh.revertStore.CreateRevertTransaction(&req); err != nil {
		if isRevertClientErr(err) {
			utils.BadRequest(w, rh.logger, op, err)
			return
		}
		utils.ServerError(w, rh.logger, op, err)
		return
	}

	utils.WriteJSON(w, http.StatusCreated, utils.Envelope{"message": "revert transaction created successfully", "revert_transaction": req})
}

func isRevertClientErr(err error) bool {
	msg := err.Error()
	return msg == "revert_on user not found" ||
		msg == "revert_by user not found" ||
		msg == "insufficient balance" ||
		strings.HasPrefix(msg, "unknown user type for id:")
}
