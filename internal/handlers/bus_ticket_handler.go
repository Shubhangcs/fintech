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

	if req.OperatorID == 0 || req.ServiceID == "" || req.SourceStationID == 0 || req.DestinationStationID == 0 || req.JourneyDate == "" {
		utils.BadRequest(w, bh.logger, "get bus seating layout", fmt.Errorf("operator_Id, service_Id, sourceStation_Id, destinationStation_Id and journeyDate are required"))
		return
	}

	var resp json.RawMessage
	err := utils.PostRequest(utils.RechargeKitAPI2+"/bus/seatMap", "Authorization", bh.busAuthHeader(), map[string]any{
		"operator_Id":           req.OperatorID,
		"service_Id":            req.ServiceID,
		"sourceStation_Id":      req.SourceStationID,
		"destinationStation_Id": req.DestinationStationID,
		"journeyDate":           req.JourneyDate,
		"layoutId":              req.LayoutID,
		"seatFare":              req.SeatFare,
		"partnerreqid":          req.PartnerRequestID,
	}, &resp)
	if err != nil {
		utils.ServerError(w, bh.logger, "get bus seating layout", err)
		return
	}
	utils.WriteJSON(w, http.StatusOK, utils.Envelope{"response": resp})
}
