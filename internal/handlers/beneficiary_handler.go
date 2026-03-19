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

const paysprintPennyDropPath = "/api/v1/verification/penny_drop_v2"

type BeneficiaryHandler struct {
	beneficiaryStore store.BeneficiaryStore
	logger           *slog.Logger
}

func NewBeneficiaryHandler(beneficiaryStore store.BeneficiaryStore, logger *slog.Logger) *BeneficiaryHandler {
	return &BeneficiaryHandler{beneficiaryStore: beneficiaryStore, logger: logger}
}

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

	utils.WriteJSON(w, http.StatusCreated, utils.Envelope{"beneficiary": b})
}

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

	if err := bh.beneficiaryStore.UpdateBeneficiary(id, &b); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			utils.BadRequest(w, bh.logger, "update beneficiary", errors.New("beneficiary not found"))
			return
		}
		utils.ServerError(w, bh.logger, "update beneficiary", err)
		return
	}

	utils.WriteJSON(w, http.StatusOK, utils.Envelope{"message": "beneficiary updated successfully"})
}

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

	reqID := utils.GenerateReqID()
	token, err := utils.GeneratePaysprintToken(reqID)
	if err != nil {
		utils.ServerError(w, bh.logger, "verify beneficiary", err)
		return
	}

	var apiResp models.VerifyBeneficiaryResponse
	err = utils.PostRequest(utils.PaysprintAPI+paysprintPennyDropPath, "Token", token, map[string]any{
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

func (bh *BeneficiaryHandler) HandleGetBeneficiary(w http.ResponseWriter, r *http.Request) {
	id, err := utils.ReadParamID(r)
	if err != nil {
		utils.BadRequest(w, bh.logger, "get beneficiary", err)
		return
	}

	b, err := bh.beneficiaryStore.GetBeneficiary(id)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			utils.BadRequest(w, bh.logger, "get beneficiary", errors.New("beneficiary not found"))
			return
		}
		utils.ServerError(w, bh.logger, "get beneficiary", err)
		return
	}

	utils.WriteJSON(w, http.StatusOK, utils.Envelope{"beneficiary": b})
}
