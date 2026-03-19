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

func (fh *FundTransferHandler) HandleAdminToMD(w http.ResponseWriter, r *http.Request) {
	fh.handleTransfer(w, r, "admin to md", fh.fundTransferStore.AdminToMD)
}

func (fh *FundTransferHandler) HandleAdminToDistributor(w http.ResponseWriter, r *http.Request) {
	fh.handleTransfer(w, r, "admin to distributor", fh.fundTransferStore.AdminToDistributor)
}

func (fh *FundTransferHandler) HandleAdminToRetailer(w http.ResponseWriter, r *http.Request) {
	fh.handleTransfer(w, r, "admin to retailer", fh.fundTransferStore.AdminToRetailer)
}

func (fh *FundTransferHandler) HandleMDToDistributor(w http.ResponseWriter, r *http.Request) {
	fh.handleTransfer(w, r, "md to distributor", fh.fundTransferStore.MDToDistributor)
}

func (fh *FundTransferHandler) HandleMDToRetailer(w http.ResponseWriter, r *http.Request) {
	fh.handleTransfer(w, r, "md to retailer", fh.fundTransferStore.MDToRetailer)
}

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

	utils.WriteJSON(w, http.StatusOK, utils.Envelope{"fund_transfer": req})
}

func (fh *FundTransferHandler) HandleGetFundTransfersByTransfererID(w http.ResponseWriter, r *http.Request) {
	id, err := utils.ReadParamID(r)
	if err != nil {
		utils.BadRequest(w, fh.logger, "get fund transfers by transferer id", err)
		return
	}

	p := utils.ReadPaginationParams(r)
	startDate := utils.ParseDateParam(r, "start_date")
	endDate := utils.ParseDateParam(r, "end_date")

	transfers, err := fh.fundTransferStore.GetFundTransfersByTransfererID(id, p.Limit, p.Offset, startDate, endDate)
	if err != nil {
		utils.ServerError(w, fh.logger, "get fund transfers by transferer id", err)
		return
	}

	utils.WriteJSON(w, http.StatusOK, utils.Envelope{"fund_transfers": transfers})
}

func (fh *FundTransferHandler) HandleGetFundTransfersByReceiverID(w http.ResponseWriter, r *http.Request) {
	id, err := utils.ReadParamID(r)
	if err != nil {
		utils.BadRequest(w, fh.logger, "get fund transfers by receiver id", err)
		return
	}

	p := utils.ReadPaginationParams(r)
	startDate := utils.ParseDateParam(r, "start_date")
	endDate := utils.ParseDateParam(r, "end_date")

	transfers, err := fh.fundTransferStore.GetFundTransfersByReceiverID(id, p.Limit, p.Offset, startDate, endDate)
	if err != nil {
		utils.ServerError(w, fh.logger, "get fund transfers by receiver id", err)
		return
	}

	utils.WriteJSON(w, http.StatusOK, utils.Envelope{"fund_transfers": transfers})
}

func (fh *FundTransferHandler) HandleGetAllFundTransfers(w http.ResponseWriter, r *http.Request) {
	p := utils.ReadPaginationParams(r)
	startDate := utils.ParseDateParam(r, "start_date")
	endDate := utils.ParseDateParam(r, "end_date")

	transfers, err := fh.fundTransferStore.GetAllFundTransfers(p.Limit, p.Offset, startDate, endDate)
	if err != nil {
		utils.ServerError(w, fh.logger, "get all fund transfers", err)
		return
	}

	utils.WriteJSON(w, http.StatusOK, utils.Envelope{"fund_transfers": transfers})
}

func isClientErr(err error) bool {
	msg := err.Error()
	return msg == "insufficient balance" || msg == "sender not found" || msg == "receiver not found"
}
