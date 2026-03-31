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

type DistributorHandler struct {
	distributorStore   store.DistributorStore
	loginActivityStore store.LoginActivityStore
	logger             *slog.Logger
	awss3              *utils.AWSS3
}

func NewDistributorHandler(distributorStore store.DistributorStore, loginActivityStore store.LoginActivityStore, logger *slog.Logger, awss3 *utils.AWSS3) *DistributorHandler {
	return &DistributorHandler{
		distributorStore:   distributorStore,
		loginActivityStore: loginActivityStore,
		logger:             logger,
		awss3:              awss3,
	}
}

// Create Distributor Handler
func (dh *DistributorHandler) HandleCreateDistributor(w http.ResponseWriter, r *http.Request) {
	var req models.DistributorModel
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		utils.BadRequest(w, dh.logger, "create distributor", err)
		return
	}

	if err := req.ValidateCreateDistributor(); err != nil {
		utils.BadRequest(w, dh.logger, "create distributor", err)
		return
	}

	if err := dh.distributorStore.CreateDistributor(&req); err != nil {
		utils.ServerError(w, dh.logger, "create distributor", err)
		return
	}

	utils.WriteJSON(w, http.StatusCreated, utils.Envelope{"message": "distributor created successfully", "distributor": req})
}

// Update Distributor Details Handler
func (dh *DistributorHandler) HandleUpdateDistributorDetails(w http.ResponseWriter, r *http.Request) {
	id, err := utils.ReadParamID(r)
	if err != nil {
		utils.BadRequest(w, dh.logger, "update distributor", err)
		return
	}

	var req models.DistributorModel
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		utils.BadRequest(w, dh.logger, "update distributor", err)
		return
	}

	req.DistributorID = id
	if err := dh.distributorStore.UpdateDistributorDetails(&req); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			utils.BadRequest(w, dh.logger, "update distributor", errors.New("distributor not found"))
			return
		}
		utils.ServerError(w, dh.logger, "update distributor", err)
		return
	}

	utils.WriteJSON(w, http.StatusOK, utils.Envelope{"message": "distributor updated successfully"})
}

// Update Distributor Password Handler
func (dh *DistributorHandler) HandleUpdateDistributorPassword(w http.ResponseWriter, r *http.Request) {
	id, err := utils.ReadParamID(r)
	if err != nil {
		utils.BadRequest(w, dh.logger, "update distributor password", err)
		return
	}

	var req models.DistributorModel
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		utils.BadRequest(w, dh.logger, "update distributor password", err)
		return
	}

	if req.DistributorPassword == "" || !utils.IsValidPassword(req.DistributorPassword) {
		utils.BadRequest(w, dh.logger, "update distributor password", errors.New("password is empty or does not meet strength requirements"))
		return
	}

	req.DistributorID = id
	if err := dh.distributorStore.UpdateDistributorPassword(&req); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			utils.BadRequest(w, dh.logger, "update distributor password", errors.New("distributor not found"))
			return
		}
		utils.ServerError(w, dh.logger, "update distributor password", err)
		return
	}

	utils.WriteJSON(w, http.StatusOK, utils.Envelope{"message": "distributor password updated successfully", "password": req.DistributorPassword})
}

// Update Distributor MPIN Handler
func (dh *DistributorHandler) HandleUpdateDistributorMpin(w http.ResponseWriter, r *http.Request) {
	id, err := utils.ReadParamID(r)
	if err != nil {
		utils.BadRequest(w, dh.logger, "update distributor mpin", err)
		return
	}

	var req models.DistributorModel
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		utils.BadRequest(w, dh.logger, "update distributor mpin", err)
		return
	}

	if req.DistributorMpin < 1000 || req.DistributorMpin > 9999 {
		utils.BadRequest(w, dh.logger, "update distributor mpin", errors.New("mpin must be a 4-digit number"))
		return
	}

	req.DistributorID = id
	if err := dh.distributorStore.UpdateDistributorMpin(&req); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			utils.BadRequest(w, dh.logger, "update distributor mpin", errors.New("distributor not found"))
			return
		}
		utils.ServerError(w, dh.logger, "update distributor mpin", err)
		return
	}

	utils.WriteJSON(w, http.StatusOK, utils.Envelope{"message": "distributor mpin updated successfully", "mpin": req.DistributorMpin})
}

// Update Distributor KYC Status Handler
func (dh *DistributorHandler) HandleUpdateDistributorKYCStatus(w http.ResponseWriter, r *http.Request) {
	id, err := utils.ReadParamID(r)
	if err != nil {
		utils.BadRequest(w, dh.logger, "update distributor kyc status", err)
		return
	}

	var req models.DistributorModel
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		utils.BadRequest(w, dh.logger, "update distributor kyc status", err)
		return
	}

	req.DistributorID = id
	if err := dh.distributorStore.UpdateDistributorKYCStatus(&req); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			utils.BadRequest(w, dh.logger, "update distributor kyc status", errors.New("distributor not found"))
			return
		}
		utils.ServerError(w, dh.logger, "update distributor kyc status", err)
		return
	}

	utils.WriteJSON(w, http.StatusOK, utils.Envelope{"message": "distributor kyc status updated successfully"})
}

// Update Distributor Block Status Handler
func (dh *DistributorHandler) HandleUpdateDistributorBlockStatus(w http.ResponseWriter, r *http.Request) {
	id, err := utils.ReadParamID(r)
	if err != nil {
		utils.BadRequest(w, dh.logger, "update distributor block status", err)
		return
	}

	var req models.DistributorModel
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		utils.BadRequest(w, dh.logger, "update distributor block status", err)
		return
	}

	req.DistributorID = id
	if err := dh.distributorStore.UpdateDistributorBlockStatus(&req); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			utils.BadRequest(w, dh.logger, "update distributor block status", errors.New("distributor not found"))
			return
		}
		utils.ServerError(w, dh.logger, "update distributor block status", err)
		return
	}

	utils.WriteJSON(w, http.StatusOK, utils.Envelope{"message": "distributor block status updated successfully"})
}

// Get Distributor By ID Handler
func (dh *DistributorHandler) HandleGetDistributorByID(w http.ResponseWriter, r *http.Request) {
	id, err := utils.ReadParamID(r)
	if err != nil {
		utils.BadRequest(w, dh.logger, "get distributor by id", err)
		return
	}

	d, err := dh.distributorStore.GetDistributorByID(id)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			utils.BadRequest(w, dh.logger, "get distributor by id", errors.New("distributor not found"))
			return
		}
		utils.ServerError(w, dh.logger, "get distributor by id", err)
		return
	}

	utils.WriteJSON(w, http.StatusOK, utils.Envelope{"message": "distributor details fetched successfully", "distributor": d})
}

// Get Distributors By Master Distributor ID For Dropdown Handler
func (dh *DistributorHandler) HandleGetDistributorsByMasterDistributorIDForDropdown(w http.ResponseWriter, r *http.Request) {
	mdID, err := utils.ReadParamID(r)
	if err != nil {
		utils.BadRequest(w, dh.logger, "get distributors dropdown by md id", err)
		return
	}

	distributors, err := dh.distributorStore.GetDistributorsByMasterDistributorIDForDropdown(mdID)
	if err != nil {
		utils.ServerError(w, dh.logger, "get distributors dropdown by md id", err)
		return
	}

	utils.WriteJSON(w, http.StatusOK, utils.Envelope{"message": "distributors fetched successfully", "distributors": distributors})
}

// Get Distributor By Admin ID Handler
func (dh *DistributorHandler) HandleGetDistributorsByAdminIDForDropdown(w http.ResponseWriter, r *http.Request) {
	adminID, err := utils.ReadParamID(r)
	if err != nil {
		utils.BadRequest(w, dh.logger, "get distributors dropdown by admin id", err)
		return
	}

	distributors, err := dh.distributorStore.GetDistributorsByAdminIDForDropdown(adminID)
	if err != nil {
		utils.ServerError(w, dh.logger, "get distributors dropdown by admin id", err)
		return
	}

	utils.WriteJSON(w, http.StatusOK, utils.Envelope{"message": "distributors fetched successfully", "distributors": distributors})
}

// Get Distributors By Master Distributor ID Handler
func (dh *DistributorHandler) HandleGetDistributorsByMasterDistributorID(w http.ResponseWriter, r *http.Request) {
	mdID, err := utils.ReadParamID(r)
	if err != nil {
		utils.BadRequest(w, dh.logger, "get distributors by master distributor id", err)
		return
	}

	p := utils.ReadPaginationParams(r)

	distributors, err := dh.distributorStore.GetDistributorsByMasterDistributorID(mdID, p.Limit, p.Offset)
	if err != nil {
		utils.ServerError(w, dh.logger, "get distributors by master distributor id", err)
		return
	}

	utils.WriteJSON(w, http.StatusOK, utils.Envelope{"message": "distributors fetched successfully", "distributors": distributors})
}

// Get Distributors By Admin ID Handler
func (dh *DistributorHandler) HandleGetDistributorsByAdminID(w http.ResponseWriter, r *http.Request) {
	adminID, err := utils.ReadParamID(r)
	if err != nil {
		utils.BadRequest(w, dh.logger, "get distributors by admin id", err)
		return
	}

	p := utils.ReadPaginationParams(r)

	distributors, err := dh.distributorStore.GetDistributorsByAdminID(adminID, p.Limit, p.Offset)
	if err != nil {
		utils.ServerError(w, dh.logger, "get distributors by admin id", err)
		return
	}

	utils.WriteJSON(w, http.StatusOK, utils.Envelope{"message": "distributors fetched successfully", "distributors": distributors})
}

// Distributor Login Handler
func (dh *DistributorHandler) HandleDistributorLogin(w http.ResponseWriter, r *http.Request) {
	var req struct {
		models.DistributorModel
		models.LoginDeviceInfo
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		utils.BadRequest(w, dh.logger, "distributor login", err)
		return
	}

	if req.DistributorID == "" || req.DistributorPassword == "" {
		utils.BadRequest(w, dh.logger, "distributor login", errors.New("id and password are required"))
		return
	}

	if err := dh.distributorStore.GetDistributorDetailsForLogin(&req.DistributorModel); err != nil {
		utils.BadRequest(w, dh.logger, "distributor login", err)
		return
	}

	token, err := utils.GenerateToken(req.DistributorID, req.DistributorName)
	if err != nil {
		utils.ServerError(w, dh.logger, "distributor login", err)
		return
	}

	dh.logger.Info("distributor login",
		"user_id", req.DistributorID,
		"user_agent", req.UserAgent,
		"platform", req.Platform,
		"latitude", req.Latitude,
		"longitude", req.Longitude,
		"timestamp", req.Timestamp,
	)
	go func() {
		if err := dh.loginActivityStore.CreateLoginActivity(models.LoginActivity{
			UserID: req.DistributorID, UserAgent: req.UserAgent, Platform: req.Platform,
			Latitude: req.Latitude, Longitude: req.Longitude, Accuracy: req.Accuracy,
			LoginTimestamp: req.Timestamp,
		}); err != nil {
			dh.logger.Error("failed to save login activity", "error", err, "user_id", req.DistributorID)
		}
	}()

	utils.WriteJSON(w, http.StatusOK, utils.Envelope{"message": "distributor login successful", "token": token})
}

// Delete Distributor Handler
func (dh *DistributorHandler) HandleDeleteDistributor(w http.ResponseWriter, r *http.Request) {
	id, err := utils.ReadParamID(r)
	if err != nil {
		utils.BadRequest(w, dh.logger, "delete distributor", err)
		return
	}

	if err := dh.distributorStore.DeleteDistributor(id); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			utils.BadRequest(w, dh.logger, "delete distributor", errors.New("distributor not found"))
			return
		}
		utils.ServerError(w, dh.logger, "delete distributor", err)
		return
	}

	utils.WriteJSON(w, http.StatusOK, utils.Envelope{"message": "distributor deleted successfully"})
}

// Change Distributors Master Distributor Handler
func (dh *DistributorHandler) HandleChangeDistributorsMasterDistributor(w http.ResponseWriter, r *http.Request) {
	id, err := utils.ReadParamID(r)
	if err != nil {
		utils.BadRequest(w, dh.logger, "update distributor master distributor", err)
		return
	}

	var req struct {
		MasterDistributorID string `json:"master_distributor_id"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		utils.BadRequest(w, dh.logger, "update distributor master distributor", err)
		return
	}
	if req.MasterDistributorID == "" {
		utils.BadRequest(w, dh.logger, "update distributor master distributor", errors.New("master_distributor_id is required"))
		return
	}

	if err := dh.distributorStore.ChangeDistributorsMasterDistributor(id, req.MasterDistributorID); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			utils.BadRequest(w, dh.logger, "update distributor master distributor", errors.New("distributor not found"))
			return
		}
		utils.ServerError(w, dh.logger, "update distributor master distributor", err)
		return
	}

	utils.WriteJSON(w, http.StatusOK, utils.Envelope{"message": "distributor master distributor updated successfully"})
}

// Update Distributor Aadhar Image Handler
func (dh *DistributorHandler) HandleUpdateDistributorAadharImage(w http.ResponseWriter, r *http.Request) {
	id, err := utils.ReadParamID(r)
	if err != nil {
		utils.BadRequest(w, dh.logger, "update distributor aadhar image", err)
		return
	}
	path := fmt.Sprintf("documents/%s/%s_aadhar_%d.png", id, id, time.Now().Unix())
	url, err := dh.awss3.GenerateUploadPresignedURL(path)
	if err != nil {
		utils.ServerError(w, dh.logger, "update distributor aadhar image", err)
		return
	}
	err = dh.distributorStore.UpdateDistributorAadharImage(path, id)
	if err != nil {
		utils.ServerError(w, dh.logger, "update distributor aadhar image", err)
		return
	}

	utils.WriteJSON(w, http.StatusOK, utils.Envelope{"message": "distributor aadhar image upload url generated successfully", "url": url})
}

// Update Distributor Pan Image Handler
func (dh *DistributorHandler) HandleUpdateDistributorPanImage(w http.ResponseWriter, r *http.Request) {
	id, err := utils.ReadParamID(r)
	if err != nil {
		utils.BadRequest(w, dh.logger, "update distributor pan image", err)
		return
	}
	path := fmt.Sprintf("documents/%s/%s_pan_%d.png", id, id, time.Now().Unix())
	url, err := dh.awss3.GenerateUploadPresignedURL(path)
	if err != nil {
		utils.ServerError(w, dh.logger, "update distributor pan image", err)
		return
	}
	err = dh.distributorStore.UpdateDistributorPanImage(path, id)
	if err != nil {
		utils.ServerError(w, dh.logger, "update distributor pan image", err)
		return
	}

	utils.WriteJSON(w, http.StatusOK, utils.Envelope{"message": "distributor pan image upload url generated successfully", "url": url})
}

// Update Distributor Image Handler
func (dh *DistributorHandler) HandleUpdateDistributorImage(w http.ResponseWriter, r *http.Request) {
	id, err := utils.ReadParamID(r)
	if err != nil {
		utils.BadRequest(w, dh.logger, "update distributor image", err)
		return
	}
	path := fmt.Sprintf("documents/%s/%s_image_%d.png", id, id, time.Now().Unix())
	url, err := dh.awss3.GenerateUploadPresignedURL(path)
	if err != nil {
		utils.ServerError(w, dh.logger, "update distributor image", err)
		return
	}
	err = dh.distributorStore.UpdateDistributorImage(path, id)
	if err != nil {
		utils.ServerError(w, dh.logger, "update distributor image", err)
		return
	}

	utils.WriteJSON(w, http.StatusOK, utils.Envelope{"message": "distributor image upload url generated successfully", "url": url})
}

// Get Distributor Wallet Balance Handler
func (dh *DistributorHandler) HandleGetDistributorWalletBalance(w http.ResponseWriter, r *http.Request) {
	id, err := utils.ReadParamID(r)
	if err != nil {
		utils.BadRequest(w, dh.logger, "get distributor wallet balance", err)
		return
	}
	balance, err := dh.distributorStore.GetDistributorWalletBalance(id)
	if err != nil {
		utils.ServerError(w, dh.logger, "get distributor wallet balance", err)
		return
	}
	utils.WriteJSON(w, http.StatusOK, utils.Envelope{"message": "distributor wallet balance fetched successfully", "balance": balance})
}

func (dh *DistributorHandler) HandleUpdateDistributorHoldAmount(w http.ResponseWriter, r *http.Request) {
	id, err := utils.ReadParamID(r)
	if err != nil {
		utils.BadRequest(w, dh.logger, "update distributor hold amount", err)
		return
	}
	var req struct {
		HoldAmount float64 `json:"hold_amount"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		utils.BadRequest(w, dh.logger, "update distributor hold amount", err)
		return
	}
	if req.HoldAmount < 0 {
		utils.BadRequest(w, dh.logger, "update distributor hold amount", fmt.Errorf("hold_amount cannot be negative"))
		return
	}
	if err := dh.distributorStore.UpdateDistributorHoldAmount(id, req.HoldAmount); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			utils.BadRequest(w, dh.logger, "update distributor hold amount", errors.New("distributor not found"))
			return
		}
		utils.ServerError(w, dh.logger, "update distributor hold amount", err)
		return
	}
	utils.WriteJSON(w, http.StatusOK, utils.Envelope{"message": "distributor hold amount updated successfully"})
}
