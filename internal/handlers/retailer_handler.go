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

type RetailerHandler struct {
	retailerStore      store.RetailerStore
	loginActivityStore store.LoginActivityStore
	logger             *slog.Logger
	awss3              *utils.AWSS3
}

func NewRetailerHandler(retailerStore store.RetailerStore, loginActivityStore store.LoginActivityStore, logger *slog.Logger, awss3 *utils.AWSS3) *RetailerHandler {
	return &RetailerHandler{
		retailerStore:      retailerStore,
		loginActivityStore: loginActivityStore,
		logger:             logger,
		awss3:              awss3,
	}
}

// Create Retailer Handler
func (rh *RetailerHandler) HandleCreateRetailer(w http.ResponseWriter, r *http.Request) {
	var req models.RetailerModel
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		utils.BadRequest(w, rh.logger, "create retailer", err)
		return
	}

	if err := req.ValidateCreateRetailer(); err != nil {
		utils.BadRequest(w, rh.logger, "create retailer", err)
		return
	}

	if err := rh.retailerStore.CreateRetailer(&req); err != nil {
		utils.ServerError(w, rh.logger, "create retailer", err)
		return
	}

	utils.WriteJSON(w, http.StatusCreated, utils.Envelope{"message": "retailer created successfully", "retailer": req})
}

// Update Retailer Details Handler
func (rh *RetailerHandler) HandleUpdateRetailerDetails(w http.ResponseWriter, r *http.Request) {
	id, err := utils.ReadParamID(r)
	if err != nil {
		utils.BadRequest(w, rh.logger, "update retailer", err)
		return
	}

	var req models.RetailerModel
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		utils.BadRequest(w, rh.logger, "update retailer", err)
		return
	}

	req.RetailerID = id
	if err := rh.retailerStore.UpdateRetailerDetails(&req); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			utils.BadRequest(w, rh.logger, "update retailer", errors.New("retailer not found"))
			return
		}
		utils.ServerError(w, rh.logger, "update retailer", err)
		return
	}

	utils.WriteJSON(w, http.StatusOK, utils.Envelope{"message": "retailer updated successfully"})
}

// Update Retailer Password Handler
func (rh *RetailerHandler) HandleUpdateRetailerPassword(w http.ResponseWriter, r *http.Request) {
	id, err := utils.ReadParamID(r)
	if err != nil {
		utils.BadRequest(w, rh.logger, "update retailer password", err)
		return
	}

	var req models.RetailerModel
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		utils.BadRequest(w, rh.logger, "update retailer password", err)
		return
	}

	if req.RetailerPassword == "" || !utils.IsValidPassword(req.RetailerPassword) {
		utils.BadRequest(w, rh.logger, "update retailer password", errors.New("password is empty or does not meet strength requirements"))
		return
	}

	req.RetailerID = id
	if err := rh.retailerStore.UpdateRetailerPassword(&req); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			utils.BadRequest(w, rh.logger, "update retailer password", errors.New("retailer not found"))
			return
		}
		utils.ServerError(w, rh.logger, "update retailer password", err)
		return
	}

	utils.WriteJSON(w, http.StatusOK, utils.Envelope{"message": "retailer password updated successfully", "password": req.RetailerPassword})
}

// Update Retailer MPIN Handler
func (rh *RetailerHandler) HandleUpdateRetailerMpin(w http.ResponseWriter, r *http.Request) {
	id, err := utils.ReadParamID(r)
	if err != nil {
		utils.BadRequest(w, rh.logger, "update retailer mpin", err)
		return
	}

	var req models.RetailerModel
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		utils.BadRequest(w, rh.logger, "update retailer mpin", err)
		return
	}

	if req.RetailerMpin < 1000 || req.RetailerMpin > 9999 {
		utils.BadRequest(w, rh.logger, "update retailer mpin", errors.New("mpin must be a 4-digit number"))
		return
	}

	req.RetailerID = id
	if err := rh.retailerStore.UpdateRetailerMpin(&req); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			utils.BadRequest(w, rh.logger, "update retailer mpin", errors.New("retailer not found"))
			return
		}
		utils.ServerError(w, rh.logger, "update retailer mpin", err)
		return
	}

	utils.WriteJSON(w, http.StatusOK, utils.Envelope{"message": "retailer mpin updated successfully", "mpin": req.RetailerMpin})
}

// Update Retailer KYC Status Handler
func (rh *RetailerHandler) HandleUpdateRetailerKYCStatus(w http.ResponseWriter, r *http.Request) {
	id, err := utils.ReadParamID(r)
	if err != nil {
		utils.BadRequest(w, rh.logger, "update retailer kyc status", err)
		return
	}

	var req models.RetailerModel
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		utils.BadRequest(w, rh.logger, "update retailer kyc status", err)
		return
	}

	req.RetailerID = id
	if err := rh.retailerStore.UpdateRetailerKYCStatus(&req); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			utils.BadRequest(w, rh.logger, "update retailer kyc status", errors.New("retailer not found"))
			return
		}
		utils.ServerError(w, rh.logger, "update retailer kyc status", err)
		return
	}

	utils.WriteJSON(w, http.StatusOK, utils.Envelope{"message": "retailer kyc status updated successfully"})
}

// Update Retailer Block Status Handler
func (rh *RetailerHandler) HandleUpdateRetailerBlockStatus(w http.ResponseWriter, r *http.Request) {
	id, err := utils.ReadParamID(r)
	if err != nil {
		utils.BadRequest(w, rh.logger, "update retailer block status", err)
		return
	}

	var req models.RetailerModel
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		utils.BadRequest(w, rh.logger, "update retailer block status", err)
		return
	}

	req.RetailerID = id
	if err := rh.retailerStore.UpdateRetailerBlockStatus(&req); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			utils.BadRequest(w, rh.logger, "update retailer block status", errors.New("retailer not found"))
			return
		}
		utils.ServerError(w, rh.logger, "update retailer block status", err)
		return
	}

	utils.WriteJSON(w, http.StatusOK, utils.Envelope{"message": "retailer block status updated successfully"})
}

// Get Retailer By ID Handler
func (rh *RetailerHandler) HandleGetRetailerByID(w http.ResponseWriter, r *http.Request) {
	id, err := utils.ReadParamID(r)
	if err != nil {
		utils.BadRequest(w, rh.logger, "get retailer by id", err)
		return
	}

	re, err := rh.retailerStore.GetRetailerByID(id)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			utils.BadRequest(w, rh.logger, "get retailer by id", errors.New("retailer not found"))
			return
		}
		utils.ServerError(w, rh.logger, "get retailer by id", err)
		return
	}

	utils.WriteJSON(w, http.StatusOK, utils.Envelope{"message": "retailer details fetched successfully", "retailer": re})
}

// Get Retailers By Distributor ID Handler
func (rh *RetailerHandler) HandleGetRetailersByDistributorID(w http.ResponseWriter, r *http.Request) {
	id, err := utils.ReadParamID(r)
	if err != nil {
		utils.BadRequest(w, rh.logger, "get retailers by distributor id", err)
		return
	}

	p := utils.ReadPaginationParams(r)

	retailers, err := rh.retailerStore.GetRetailersByDistributorID(id, p.Limit, p.Offset)
	if err != nil {
		utils.ServerError(w, rh.logger, "get retailers by distributor id", err)
		return
	}

	utils.WriteJSON(w, http.StatusOK, utils.Envelope{"message": "retailers fetched successfully", "retailers": retailers})
}

// Get Retailers By Master Distributor ID Handler
func (rh *RetailerHandler) HandleGetRetailersByMasterDistributorID(w http.ResponseWriter, r *http.Request) {
	id, err := utils.ReadParamID(r)
	if err != nil {
		utils.BadRequest(w, rh.logger, "get retailers by master distributor id", err)
		return
	}

	p := utils.ReadPaginationParams(r)

	retailers, err := rh.retailerStore.GetRetailersByMasterDistributorID(id, p.Limit, p.Offset)
	if err != nil {
		utils.ServerError(w, rh.logger, "get retailers by master distributor id", err)
		return
	}

	utils.WriteJSON(w, http.StatusOK, utils.Envelope{"message": "retailers fetched successfully", "retailers": retailers})
}

// Get Retailers By Admin ID Handler
func (rh *RetailerHandler) HandleGetRetailersByAdminID(w http.ResponseWriter, r *http.Request) {
	id, err := utils.ReadParamID(r)
	if err != nil {
		utils.BadRequest(w, rh.logger, "get retailers by admin id", err)
		return
	}

	p := utils.ReadPaginationParams(r)

	retailers, err := rh.retailerStore.GetRetailersByAdminID(id, p.Limit, p.Offset)
	if err != nil {
		utils.ServerError(w, rh.logger, "get retailers by admin id", err)
		return
	}

	utils.WriteJSON(w, http.StatusOK, utils.Envelope{"message": "retailers fetched successfully", "retailers": retailers})
}

// Retailer Login Handler
func (rh *RetailerHandler) HandleRetailerLogin(w http.ResponseWriter, r *http.Request) {
	var req struct {
		models.RetailerModel
		models.LoginDeviceInfo
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		utils.BadRequest(w, rh.logger, "retailer login", err)
		return
	}

	if req.RetailerID == "" || req.RetailerPassword == "" {
		utils.BadRequest(w, rh.logger, "retailer login", errors.New("id and password are required"))
		return
	}

	if err := rh.retailerStore.GetRetailerDetailsForLogin(&req.RetailerModel); err != nil {
		utils.BadRequest(w, rh.logger, "retailer login", err)
		return
	}

	token, err := utils.GenerateToken(req.RetailerID, req.RetailerName)
	if err != nil {
		utils.ServerError(w, rh.logger, "retailer login", err)
		return
	}

	rh.logger.Info("retailer login",
		"user_id", req.RetailerID,
		"user_agent", req.UserAgent,
		"platform", req.Platform,
		"latitude", req.Latitude,
		"longitude", req.Longitude,
		"timestamp", req.Timestamp,
	)
	go func() {
		if err := rh.loginActivityStore.CreateLoginActivity(models.LoginActivity{
			UserID: req.RetailerID, UserAgent: req.UserAgent, Platform: req.Platform,
			Latitude: req.Latitude, Longitude: req.Longitude, Accuracy: req.Accuracy,
			LoginTimestamp: req.Timestamp,
		}); err != nil {
			rh.logger.Error("failed to save login activity", "error", err, "user_id", req.RetailerID)
		}
	}()

	utils.WriteJSON(w, http.StatusOK, utils.Envelope{"message": "retailer login successful", "token": token})
}

// Get Retailers By Distributor ID For Dropdown Handler
func (rh *RetailerHandler) HandleGetRetailersByDistributorIDForDropdown(w http.ResponseWriter, r *http.Request) {
	id, err := utils.ReadParamID(r)
	if err != nil {
		utils.BadRequest(w, rh.logger, "get retailers dropdown by distributor id", err)
		return
	}

	retailers, err := rh.retailerStore.GetRetailersByDistributorIDForDropdown(id)
	if err != nil {
		utils.ServerError(w, rh.logger, "get retailers dropdown by distributor id", err)
		return
	}

	utils.WriteJSON(w, http.StatusOK, utils.Envelope{"message": "retailers fetched successfully", "retailers": retailers})
}

// Get Retailers By Master Distributor ID For Dropdown Handler
func (rh *RetailerHandler) HandleGetRetailersByMasterDistributorIDForDropdown(w http.ResponseWriter, r *http.Request) {
	id, err := utils.ReadParamID(r)
	if err != nil {
		utils.BadRequest(w, rh.logger, "get retailers dropdown by md id", err)
		return
	}

	retailers, err := rh.retailerStore.GetRetailersByMasterDistributorIDForDropdown(id)
	if err != nil {
		utils.ServerError(w, rh.logger, "get retailers dropdown by md id", err)
		return
	}

	utils.WriteJSON(w, http.StatusOK, utils.Envelope{"message": "retailers fetched successfully", "retailers": retailers})
}

// Get Retailers By Admin ID For Dropdown Handler
func (rh *RetailerHandler) HandleGetRetailersByAdminIDForDropdown(w http.ResponseWriter, r *http.Request) {
	id, err := utils.ReadParamID(r)
	if err != nil {
		utils.BadRequest(w, rh.logger, "get retailers dropdown by admin id", err)
		return
	}

	retailers, err := rh.retailerStore.GetRetailersByAdminIDForDropdown(id)
	if err != nil {
		utils.ServerError(w, rh.logger, "get retailers dropdown by admin id", err)
		return
	}

	utils.WriteJSON(w, http.StatusOK, utils.Envelope{"message": "retailers fetched successfully", "retailers": retailers})
}

// Delete Retailer Handler
func (rh *RetailerHandler) HandleDeleteRetailer(w http.ResponseWriter, r *http.Request) {
	id, err := utils.ReadParamID(r)
	if err != nil {
		utils.BadRequest(w, rh.logger, "delete retailer", err)
		return
	}

	if err := rh.retailerStore.DeleteRetailer(id); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			utils.BadRequest(w, rh.logger, "delete retailer", errors.New("retailer not found"))
			return
		}
		utils.ServerError(w, rh.logger, "delete retailer", err)
		return
	}

	utils.WriteJSON(w, http.StatusOK, utils.Envelope{"message": "retailer deleted successfully"})
}

// Change Retailers Distributor
func (rh *RetailerHandler) HandleChangeRetailersDistributor(w http.ResponseWriter, r *http.Request) {
	id, err := utils.ReadParamID(r)
	if err != nil {
		utils.BadRequest(w, rh.logger, "update retailer distributor", err)
		return
	}

	var req struct {
		DistributorID string `json:"distributor_id"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		utils.BadRequest(w, rh.logger, "update retailer distributor", err)
		return
	}
	if req.DistributorID == "" {
		utils.BadRequest(w, rh.logger, "update retailer distributor", errors.New("distributor_id is required"))
		return
	}

	if err := rh.retailerStore.ChangeRetailersDistributor(id, req.DistributorID); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			utils.BadRequest(w, rh.logger, "update retailer distributor", errors.New("retailer not found"))
			return
		}
		utils.ServerError(w, rh.logger, "update retailer distributor", err)
		return
	}

	utils.WriteJSON(w, http.StatusOK, utils.Envelope{"message": "retailer distributor updated successfully"})
}

// Update Retailer Aadhar Image
func (rh *RetailerHandler) HandleUpdateRetailerAadharImage(w http.ResponseWriter, r *http.Request) {
	id, err := utils.ReadParamID(r)
	if err != nil {
		utils.BadRequest(w, rh.logger, "update retailer aadhar image", err)
		return
	}
	path := fmt.Sprintf("documents/%s/%s_aadhar_%d.png", id, id, time.Now().Unix())
	url, err := rh.awss3.GenerateUploadPresignedURL(path)
	if err != nil {
		utils.ServerError(w, rh.logger, "update retailer aadhar image", err)
		return
	}
	err = rh.retailerStore.UpdateRetailerAadharImage(path, id)
	if err != nil {
		utils.ServerError(w, rh.logger, "update retailer aadhar image", err)
		return
	}

	utils.WriteJSON(w, http.StatusOK, utils.Envelope{"message": "retailer aadhar image upload url generated successfully", "url": url})
}

// Update Retailer Pan Image
func (rh *RetailerHandler) HandleUpdateRetailerPanImage(w http.ResponseWriter, r *http.Request) {
	id, err := utils.ReadParamID(r)
	if err != nil {
		utils.BadRequest(w, rh.logger, "update retailer pan image", err)
		return
	}
	path := fmt.Sprintf("documents/%s/%s_pan_%d.png", id, id, time.Now().Unix())
	url, err := rh.awss3.GenerateUploadPresignedURL(path)
	if err != nil {
		utils.ServerError(w, rh.logger, "update retailer pan image", err)
		return
	}
	err = rh.retailerStore.UpdateRetailerPanImage(path, id)
	if err != nil {
		utils.ServerError(w, rh.logger, "update retailer pan image", err)
		return
	}

	utils.WriteJSON(w, http.StatusOK, utils.Envelope{"message": "retailer pan image upload url generated successfully", "url": url})
}

// Update Retailer Image
func (rh *RetailerHandler) HandleUpdateRetailerImage(w http.ResponseWriter, r *http.Request) {
	id, err := utils.ReadParamID(r)
	if err != nil {
		utils.BadRequest(w, rh.logger, "update retailer image", err)
		return
	}
	path := fmt.Sprintf("documents/%s/%s_image_%d.png", id, id, time.Now().Unix())
	url, err := rh.awss3.GenerateUploadPresignedURL(path)
	if err != nil {
		utils.ServerError(w, rh.logger, "update retailer image", err)
		return
	}
	err = rh.retailerStore.UpdateRetailerImage(path, id)
	if err != nil {
		utils.ServerError(w, rh.logger, "update retailer image", err)
		return
	}

	utils.WriteJSON(w, http.StatusOK, utils.Envelope{"message": "retailer image upload url generated successfully", "url": url})
}

// Get Retailer Wallet Balance
func (rh *RetailerHandler) HandleGetRetailerWalletBalance(w http.ResponseWriter, r *http.Request) {
	id, err := utils.ReadParamID(r)
	if err != nil {
		utils.BadRequest(w, rh.logger, "get retailer wallet balance", err)
		return
	}
	balance, err := rh.retailerStore.GetRetailerWalletBalance(id)
	if err != nil {
		utils.ServerError(w, rh.logger, "get retailer wallet balance", err)
		return
	}

	utils.WriteJSON(w, http.StatusOK, utils.Envelope{"message": "retailer wallet balance fetched successfully", "balance": balance})
}
