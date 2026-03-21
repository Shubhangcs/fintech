package handlers

import (
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"

	"github.com/levionstudio/fintech/internal/models"
	"github.com/levionstudio/fintech/internal/store"
	"github.com/levionstudio/fintech/internal/utils"
)

type FundTransferHandler struct {
	fundTransferStore store.FundTransferStore
	logger            *slog.Logger
}

func NewFundTransferHandler(fundTransferStore store.FundTransferStore, logger *slog.Logger) *FundTransferHandler {
	return &FundTransferHandler{
		fundTransferStore: fundTransferStore,
		logger:            logger,
	}
}

// Admin To MD Fund Transfer Handler
func (fh *FundTransferHandler) HandleAdminToMD(w http.ResponseWriter, r *http.Request) {
	fh.handleTransfer(w, r, "admin to md", fh.fundTransferStore.AdminToMD)
}

// Admin To Distributor Fund Transfer Handler
func (fh *FundTransferHandler) HandleAdminToDistributor(w http.ResponseWriter, r *http.Request) {
	fh.handleTransfer(w, r, "admin to distributor", fh.fundTransferStore.AdminToDistributor)
}

// Admin To Retailer Fund Transfer Handler
func (fh *FundTransferHandler) HandleAdminToRetailer(w http.ResponseWriter, r *http.Request) {
	fh.handleTransfer(w, r, "admin to retailer", fh.fundTransferStore.AdminToRetailer)
}

// MD To Distributor Fund Transfer Handler
func (fh *FundTransferHandler) HandleMDToDistributor(w http.ResponseWriter, r *http.Request) {
	fh.handleTransfer(w, r, "md to distributor", fh.fundTransferStore.MDToDistributor)
}

// MD To Retailer Fund Transfer Handler
func (fh *FundTransferHandler) HandleMDToRetailer(w http.ResponseWriter, r *http.Request) {
	fh.handleTransfer(w, r, "md to retailer", fh.fundTransferStore.MDToRetailer)
}

// Distributor To Retailer Fund Transfer Handler
func (fh *FundTransferHandler) HandleDistributorToRetailer(w http.ResponseWriter, r *http.Request) {
	fh.handleTransfer(w, r, "distributor to retailer", fh.fundTransferStore.DistributorToRetailer)
}

func (fh *FundTransferHandler) handleTransfer(
	w http.ResponseWriter,
	r *http.Request,
	op string,
	transferFn func(*models.FundTransferModel) error,
) {
	var req models.FundTransferModel
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		utils.BadRequest(w, fh.logger, op, err)
		return
	}

	if err := req.Validate(); err != nil {
		utils.BadRequest(w, fh.logger, op, err)
		return
	}

	if req.Remarks == "" {
		req.Remarks = fmt.Sprintf("Fund transfer from %s to %s", req.FundTransfererID, req.FundReceiverID)
	}

	if err := transferFn(&req); err != nil {
		if isClientErr(err) {
			utils.BadRequest(w, fh.logger, op, err)
			return
		}
		utils.ServerError(w, fh.logger, op, err)
		return
	}

	utils.WriteJSON(w, http.StatusOK, utils.Envelope{"message": "fund transfer successful", "fund_transfer": req})
}

// Get Fund Transfers By Transferer ID Handler
func (fh *FundTransferHandler) HandleGetFundTransfersByTransfererID(w http.ResponseWriter, r *http.Request) {
	id, err := utils.ReadParamID(r)
	if err != nil {
		utils.BadRequest(w, fh.logger, "get fund transfers by transferer id", err)
		return
	}

	p := utils.ReadQueryParams(r)

	transfers, err := fh.fundTransferStore.GetFundTransfersByTransfererID(id, p)
	if err != nil {
		utils.ServerError(w, fh.logger, "get fund transfers by transferer id", err)
		return
	}

	utils.WriteJSON(w, http.StatusOK, utils.Envelope{"message": "fund transfers fetched successfully", "fund_transfers": transfers})
}

// Get Fund Transfers By Receiver ID Handler
func (fh *FundTransferHandler) HandleGetFundTransfersByReceiverID(w http.ResponseWriter, r *http.Request) {
	id, err := utils.ReadParamID(r)
	if err != nil {
		utils.BadRequest(w, fh.logger, "get fund transfers by receiver id", err)
		return
	}

	p := utils.ReadQueryParams(r)

	transfers, err := fh.fundTransferStore.GetFundTransfersByReceiverID(id, p)
	if err != nil {
		utils.ServerError(w, fh.logger, "get fund transfers by receiver id", err)
		return
	}

	utils.WriteJSON(w, http.StatusOK, utils.Envelope{"message": "fund transfers fetched successfully", "fund_transfers": transfers})
}

// Get All Fund Transfers Handler
func (fh *FundTransferHandler) HandleGetAllFundTransfers(w http.ResponseWriter, r *http.Request) {
	p := utils.ReadQueryParams(r)

	transfers, err := fh.fundTransferStore.GetAllFundTransfers(p)
	if err != nil {
		utils.ServerError(w, fh.logger, "get all fund transfers", err)
		return
	}

	utils.WriteJSON(w, http.StatusOK, utils.Envelope{"message": "fund transfers fetched successfully", "fund_transfers": transfers})
}

func isClientErr(err error) bool {
	msg := err.Error()
	return msg == "insufficient balance" || msg == "sender not found" || msg == "receiver not found"
}
