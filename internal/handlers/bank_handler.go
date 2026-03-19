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

type BankHandler struct {
	bankStore store.BankStore
	logger    *slog.Logger
}

func NewBankHandler(bankStore store.BankStore, logger *slog.Logger) *BankHandler {
	return &BankHandler{bankStore: bankStore, logger: logger}
}

// --- bank handlers ---

func (bh *BankHandler) HandleCreateBank(w http.ResponseWriter, r *http.Request) {
	var bank models.BankModel
	if err := json.NewDecoder(r.Body).Decode(&bank); err != nil {
		utils.BadRequest(w, bh.logger, "create bank", err)
		return
	}

	if err := bank.Validate(); err != nil {
		utils.BadRequest(w, bh.logger, "create bank", err)
		return
	}

	if err := bh.bankStore.CreateBank(&bank); err != nil {
		utils.ServerError(w, bh.logger, "create bank", err)
		return
	}

	utils.WriteJSON(w, http.StatusCreated, utils.Envelope{"bank": bank})
}

func (bh *BankHandler) HandleUpdateBank(w http.ResponseWriter, r *http.Request) {
	bankID, err := readBankID(r)
	if err != nil {
		utils.BadRequest(w, bh.logger, "update bank", err)
		return
	}

	var bank models.BankModel
	if err := json.NewDecoder(r.Body).Decode(&bank); err != nil {
		utils.BadRequest(w, bh.logger, "update bank", err)
		return
	}

	if err := bh.bankStore.UpdateBank(bankID, &bank); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			utils.BadRequest(w, bh.logger, "update bank", errors.New("bank not found"))
			return
		}
		utils.ServerError(w, bh.logger, "update bank", err)
		return
	}

	utils.WriteJSON(w, http.StatusOK, utils.Envelope{"message": "bank updated successfully"})
}

func (bh *BankHandler) HandleDeleteBank(w http.ResponseWriter, r *http.Request) {
	bankID, err := readBankID(r)
	if err != nil {
		utils.BadRequest(w, bh.logger, "delete bank", err)
		return
	}

	if err := bh.bankStore.DeleteBank(bankID); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			utils.BadRequest(w, bh.logger, "delete bank", errors.New("bank not found"))
			return
		}
		utils.ServerError(w, bh.logger, "delete bank", err)
		return
	}

	utils.WriteJSON(w, http.StatusOK, utils.Envelope{"message": "bank deleted successfully"})
}

func (bh *BankHandler) HandleGetAllBanks(w http.ResponseWriter, r *http.Request) {
	banks, err := bh.bankStore.GetAllBanks()
	if err != nil {
		utils.ServerError(w, bh.logger, "get all banks", err)
		return
	}

	utils.WriteJSON(w, http.StatusOK, utils.Envelope{"banks": banks})
}

// --- admin bank handlers ---

func (bh *BankHandler) HandleCreateAdminBank(w http.ResponseWriter, r *http.Request) {
	var adminBank models.AdminBankModel
	if err := json.NewDecoder(r.Body).Decode(&adminBank); err != nil {
		utils.BadRequest(w, bh.logger, "create admin bank", err)
		return
	}

	if err := adminBank.Validate(); err != nil {
		utils.BadRequest(w, bh.logger, "create admin bank", err)
		return
	}

	if err := bh.bankStore.CreateAdminBank(&adminBank); err != nil {
		utils.ServerError(w, bh.logger, "create admin bank", err)
		return
	}

	utils.WriteJSON(w, http.StatusCreated, utils.Envelope{"admin_bank": adminBank})
}

func (bh *BankHandler) HandleUpdateAdminBank(w http.ResponseWriter, r *http.Request) {
	adminBankID, err := readAdminBankID(r)
	if err != nil {
		utils.BadRequest(w, bh.logger, "update admin bank", err)
		return
	}

	var adminBank models.AdminBankModel
	if err := json.NewDecoder(r.Body).Decode(&adminBank); err != nil {
		utils.BadRequest(w, bh.logger, "update admin bank", err)
		return
	}

	if err := bh.bankStore.UpdateAdminBank(adminBankID, &adminBank); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			utils.BadRequest(w, bh.logger, "update admin bank", errors.New("admin bank not found"))
			return
		}
		utils.ServerError(w, bh.logger, "update admin bank", err)
		return
	}

	utils.WriteJSON(w, http.StatusOK, utils.Envelope{"message": "admin bank updated successfully"})
}

func (bh *BankHandler) HandleDeleteAdminBank(w http.ResponseWriter, r *http.Request) {
	adminBankID, err := readAdminBankID(r)
	if err != nil {
		utils.BadRequest(w, bh.logger, "delete admin bank", err)
		return
	}

	if err := bh.bankStore.DeleteAdminBank(adminBankID); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			utils.BadRequest(w, bh.logger, "delete admin bank", errors.New("admin bank not found"))
			return
		}
		utils.ServerError(w, bh.logger, "delete admin bank", err)
		return
	}

	utils.WriteJSON(w, http.StatusOK, utils.Envelope{"message": "admin bank deleted successfully"})
}

func (bh *BankHandler) HandleGetAllAdminBanks(w http.ResponseWriter, r *http.Request) {
	banks, err := bh.bankStore.GetAllAdminBanks()
	if err != nil {
		utils.ServerError(w, bh.logger, "get all admin banks", err)
		return
	}

	utils.WriteJSON(w, http.StatusOK, utils.Envelope{"admin_banks": banks})
}

// --- helpers ---

func readBankID(r *http.Request) (int64, error) {
	idStr := chi.URLParam(r, "id")
	id, err := strconv.ParseInt(idStr, 10, 64)
	if err != nil {
		return 0, errors.New("invalid bank id")
	}
	return id, nil
}

func readAdminBankID(r *http.Request) (int64, error) {
	idStr := chi.URLParam(r, "id")
	id, err := strconv.ParseInt(idStr, 10, 64)
	if err != nil {
		return 0, errors.New("invalid admin bank id")
	}
	return id, nil
}
