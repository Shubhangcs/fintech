package handlers

import (
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"time"

	"github.com/levionstudio/fintech/internal/models"
	"github.com/levionstudio/fintech/internal/store"
	"github.com/levionstudio/fintech/internal/utils"
)

type AdminHandler struct {
	adminStore  store.AdminStore
	walletStore store.WalletTransactionStore
	logger      *slog.Logger
}

func NewAdminHandler(adminStore store.AdminStore, walletStore store.WalletTransactionStore, logger *slog.Logger) *AdminHandler {
	return &AdminHandler{
		adminStore:  adminStore,
		walletStore: walletStore,
		logger:      logger,
	}
}

// Create Admin Handler
func (ah *AdminHandler) HandleCreateAdmin(w http.ResponseWriter, r *http.Request) {
	var req models.AdminModel
	err := json.NewDecoder(r.Body).Decode(&req)
	if err != nil {
		utils.BadRequest(w, ah.logger, "create admin", err)
		return
	}

	err = req.ValidateCreateAdmin()
	if err != nil {
		utils.BadRequest(w, ah.logger, "create admin", err)
		return
	}

	err = ah.adminStore.CreateAdmin(&req)
	if err != nil {
		utils.ServerError(w, ah.logger, "create admin", err)
		return
	}
	utils.WriteJSON(w, http.StatusCreated, utils.Envelope{"message": "admin created successfully", "admin": req})
}

// Update Admin Details Handler
func (ah *AdminHandler) HandleUpdateAdminDetails(w http.ResponseWriter, r *http.Request) {
	id, err := utils.ReadParamID(r)
	if err != nil {
		utils.BadRequest(w, ah.logger, "update admin details", err)
		return
	}

	var req models.AdminModel
	err = json.NewDecoder(r.Body).Decode(&req)
	if err != nil {
		utils.BadRequest(w, ah.logger, "update admin details", err)
		return
	}

	req.AdminID = id
	err = ah.adminStore.UpdateAdminDetails(&req)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			utils.BadRequest(w, ah.logger, "update admin details", errors.New("admin not found"))
			return
		}
		utils.ServerError(w, ah.logger, "update admin details", err)
		return
	}

	utils.WriteJSON(w, http.StatusOK, utils.Envelope{"message": "admin details updated successfully"})
}

// Update admin Password Handler
func (ah *AdminHandler) HandleUpdateAdminPassword(w http.ResponseWriter, r *http.Request) {
	id, err := utils.ReadParamID(r)
	if err != nil {
		utils.BadRequest(w, ah.logger, "update admin password", err)
		return
	}

	var req models.AdminModel
	err = json.NewDecoder(r.Body).Decode(&req)
	if err != nil {
		utils.BadRequest(w, ah.logger, "update admin password", err)
		return
	}

	if req.AdminPassword == "" || !utils.IsValidPassword(req.AdminPassword) {
		utils.BadRequest(w, ah.logger, "update admin password", errors.New("invalid request format, admin password is empty or weak"))
		return
	}

	req.AdminID = id
	err = ah.adminStore.UpdateAdminPassword(&req)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			utils.BadRequest(w, ah.logger, "update admin password", errors.New("admin not found"))
			return
		}
		utils.ServerError(w, ah.logger, "update admin password", err)
		return
	}

	utils.WriteJSON(w, http.StatusOK, utils.Envelope{"message": "admin password updated successfully", "password": req.AdminPassword})
}

// Update Admin Wallet Balance Handler
func (ah *AdminHandler) HandleUpdateAdminWalletBalance(w http.ResponseWriter, r *http.Request) {
	id, err := utils.ReadParamID(r)
	if err != nil {
		utils.BadRequest(w, ah.logger, "update admin wallet balance", err)
		return
	}

	var req struct {
		Amount  float64 `json:"amount"`
		Remarks string  `json:"remarks"`
	}
	if err = json.NewDecoder(r.Body).Decode(&req); err != nil {
		utils.BadRequest(w, ah.logger, "update admin wallet balance", err)
		return
	}

	if req.Amount <= 0 {
		utils.BadRequest(w, ah.logger, "update admin wallet balance", errors.New("amount must be greater than 0"))
		return
	}

	admin, err := ah.adminStore.GetAdminByID(id)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			utils.BadRequest(w, ah.logger, "update admin wallet balance", errors.New("admin not found"))
			return
		}
		utils.ServerError(w, ah.logger, "update admin wallet balance", err)
		return
	}

	beforeBalance := admin.AdminWalletBalance
	afterBalance := beforeBalance + req.Amount

	admin.AdminWalletBalance = req.Amount
	if err = ah.adminStore.UpdateAdminWalletBalance(admin); err != nil {
		utils.ServerError(w, ah.logger, "update admin wallet balance", err)
		return
	}

	creditAmount := req.Amount
	wt := models.WalletTransactionModel{
		UserID:            id,
		ReferenceID:       "NO",
		CreditAmount:      &creditAmount,
		BeforeBalance:     beforeBalance,
		AfterBalance:      afterBalance,
		TransactionReason: "TOPUP",
		Remarks:           fmt.Sprintf("TOPUP on %s", time.Now().Format(utils.DateLayout)),
	}
	if err = ah.walletStore.CreateWalletTransaction(&wt); err != nil {
		utils.ServerError(w, ah.logger, "update admin wallet balance", err)
		return
	}

	utils.WriteJSON(w, http.StatusOK, utils.Envelope{"message": "admin wallet balance updated successfully", "admin_wallet_balance": afterBalance})
}

// Delete Admin Handler
func (ah *AdminHandler) HandleDeleteAdmin(w http.ResponseWriter, r *http.Request) {
	id, err := utils.ReadParamID(r)
	if err != nil {
		utils.BadRequest(w, ah.logger, "delete admin", err)
		return
	}

	err = ah.adminStore.DeleteAdmin(id)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			utils.BadRequest(w, ah.logger, "delete admin", errors.New("admin not found"))
			return
		}
		utils.ServerError(w, ah.logger, "delete admin", err)
		return
	}

	utils.WriteJSON(w, http.StatusOK, utils.Envelope{"message": "admin deleted successfully"})
}

// Get Admin By ID Handler
func (ah *AdminHandler) HandleGetAdminByID(w http.ResponseWriter, r *http.Request) {
	id, err := utils.ReadParamID(r)
	if err != nil {
		utils.BadRequest(w, ah.logger, "get admin by id", err)
		return
	}

	admin, err := ah.adminStore.GetAdminByID(id)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			utils.BadRequest(w, ah.logger, "get admin by id", errors.New("admin not found"))
			return
		}
		utils.ServerError(w, ah.logger, "get admin by id", err)
		return
	}

	utils.WriteJSON(w, http.StatusOK, utils.Envelope{"message": "admin details fetched successfully", "admin": admin})
}

// Get Admins For Dropdown Handler
func (ah *AdminHandler) HandleGetAdminsForDropdown(w http.ResponseWriter, r *http.Request) {
	admins, err := ah.adminStore.GetAdminsForDropdown()
	if err != nil {
		utils.ServerError(w, ah.logger, "get admins dropdown", err)
		return
	}

	utils.WriteJSON(w, http.StatusOK, utils.Envelope{"message": "admins fetched successfully", "admins": admins})
}

// Get Admins Handler
func (ah *AdminHandler) HandleGetAdmins(w http.ResponseWriter, r *http.Request) {
	p := utils.ReadPaginationParams(r)

	admins, err := ah.adminStore.GetAdmins(p.Limit, p.Offset)
	if err != nil {
		utils.ServerError(w, ah.logger, "get admins", err)
		return
	}

	utils.WriteJSON(w, http.StatusOK, utils.Envelope{"message": "admins fetched successfully", "admins": admins})
}

// Admin Login Handler
func (ah *AdminHandler) HandleAdminLogin(w http.ResponseWriter, r *http.Request) {
	var req models.AdminModel
	err := json.NewDecoder(r.Body).Decode(&req)
	if err != nil {
		utils.BadRequest(w, ah.logger, "admin login", err)
		return
	}

	if req.AdminID == "" || req.AdminPassword == "" {
		utils.BadRequest(w, ah.logger, "admin login", errors.New("id and password are required"))
		return
	}

	err = ah.adminStore.GetAdminDetailsForLogin(&req)
	if err != nil {
		utils.BadRequest(w, ah.logger, "admin login", err)
		return
	}

	token, err := utils.GenerateToken(req.AdminID, req.AdminName)
	if err != nil {
		utils.ServerError(w, ah.logger, "admin login", err)
		return
	}

	utils.WriteJSON(w, http.StatusOK, utils.Envelope{"message": "admin login successful", "token": token})
}

// Get Admin Wallet Balance Handler
func (ah *AdminHandler) HandleGetAdminWalletBalance(w http.ResponseWriter, r *http.Request) {
	id, err := utils.ReadParamID(r)
	if err != nil {
		utils.BadRequest(w, ah.logger, "get admin by id", err)
		return
	}
	balance, err := ah.adminStore.GetAdminWalletBalance(id)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			utils.BadRequest(w, ah.logger, "get admin wallet balance", errors.New("admin not found"))
			return
		}
		utils.ServerError(w, ah.logger, "get admin wallet balance", err)
		return
	}

	utils.WriteJSON(w, http.StatusOK, utils.Envelope{"message": "admin wallet balance fetched successfully", "balance": balance})
}
