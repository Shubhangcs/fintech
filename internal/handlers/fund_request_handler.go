package handlers

import (
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"strings"
	"time"

	"github.com/levionstudio/fintech/internal/models"
	"github.com/levionstudio/fintech/internal/store"
	"github.com/levionstudio/fintech/internal/utils"
)

type FundRequestHandler struct {
	fundRequestStore store.FundRequestStore
	logger           *slog.Logger
	awss3            *utils.AWSS3
}

func NewFundRequestHandler(fundRequestStore store.FundRequestStore, logger *slog.Logger, awss3 *utils.AWSS3) *FundRequestHandler {
	return &FundRequestHandler{
		fundRequestStore: fundRequestStore,
		logger:           logger,
		awss3:            awss3,
	}
}

// Fund Request From Master Distributor to Admin Handler
func (fh *FundRequestHandler) HandleMDRequestToAdmin(w http.ResponseWriter, r *http.Request) {
	fh.handleCreate(w, r, "fund request from master distributor to admin", "M", "A")
}

// Fund Request From Distributor to Admin Handler
func (fh *FundRequestHandler) HandleDistributorRequestToAdmin(w http.ResponseWriter, r *http.Request) {
	fh.handleCreate(w, r, "fund request from distributor to admin", "D", "A")
}

// Fund Request From Distributor to Master Distributor Handler
func (fh *FundRequestHandler) HandleDistributorRequestToMD(w http.ResponseWriter, r *http.Request) {
	fh.handleCreate(w, r, "fund request from distributor to master distributor", "D", "M")
}

// Fund Request From Retailer to Admin Handler
func (fh *FundRequestHandler) HandleRetailerRequestToAdmin(w http.ResponseWriter, r *http.Request) {
	fh.handleCreate(w, r, "fund request from retailer to admin", "R", "A")
}

// Fund Request From Retailer to Master Distributor Handler
func (fh *FundRequestHandler) HandleRetailerRequestToMD(w http.ResponseWriter, r *http.Request) {
	fh.handleCreate(w, r, "fund request from retailer to master distributor", "R", "M")
}

// Fund Request From Retailer to Distributor Handler
func (fh *FundRequestHandler) HandleRetailerRequestToDistributor(w http.ResponseWriter, r *http.Request) {
	fh.handleCreate(w, r, "fund request from retailer to distributor", "R", "D")
}

// Common Create Function
func (fh *FundRequestHandler) handleCreate(
	w http.ResponseWriter,
	r *http.Request,
	op, requesterPrefix, requestToPrefix string,
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

	if err := fh.fundRequestStore.CreateFundRequest(&req); err != nil {
		utils.ServerError(w, fh.logger, op, err)
		return
	}

	utils.WriteJSON(w, http.StatusCreated, utils.Envelope{"message": "fund request created successfully", "fund_request": req})
}

// Approve Fund Request Handler
func (fh *FundRequestHandler) HandleApproveFundRequest(w http.ResponseWriter, r *http.Request) {
	id, err := utils.ReadParamIDInt(r)
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

// Reject Fund Request Handler
func (fh *FundRequestHandler) HandleRejectFundRequest(w http.ResponseWriter, r *http.Request) {
	id, err := utils.ReadParamIDInt(r)
	if err != nil {
		utils.BadRequest(w, fh.logger, "reject fund request", err)
		return
	}

	var req models.FundRequestModel
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		utils.BadRequest(w, fh.logger, "reject fund request", err)
		return
	}
	if req.RejectRemarks == nil || *req.RejectRemarks == "" {
		utils.BadRequest(w, fh.logger, "reject fund request", errors.New("reject_remarks is required"))
		return
	}

	req.FundRequestID = id
	if err := fh.fundRequestStore.RejectFundRequest(&req); err != nil {
		if isFundRequestClientErr(err) {
			utils.BadRequest(w, fh.logger, "reject fund request", err)
			return
		}
		utils.ServerError(w, fh.logger, "reject fund request", err)
		return
	}

	utils.WriteJSON(w, http.StatusOK, utils.Envelope{"message": "fund request rejected successfully"})
}

// Get Fund Request By Requester ID Handler
func (fh *FundRequestHandler) HandleGetFundRequestsByRequesterID(w http.ResponseWriter, r *http.Request) {
	id, err := utils.ReadParamID(r)
	if err != nil {
		utils.BadRequest(w, fh.logger, "get fund requests by requester id", err)
		return
	}

	requests, err := fh.fundRequestStore.GetFundRequestsByRequesterID(id, utils.ReadQueryParams(r))
	if err != nil {
		utils.ServerError(w, fh.logger, "get fund requests by requester id", err)
		return
	}

	utils.WriteJSON(w, http.StatusOK, utils.Envelope{"message": "fund requests fetched successfully", "fund_requests": requests})
}

// Get Fund Request By Request To ID Handler
func (fh *FundRequestHandler) HandleGetFundRequestsByRequestToID(w http.ResponseWriter, r *http.Request) {
	id, err := utils.ReadParamID(r)
	if err != nil {
		utils.BadRequest(w, fh.logger, "get fund requests by request_to id", err)
		return
	}

	requests, err := fh.fundRequestStore.GetFundRequestsByRequestToID(id, utils.ReadQueryParams(r))
	if err != nil {
		utils.ServerError(w, fh.logger, "get fund requests by request_to id", err)
		return
	}

	utils.WriteJSON(w, http.StatusOK, utils.Envelope{"message": "fund requests fetched successfully", "fund_requests": requests})
}

// Get All Fund Request Handler
func (fh *FundRequestHandler) HandleGetAllFundRequests(w http.ResponseWriter, r *http.Request) {
	requests, err := fh.fundRequestStore.GetAllFundRequests(utils.ReadQueryParams(r))
	if err != nil {
		utils.ServerError(w, fh.logger, "get all fund requests", err)
		return
	}

	utils.WriteJSON(w, http.StatusOK, utils.Envelope{"message": "fund requests fetched successfully", "fund_requests": requests})
}

// Upload Fund Request Recipt Handler
func (fh *FundRequestHandler) HandleUploadFundRequestRecipt(w http.ResponseWriter, r *http.Request) {
	id, err := utils.ReadParamIDInt(r)
	if err != nil {
		utils.BadRequest(w, fh.logger, "upload fund request recipt", err)
		return
	}

	var req struct {
		FileExtension string `json:"file_extension"`
	}
	if err = json.NewDecoder(r.Body).Decode(&req); err != nil {
		utils.BadRequest(w, fh.logger, "upload fund request recipt", err)
		return
	}
	if req.FileExtension == "" {
		req.FileExtension = "pdf"
	}

	key := fmt.Sprintf("fund-requests/%d/%d_recipt_%d.%s", id, id, time.Now().Unix(), req.FileExtension)
	url, err := fh.awss3.GenerateUploadPresignedURL(key)
	if err != nil {
		utils.ServerError(w, fh.logger, "upload fund request recipt", err)
		return
	}

	if err = fh.fundRequestStore.UploadFundRequestRecipt(id, key); err != nil {
		if isFundRequestClientErr(err) {
			utils.BadRequest(w, fh.logger, "upload fund request recipt", err)
			return
		}
		utils.ServerError(w, fh.logger, "upload fund request recipt", err)
		return
	}

	utils.WriteJSON(w, http.StatusOK, utils.Envelope{
		"message": "fund request recipt upload url generated successfully",
		"url":     url,
		"path":    key,
	})
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

// Error Helper Function
func isFundRequestClientErr(err error) bool {
	msg := err.Error()
	return msg == "fund request not found" ||
		msg == "fund request not found or already processed" ||
		msg == "fund request not found or not a NORMAL type request" ||
		msg == "insufficient balance" ||
		msg == "requester not found" ||
		msg == "request_to user not found" ||
		strings.HasPrefix(msg, "fund request is already ")
}
