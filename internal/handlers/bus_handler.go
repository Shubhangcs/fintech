package handlers

import (
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"

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
	var resp json.RawMessage
	err := utils.PostRequest(utils.RechargeKitAPI2+"/bus/stations", "Authorization", bh.busAuthHeader(), map[string]any{}, &resp)
	if err != nil {
		utils.ServerError(w, bh.logger, "get bus stations", err)
		return
	}
	utils.WriteJSON(w, http.StatusOK, utils.Envelope{"data": resp})
}

// GET /bus/operators
func (bh *BusHandler) HandleGetBusOperators(w http.ResponseWriter, r *http.Request) {
	var resp json.RawMessage
	err := utils.GetRequest(utils.RechargeKitAPI2+"/bus/operators", "Authorization", bh.busAuthHeader(), &resp)
	if err != nil {
		utils.ServerError(w, bh.logger, "get bus operators", err)
		return
	}
	utils.WriteJSON(w, http.StatusOK, utils.Envelope{"data": resp})
}

// GET /bus/available-services?sourceStationId=&destinationStationId=&journeyDate=
func (bh *BusHandler) HandleGetAvailableServices(w http.ResponseWriter, r *http.Request) {
	sourceStationID := r.URL.Query().Get("sourceStationId")
	destinationStationID := r.URL.Query().Get("destinationStationId")
	journeyDate := r.URL.Query().Get("journeyDate")

	if sourceStationID == "" || destinationStationID == "" || journeyDate == "" {
		utils.BadRequest(w, bh.logger, "get available bus services", fmt.Errorf("sourceStationId, destinationStationId and journeyDate are required"))
		return
	}

	url := fmt.Sprintf("%s/bus/searchService?sourceStationId=%s&destinationStationId=%s&journeyDate=%s",
		utils.RechargeKitAPI2, sourceStationID, destinationStationID, journeyDate)

	var resp json.RawMessage
	err := utils.GetRequest(url, "Authorization", bh.busAuthHeader(), &resp)
	if err != nil {
		utils.ServerError(w, bh.logger, "get available bus services", err)
		return
	}
	utils.WriteJSON(w, http.StatusOK, utils.Envelope{"data": resp})
}

// GET /bus/seat-map?operatorId=&serviceId=&sourceStationId=&destinationStationId=&journeyDate=
func (bh *BusHandler) HandleGetServiceSeatingLayout(w http.ResponseWriter, r *http.Request) {
	operatorID := r.URL.Query().Get("operatorId")
	serviceID := r.URL.Query().Get("serviceId")
	sourceStationID := r.URL.Query().Get("sourceStationId")
	destinationStationID := r.URL.Query().Get("destinationStationId")
	journeyDate := r.URL.Query().Get("journeyDate")

	if operatorID == "" || serviceID == "" || sourceStationID == "" || destinationStationID == "" || journeyDate == "" {
		utils.BadRequest(w, bh.logger, "get bus seating layout", fmt.Errorf("operatorId, serviceId, sourceStationId, destinationStationId and journeyDate are required"))
		return
	}

	url := fmt.Sprintf("%s/bus/seatMap?operatorId=%s&serviceId=%s&sourceStationId=%s&destinationStationId=%s&journeyDate=%s",
		utils.RechargeKitAPI2, operatorID, serviceID, sourceStationID, destinationStationID, journeyDate)

	var resp json.RawMessage
	err := utils.GetRequest(url, "Authorization", bh.busAuthHeader(), &resp)
	if err != nil {
		utils.ServerError(w, bh.logger, "get bus seating layout", err)
		return
	}
	utils.WriteJSON(w, http.StatusOK, utils.Envelope{"data": resp})
}
