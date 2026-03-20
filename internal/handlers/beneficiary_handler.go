package handlers

import (
	"database/sql"
	"encoding/json"
	"errors"
	"log/slog"
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"github.com/levionstudio/fintech/internal/models"
	"github.com/levionstudio/fintech/internal/store"
	"github.com/levionstudio/fintech/internal/utils"
)

type BeneficiaryHandler struct {
	beneficiaryStore store.BeneficiaryStore
	logger           *slog.Logger
}

func NewBeneficiaryHandler(beneficiaryStore store.BeneficiaryStore, logger *slog.Logger) *BeneficiaryHandler {
	return &BeneficiaryHandler{beneficiaryStore: beneficiaryStore, logger: logger}
}

// Create Beneficiary Handler
func (bh *BeneficiaryHandler) HandleCreateBeneficiary(w http.ResponseWriter, r *http.Request) {
	var b models.BeneficiaryModel
	if err := json.NewDecoder(r.Body).Decode(&b); err != nil {
		utils.BadRequest(w, bh.logger, "create beneficiary", err)
		return
	}

	if err := b.Validate(); err != nil {
		utils.BadRequest(w, bh.logger, "create beneficiary", err)
		return
	}

	if err := bh.beneficiaryStore.CreateBeneficiary(&b); err != nil {
		utils.ServerError(w, bh.logger, "create beneficiary", err)
		return
	}

	utils.WriteJSON(w, http.StatusCreated, utils.Envelope{"message": "beneficiary created successfully", "beneficiary": b})
}

// Update Beneficiary Handler
func (bh *BeneficiaryHandler) HandleUpdateBeneficiary(w http.ResponseWriter, r *http.Request) {
	id, err := utils.ReadParamID(r)
	if err != nil {
		utils.BadRequest(w, bh.logger, "update beneficiary", err)
		return
	}

	var b models.BeneficiaryModel
	if err := json.NewDecoder(r.Body).Decode(&b); err != nil {
		utils.BadRequest(w, bh.logger, "update beneficiary", err)
		return
	}

	b.BeneficiaryID = id
	if err := bh.beneficiaryStore.UpdateBeneficiary(&b); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			utils.BadRequest(w, bh.logger, "update beneficiary", errors.New("beneficiary not found"))
			return
		}
		utils.ServerError(w, bh.logger, "update beneficiary", err)
		return
	}

	utils.WriteJSON(w, http.StatusOK, utils.Envelope{"message": "beneficiary updated successfully"})
}

// Delete Beneficiary Handler
func (bh *BeneficiaryHandler) HandleDeleteBeneficiary(w http.ResponseWriter, r *http.Request) {
	id, err := utils.ReadParamID(r)
	if err != nil {
		utils.BadRequest(w, bh.logger, "delete beneficiary", err)
		return
	}

	if err := bh.beneficiaryStore.DeleteBeneficiary(id); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			utils.BadRequest(w, bh.logger, "delete beneficiary", errors.New("beneficiary not found"))
			return
		}
		utils.ServerError(w, bh.logger, "delete beneficiary", err)
		return
	}

	utils.WriteJSON(w, http.StatusOK, utils.Envelope{"message": "beneficiary deleted successfully"})
}

// Verify Beneficiary Handler
func (bh *BeneficiaryHandler) HandleVerifyBeneficiary(w http.ResponseWriter, r *http.Request) {
	id, err := utils.ReadParamID(r)
	if err != nil {
		utils.BadRequest(w, bh.logger, "verify beneficiary", err)
		return
	}

	var req models.VerifyBeneficiaryRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		utils.BadRequest(w, bh.logger, "verify beneficiary", err)
		return
	}

	if err := req.Validate(); err != nil {
		utils.BadRequest(w, bh.logger, "verify beneficiary", err)
		return
	}

	reqID := id + uuid.NewString()
	token, err := utils.GeneratePaysprintToken(reqID)
	if err != nil {
		utils.ServerError(w, bh.logger, "verify beneficiary", err)
		return
	}

	var apiResp models.VerifyBeneficiaryResponse
	err = utils.PostRequest(utils.PaysprintAPI+utils.PennyDrop, "Token", token, map[string]any{
		"refid":          reqID,
		"account_number": req.AccountNumber,
		"ifsc_code":      req.IFSCCode,
	}, &apiResp)
	if err != nil {
		utils.ServerError(w, bh.logger, "verify beneficiary: paysprint call", err)
		return
	}

	if !apiResp.Status {
		utils.BadRequest(w, bh.logger, "verify beneficiary", errors.New(apiResp.Message))
		return
	}

	if err := bh.beneficiaryStore.VerifyBeneficiary(id); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			utils.BadRequest(w, bh.logger, "verify beneficiary", errors.New("beneficiary not found"))
			return
		}
		utils.ServerError(w, bh.logger, "verify beneficiary", err)
		return
	}

	utils.WriteJSON(w, http.StatusOK, utils.Envelope{"message": "beneficiary verified successfully", "data": apiResp.Data})
}

// Get Beneficiaries Handler
func (bh *BeneficiaryHandler) HandleGetBeneficiaries(w http.ResponseWriter, r *http.Request) {
	mobileNumber := chi.URLParam(r, "mobile")
	if mobileNumber == "" {
		utils.BadRequest(w, bh.logger, "get beneficiaries", errors.New("mobile_number is required"))
		return
	}

	p := utils.ReadPaginationParams(r)

	beneficiaries, err := bh.beneficiaryStore.GetBeneficiaries(mobileNumber, p)
	if err != nil {
		utils.ServerError(w, bh.logger, "get beneficiaries", err)
		return
	}

	utils.WriteJSON(w, http.StatusOK, utils.Envelope{"message": "beneficiaries fetched successfully", "beneficiaries": beneficiaries})
}
