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

type MasterDistributorHandler struct {
	mdStore store.MasterDistributorStore
	logger  *slog.Logger
	awss3   *utils.AWSS3
}

func NewMasterDistributorHandler(mdStore store.MasterDistributorStore, logger *slog.Logger, awss3 *utils.AWSS3) *MasterDistributorHandler {
	return &MasterDistributorHandler{
		mdStore: mdStore,
		logger:  logger,
		awss3:   awss3,
	}
}

// Create Master Distributor Handler
func (mh *MasterDistributorHandler) HandleCreateMasterDistributor(w http.ResponseWriter, r *http.Request) {
	var req models.MasterDistributorModel
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		utils.BadRequest(w, mh.logger, "create master distributor", err)
		return
	}

	if err := req.ValidateCreateMasterDistributor(); err != nil {
		utils.BadRequest(w, mh.logger, "create master distributor", err)
		return
	}

	if err := mh.mdStore.CreateMasterDistributor(&req); err != nil {
		utils.ServerError(w, mh.logger, "create master distributor", err)
		return
	}

	utils.WriteJSON(w, http.StatusCreated, utils.Envelope{"message": "master distributor created successfully", "master_distributor": req})
}

// Update Master Distributor Details Handler
func (mh *MasterDistributorHandler) HandleUpdateMasterDistributorDetails(w http.ResponseWriter, r *http.Request) {
	id, err := utils.ReadParamID(r)
	if err != nil {
		utils.BadRequest(w, mh.logger, "update master distributor", err)
		return
	}

	var req models.MasterDistributorModel
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		utils.BadRequest(w, mh.logger, "update master distributor", err)
		return
	}

	req.MasterDistributorID = id
	if err := mh.mdStore.UpdateMasterDistributorDetails(&req); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			utils.BadRequest(w, mh.logger, "update master distributor", errors.New("master distributor not found"))
			return
		}
		utils.ServerError(w, mh.logger, "update master distributor", err)
		return
	}

	utils.WriteJSON(w, http.StatusOK, utils.Envelope{"message": "master distributor updated successfully"})
}

// Update Master Distributor Password Handler
func (mh *MasterDistributorHandler) HandleUpdateMasterDistributorPassword(w http.ResponseWriter, r *http.Request) {
	id, err := utils.ReadParamID(r)
	if err != nil {
		utils.BadRequest(w, mh.logger, "update master distributor password", err)
		return
	}

	var req models.MasterDistributorModel
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		utils.BadRequest(w, mh.logger, "update master distributor password", err)
		return
	}

	if req.MasterDistributorPassword == "" || !utils.IsValidPassword(req.MasterDistributorPassword) {
		utils.BadRequest(w, mh.logger, "update master distributor password", errors.New("password is empty or does not meet strength requirements"))
		return
	}

	req.MasterDistributorID = id
	if err := mh.mdStore.UpdateMasterDistributorPassword(&req); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			utils.BadRequest(w, mh.logger, "update master distributor password", errors.New("master distributor not found"))
			return
		}
		utils.ServerError(w, mh.logger, "update master distributor password", err)
		return
	}

	utils.WriteJSON(w, http.StatusOK, utils.Envelope{"message": "master distributor password updated successfully", "password": req.MasterDistributorPassword})
}

// Update Master Distributor MPIN Handler
func (mh *MasterDistributorHandler) HandleUpdateMasterDistributorMpin(w http.ResponseWriter, r *http.Request) {
	id, err := utils.ReadParamID(r)
	if err != nil {
		utils.BadRequest(w, mh.logger, "update master distributor mpin", err)
		return
	}

	var req models.MasterDistributorModel
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		utils.BadRequest(w, mh.logger, "update master distributor mpin", err)
		return
	}

	if req.MasterDistributorMpin < 1000 || req.MasterDistributorMpin > 9999 {
		utils.BadRequest(w, mh.logger, "update master distributor mpin", errors.New("mpin must be a 4-digit number"))
		return
	}

	req.MasterDistributorID = id
	if err := mh.mdStore.UpdateMasterDistributorMpin(&req); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			utils.BadRequest(w, mh.logger, "update master distributor mpin", errors.New("master distributor not found"))
			return
		}
		utils.ServerError(w, mh.logger, "update master distributor mpin", err)
		return
	}

	utils.WriteJSON(w, http.StatusOK, utils.Envelope{"message": "master distributor mpin updated successfully", "mpin": req.MasterDistributorMpin})
}

// Update Master Distributor KYC Status Handler
func (mh *MasterDistributorHandler) HandleUpdateMasterDistributorKYCStatus(w http.ResponseWriter, r *http.Request) {
	id, err := utils.ReadParamID(r)
	if err != nil {
		utils.BadRequest(w, mh.logger, "update master distributor kyc status", err)
		return
	}

	var req models.MasterDistributorModel
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		utils.BadRequest(w, mh.logger, "update master distributor kyc status", err)
		return
	}

	req.MasterDistributorID = id
	if err := mh.mdStore.UpdateMasterDistributorKYCStatus(&req); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			utils.BadRequest(w, mh.logger, "update master distributor kyc status", errors.New("master distributor not found"))
			return
		}
		utils.ServerError(w, mh.logger, "update master distributor kyc status", err)
		return
	}

	utils.WriteJSON(w, http.StatusOK, utils.Envelope{"message": "master distributor kyc status updated successfully"})
}

// Update Master Distributor Block Status Handler
func (mh *MasterDistributorHandler) HandleUpdateMasterDistributorBlockStatus(w http.ResponseWriter, r *http.Request) {
	id, err := utils.ReadParamID(r)
	if err != nil {
		utils.BadRequest(w, mh.logger, "update master distributor block status", err)
		return
	}

	var req models.MasterDistributorModel
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		utils.BadRequest(w, mh.logger, "update master distributor block status", err)
		return
	}

	req.MasterDistributorID = id
	if err := mh.mdStore.UpdateMasterDistributorBlockStatus(&req); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			utils.BadRequest(w, mh.logger, "update master distributor block status", errors.New("master distributor not found"))
			return
		}
		utils.ServerError(w, mh.logger, "update master distributor block status", err)
		return
	}

	utils.WriteJSON(w, http.StatusOK, utils.Envelope{"message": "master distributor block status updated successfully"})
}

// Get Master Distributor By ID Handler
func (mh *MasterDistributorHandler) HandleGetMasterDistributorByID(w http.ResponseWriter, r *http.Request) {
	id, err := utils.ReadParamID(r)
	if err != nil {
		utils.BadRequest(w, mh.logger, "get master distributor by id", err)
		return
	}

	md, err := mh.mdStore.GetMasterDistributorByID(id)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			utils.BadRequest(w, mh.logger, "get master distributor by id", errors.New("master distributor not found"))
			return
		}
		utils.ServerError(w, mh.logger, "get master distributor by id", err)
		return
	}

	utils.WriteJSON(w, http.StatusOK, utils.Envelope{"message": "master distributor details fetched successfully", "master_distributor": md})
}

// Get Master Distributors By Admin ID For Dropdown Handler
func (mh *MasterDistributorHandler) HandleGetMasterDistributorsByAdminIDForDropdown(w http.ResponseWriter, r *http.Request) {
	adminID, err := utils.ReadParamID(r)
	if err != nil {
		utils.BadRequest(w, mh.logger, "get master distributors dropdown", err)
		return
	}

	mds, err := mh.mdStore.GetMasterDistributorsByAdminIDForDropdown(adminID)
	if err != nil {
		utils.ServerError(w, mh.logger, "get master distributors dropdown", err)
		return
	}

	utils.WriteJSON(w, http.StatusOK, utils.Envelope{"message": "master distributors fetched successfully", "master_distributors": mds})
}

// Get Master Distributors By Admin ID Handler
func (mh *MasterDistributorHandler) HandleGetMasterDistributorsByAdminID(w http.ResponseWriter, r *http.Request) {
	adminID, err := utils.ReadParamID(r)
	if err != nil {
		utils.BadRequest(w, mh.logger, "get master distributors by admin id", err)
		return
	}

	p := utils.ReadPaginationParams(r)

	mds, err := mh.mdStore.GetMasterDistributorsByAdminID(adminID, p.Limit, p.Offset)
	if err != nil {
		utils.ServerError(w, mh.logger, "get master distributors by admin id", err)
		return
	}

	utils.WriteJSON(w, http.StatusOK, utils.Envelope{"master_distributors": mds})
}

// Master Distributor Login Handler
func (mh *MasterDistributorHandler) HandleMasterDistributorLogin(w http.ResponseWriter, r *http.Request) {
	var req models.MasterDistributorModel
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		utils.BadRequest(w, mh.logger, "master distributor login", err)
		return
	}

	if req.MasterDistributorID == "" || req.MasterDistributorPassword == "" {
		utils.BadRequest(w, mh.logger, "master distributor login", errors.New("id and password are required"))
		return
	}

	if err := mh.mdStore.GetMasterDistributorDetailsForLogin(&req); err != nil {
		utils.BadRequest(w, mh.logger, "master distributor login", err)
		return
	}

	token, err := utils.GenerateToken(req.MasterDistributorID, req.MasterDistributorName)
	if err != nil {
		utils.ServerError(w, mh.logger, "master distributor login", err)
		return
	}

	utils.WriteJSON(w, http.StatusOK, utils.Envelope{"message": "master distributor login successfull", "token": token})
}

// Master Distributor Delete Handler
func (mh *MasterDistributorHandler) HandleDeleteMasterDistributor(w http.ResponseWriter, r *http.Request) {
	id, err := utils.ReadParamID(r)
	if err != nil {
		utils.BadRequest(w, mh.logger, "delete master distributor", err)
		return
	}

	if err := mh.mdStore.DeleteMasterDistributor(id); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			utils.BadRequest(w, mh.logger, "delete master distributor", errors.New("master distributor not found"))
			return
		}
		utils.ServerError(w, mh.logger, "delete master distributor", err)
		return
	}

	utils.WriteJSON(w, http.StatusOK, utils.Envelope{"message": "master distributor deleted successfully"})
}

// Get Master Distributor Wallet Balance
func (mh *MasterDistributorHandler) HandleGetMasterDistributorWalletBalance(w http.ResponseWriter, r *http.Request) {
	id, err := utils.ReadParamID(r)
	if err != nil {
		utils.BadRequest(w, mh.logger, "get master distributor wallet balance", err)
		return
	}

	balance, err := mh.mdStore.GetMasterDistributorWalletBalance(id)

	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			utils.BadRequest(w, mh.logger, "get master distributor wallet balance", errors.New("master distributor not found"))
			return
		}
		utils.ServerError(w, mh.logger, "get master distributor wallet balance", err)
		return
	}

	utils.WriteJSON(w, http.StatusOK, utils.Envelope{"message": "master distributor wallet balance fetched successfully", "balance": balance})
}

// Update Master Distributor Aadhar Image
func (mh *MasterDistributorHandler) HandleUpdateMasterDistributorAadharImage(w http.ResponseWriter, r *http.Request) {
	id, err := utils.ReadParamID(r)
	if err != nil {
		utils.BadRequest(w, mh.logger, "update master distributor aadhar image", err)
		return
	}
	path := fmt.Sprintf("documents/%s/%s_aadhar_%d.png", id, id, time.Now().Unix())
	url, err := mh.awss3.GenerateUploadPresignedURL(path)
	if err != nil {
		utils.ServerError(w, mh.logger, "update master distributor aadhar image", err)
		return
	}
	err = mh.mdStore.UpdateMasterDistributorAadharImage(path, id)
	if err != nil {
		utils.ServerError(w, mh.logger, "update master distributor aadhar image", err)
		return
	}

	utils.WriteJSON(w, http.StatusOK, utils.Envelope{"message": "master distributor aadhar image upload url generated successfully", "url": url})
}

// Update Master Distributor Pan Image
func (mh *MasterDistributorHandler) HandleUpdateMasterDistributorPanImage(w http.ResponseWriter, r *http.Request) {
	id, err := utils.ReadParamID(r)
	if err != nil {
		utils.BadRequest(w, mh.logger, "update master distributor pan image", err)
		return
	}
	path := fmt.Sprintf("documents/%s/%s_pan_%d.png", id, id, time.Now().Unix())
	url, err := mh.awss3.GenerateUploadPresignedURL(path)
	if err != nil {
		utils.ServerError(w, mh.logger, "update master distributor pan image", err)
		return
	}
	err = mh.mdStore.UpdateMasterDistributorPanImage(path, id)
	if err != nil {
		utils.ServerError(w, mh.logger, "update master distributor pan image", err)
		return
	}

	utils.WriteJSON(w, http.StatusOK, utils.Envelope{"message": "master distributor pan image upload url generated successfully", "url": url})
}

// Update Master Distributor Image
func (mh *MasterDistributorHandler) HandleUpdateMasterDistributorImage(w http.ResponseWriter, r *http.Request) {
	id, err := utils.ReadParamID(r)
	if err != nil {
		utils.BadRequest(w, mh.logger, "update master distributor image", err)
		return
	}
	path := fmt.Sprintf("documents/%s/%s_image_%d.png", id, id, time.Now().Unix())
	url, err := mh.awss3.GenerateUploadPresignedURL(path)
	if err != nil {
		utils.ServerError(w, mh.logger, "update master distributor image", err)
		return
	}
	err = mh.mdStore.UpdateMasterDistributorImage(path, id)
	if err != nil {
		utils.ServerError(w, mh.logger, "update master distributor image", err)
		return
	}

	utils.WriteJSON(w, http.StatusOK, utils.Envelope{"message": "master distributor image upload url generated successfully", "url": url})
}
