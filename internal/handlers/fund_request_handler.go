package handlers

import (
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"strconv"

	"github.com/levionstudio/fintech/internal/models"
	"github.com/levionstudio/fintech/internal/store"
	"github.com/levionstudio/fintech/internal/utils"
)

type FundRequestHandler struct {
	fundRequestStore store.FundRequestStore
	logger           *slog.Logger
}

func NewFundRequestHandler(fundRequestStore store.FundRequestStore, logger *slog.Logger) *FundRequestHandler {
	return &FundRequestHandler{
		fundRequestStore: fundRequestStore,
		logger:           logger,
	}
}

// --- create handlers ---

func (fh *FundRequestHandler) HandleMDRequestToAdmin(w http.ResponseWriter, r *http.Request) {
	fh.handleCreate(w, r, "md request to admin", "M", "A", fh.fundRequestStore.MDRequestToAdmin)
}

func (fh *FundRequestHandler) HandleDistributorRequestToAdmin(w http.ResponseWriter, r *http.Request) {
	fh.handleCreate(w, r, "distributor request to admin", "D", "A", fh.fundRequestStore.DistributorRequestToAdmin)
}

func (fh *FundRequestHandler) HandleDistributorRequestToMD(w http.ResponseWriter, r *http.Request) {
	fh.handleCreate(w, r, "distributor request to md", "D", "M", fh.fundRequestStore.DistributorRequestToMD)
}

func (fh *FundRequestHandler) HandleRetailerRequestToAdmin(w http.ResponseWriter, r *http.Request) {
	fh.handleCreate(w, r, "retailer request to admin", "R", "A", fh.fundRequestStore.RetailerRequestToAdmin)
}

func (fh *FundRequestHandler) HandleRetailerRequestToMD(w http.ResponseWriter, r *http.Request) {
	fh.handleCreate(w, r, "retailer request to md", "R", "M", fh.fundRequestStore.RetailerRequestToMD)
}

func (fh *FundRequestHandler) HandleRetailerRequestToDistributor(w http.ResponseWriter, r *http.Request) {
	fh.handleCreate(w, r, "retailer request to distributor", "R", "D", fh.fundRequestStore.RetailerRequestToDistributor)
}

// --- approve / reject ---

func (fh *FundRequestHandler) HandleApproveFundRequest(w http.ResponseWriter, r *http.Request) {
	id, err := readFundRequestID(r)
	if err != nil {
		utils.BadRequest(w, fh.logger, "approve fund request", err)
		return
	}

	if err := fh.fundRequestStore.ApproveFundRequest(id); err != nil {
		if isFundRequestClientErr(err) {
			utils.BadRequest(w, fh.logger, "approve fund request", err)
			return
		}
		utils.ServerError(w, fh.logger, "approve fund request", err)
		return
	}

	utils.WriteJSON(w, http.StatusOK, utils.Envelope{"message": "fund request approved successfully"})
}

func (fh *FundRequestHandler) HandleRejectFundRequest(w http.ResponseWriter, r *http.Request) {
	id, err := readFundRequestID(r)
	if err != nil {
		utils.BadRequest(w, fh.logger, "reject fund request", err)
		return
	}

	var body struct {
		RejectRemarks string `json:"reject_remarks"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		utils.BadRequest(w, fh.logger, "reject fund request", err)
		return
	}

	if body.RejectRemarks == "" {
		utils.BadRequest(w, fh.logger, "reject fund request", errors.New("reject_remarks is required"))
		return
	}

	if err := fh.fundRequestStore.RejectFundRequest(id, body.RejectRemarks); err != nil {
		if isFundRequestClientErr(err) {
			utils.BadRequest(w, fh.logger, "reject fund request", err)
			return
		}
		utils.ServerError(w, fh.logger, "reject fund request", err)
		return
	}

	utils.WriteJSON(w, http.StatusOK, utils.Envelope{"message": "fund request rejected successfully"})
}

// --- get handlers ---

func (fh *FundRequestHandler) HandleGetFundRequestsByRequesterID(w http.ResponseWriter, r *http.Request) {
	id, err := utils.ReadParamID(r)
	if err != nil {
		utils.BadRequest(w, fh.logger, "get fund requests by requester id", err)
		return
	}

	p := utils.ReadQueryParams(r)

	requests, err := fh.fundRequestStore.GetFundRequestsByRequesterID(id, p.Limit, p.Offset, p.StartDate, p.EndDate)
	if err != nil {
		utils.ServerError(w, fh.logger, "get fund requests by requester id", err)
		return
	}

	utils.WriteJSON(w, http.StatusOK, utils.Envelope{"message": "fund requests fetched successfully", "fund_requests": requests})
}

func (fh *FundRequestHandler) HandleGetFundRequestsByRequestToID(w http.ResponseWriter, r *http.Request) {
	id, err := utils.ReadParamID(r)
	if err != nil {
		utils.BadRequest(w, fh.logger, "get fund requests by request_to id", err)
		return
	}

	p := utils.ReadQueryParams(r)

	requests, err := fh.fundRequestStore.GetFundRequestsByRequestToID(id, p.Limit, p.Offset, p.StartDate, p.EndDate)
	if err != nil {
		utils.ServerError(w, fh.logger, "get fund requests by request_to id", err)
		return
	}

	utils.WriteJSON(w, http.StatusOK, utils.Envelope{"message": "fund requests fetched successfully", "fund_requests": requests})
}

func (fh *FundRequestHandler) HandleGetAllFundRequests(w http.ResponseWriter, r *http.Request) {
	p := utils.ReadQueryParams(r)

	requests, err := fh.fundRequestStore.GetAllFundRequests(p.Limit, p.Offset, p.StartDate, p.EndDate)
	if err != nil {
		utils.ServerError(w, fh.logger, "get all fund requests", err)
		return
	}

	utils.WriteJSON(w, http.StatusOK, utils.Envelope{"message": "fund requests fetched successfully", "fund_requests": requests})
}

// --- private helpers ---

func (fh *FundRequestHandler) handleCreate(
	w http.ResponseWriter,
	r *http.Request,
	op, requesterPrefix, requestToPrefix string,
	createFn func(*models.FundRequestModel) error,
) {
	var req models.FundRequestModel
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		utils.BadRequest(w, fh.logger, op, err)
		return
	}

	if err := req.Validate(); err != nil {
		utils.BadRequest(w, fh.logger, op, err)
		return
	}

	if len(req.RequesterID) == 0 || string(req.RequesterID[0]) != requesterPrefix {
		utils.BadRequest(w, fh.logger, op, fmt.Errorf("requester_id must belong to a %s user", prefixToRole(requesterPrefix)))
		return
	}

	if len(req.RequestToID) == 0 || string(req.RequestToID[0]) != requestToPrefix {
		utils.BadRequest(w, fh.logger, op, fmt.Errorf("request_to_id must belong to a %s user", prefixToRole(requestToPrefix)))
		return
	}

	if req.Remarks == "" {
		req.Remarks = fmt.Sprintf("Fund request from %s to %s", req.RequesterID, req.RequestToID)
	}

	if err := createFn(&req); err != nil {
		utils.ServerError(w, fh.logger, op, err)
		return
	}

	utils.WriteJSON(w, http.StatusCreated, utils.Envelope{"message": "fund request created successfully", "fund_request": req})
}

func readFundRequestID(r *http.Request) (int64, error) {
	idStr, err := utils.ReadParamID(r)
	if err != nil {
		return 0, err
	}

	id, err := strconv.ParseInt(idStr, 10, 64)
	if err != nil {
		return 0, errors.New("invalid fund request id")
	}

	return id, nil
}

func prefixToRole(prefix string) string {
	switch prefix {
	case "A":
		return "admin"
	case "M":
		return "master distributor"
	case "D":
		return "distributor"
	case "R":
		return "retailer"
	default:
		return "unknown"
	}
}

func isFundRequestClientErr(err error) bool {
	msg := err.Error()
	return msg == "fund request not found" ||
		msg == "fund request not found or already processed" ||
		msg == "insufficient balance" ||
		msg == "requester not found" ||
		msg == "request_to user not found" ||
		len(msg) > 24 && msg[:24] == "fund request is already "
}
