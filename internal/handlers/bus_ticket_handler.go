package handlers

import (
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"

	"github.com/google/uuid"
	"github.com/levionstudio/fintech/internal/models"
	"github.com/levionstudio/fintech/internal/utils"
)

type BusHandler struct {
	logger *slog.Logger
}

func NewBusHandler(logger *slog.Logger) *BusHandler {
	return &BusHandler{logger: logger}
}

// POST /bus/block-tickets
func (bh *BusHandler) HandleBlockBusTicket(w http.ResponseWriter, r *http.Request) {
	var req models.BusBlockTicketRequestModel
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		utils.BadRequest(w, bh.logger, "block bus ticket", err)
		return
	}

	if req.OperatorID == "" || req.ServiceID == "" || req.SourceStationID == "" ||
		req.DestinationStationID == "" || req.JourneyDate == "" || req.BoardingPointID == "" ||
		req.DroppingPointID == "" || req.ContactNumber == "" || req.EmailID == "" ||
		len(req.NamesList) == 0 || len(req.SeatNumbersList) == 0 {
		utils.BadRequest(w, bh.logger, "block bus ticket", fmt.Errorf("required fields are missing"))
		return
	}

	req.PartnerRequestID = uuid.NewString()

	var resp models.BusBlockTicketResponseModel
	err := utils.PostRequest(utils.RechargeKitAPI2+"/bus/blockTickets", "Authorization", bh.busAuthHeader(), map[string]any{
		"operator_Id":          req.OperatorID,
		"partner_request_Id":   req.PartnerRequestID,
		"boardingPoint_ID":     req.BoardingPointID,
		"service_Id":           req.ServiceID,
		"sourceStation_Id":     req.SourceStationID,
		"destinationStation_Id": req.DestinationStationID,
		"journeyDate":          req.JourneyDate,
		"layout_Id":            req.LayoutID,
		"droppingPoint_ID":     req.DroppingPointID,
		"address":              req.Address,
		"contactNumber":        req.ContactNumber,
		"email_Id":             req.EmailID,
		"namesList":            req.NamesList,
		"gendersList":          req.GendersList,
		"ageList":              req.AgeList,
		"seatNumbersList":      req.SeatNumbersList,
		"seatFareList":         req.SeatFareList,
		"seatTypeIds":          req.SeatTypeIds,
		"isAcSeat":             req.IsAcSeat,
		"serviceTaxList":       req.ServiceTaxList,
		"seatLayoutUnique_Id":  req.SeatLayoutUniqueID,
		"isSingleLady":         req.IsSingleLady,
		"additionalInfoLabel":  req.AdditionalInfoLabel,
	}, &resp)
	if err != nil {
		utils.ServerError(w, bh.logger, "block bus ticket", err)
		return
	}

	utils.WriteJSON(w, http.StatusOK, utils.Envelope{"response": resp})
}

func (bh *BusHandler) busAuthHeader() string {
	return "Bearer " + utils.RechargeKitAPIToken
}

// GET /bus/stations
func (bh *BusHandler) HandleGetBusStations(w http.ResponseWriter, r *http.Request) {
	var resp models.BusStationResponseModel
	err := utils.GetRequest(utils.RechargeKitAPI2+"/bus/stations", "Authorization", bh.busAuthHeader(), &resp)
	if err != nil {
		utils.ServerError(w, bh.logger, "get bus stations", err)
		return
	}
	utils.WriteJSON(w, http.StatusOK, utils.Envelope{"response": resp})
}

// GET /bus/operators
func (bh *BusHandler) HandleGetBusOperators(w http.ResponseWriter, r *http.Request) {
	var resp models.BusOperatorResponseModel
	err := utils.GetRequest(utils.RechargeKitAPI2+"/bus/operators", "Authorization", bh.busAuthHeader(), &resp)
	if err != nil {
		utils.ServerError(w, bh.logger, "get bus operators", err)
		return
	}
	utils.WriteJSON(w, http.StatusOK, utils.Envelope{"response": resp})
}

// POST /bus/available-services
func (bh *BusHandler) HandleGetAvailableServices(w http.ResponseWriter, r *http.Request) {
	var req models.BusAvailableServiceRequestModel
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		utils.BadRequest(w, bh.logger, "get available bus services", err)
		return
	}

	if req.SourceStationID == "" || req.DestinationStationID == "" || req.JourneyDate == "" {
		utils.BadRequest(w, bh.logger, "get available bus services", fmt.Errorf("sourceStation_Id, destinationStationId and journeyDate are required"))
		return
	}

	var resp json.RawMessage
	err := utils.PostRequest(utils.RechargeKitAPI2+"/bus/searchService", "Authorization", bh.busAuthHeader(), map[string]any{
		"sourceStationId":      req.SourceStationID,
		"destinationStationId": req.DestinationStationID,
		"journeyDate":          req.JourneyDate,
	}, &resp)
	if err != nil {
		utils.ServerError(w, bh.logger, "get available bus services", err)
		return
	}
	utils.WriteJSON(w, http.StatusOK, utils.Envelope{"response": resp})
}

// POST /bus/seat-map
func (bh *BusHandler) HandleGetServiceSeatingLayout(w http.ResponseWriter, r *http.Request) {
	var req models.BusSeatingLayoutRequestModel
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		utils.BadRequest(w, bh.logger, "get bus seating layout", err)
		return
	}
	req.PartnerRequestID = uuid.NewString()

	if req.OperatorID == "" || req.ServiceID == "" || req.SourceStationID == "" || req.DestinationStationID == "" || req.JourneyDate == "" {
		utils.BadRequest(w, bh.logger, "get bus seating layout", fmt.Errorf("operator_Id, service_Id, sourceStation_Id, destinationStation_Id and journeyDate are required"))
		return
	}

	var resp json.RawMessage
	err := utils.PostRequest(utils.RechargeKitAPI2+"/bus/seatMap", "Authorization", bh.busAuthHeader(), map[string]any{
		"operatorId":          req.OperatorID,
		"serviceId":           req.ServiceID,
		"sourceStationId":      req.SourceStationID,
		"destinationStationId": req.DestinationStationID,
		"journeyDate":          req.JourneyDate,
		"layoutId":             req.LayoutID,
		"seatFare":             req.SeatFare,
		"partnerreqid":         req.PartnerRequestID,
	}, &resp)
	if err != nil {
		utils.ServerError(w, bh.logger, "get bus seating layout", err)
		return
	}
	utils.WriteJSON(w, http.StatusOK, utils.Envelope{"response": resp})
}
