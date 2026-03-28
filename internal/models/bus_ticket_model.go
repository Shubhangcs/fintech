package models

import "time"

type BusStationModel struct {
	Success  int            `json:"success"`
	Stations []StationModel `json:"stations"`
}

type StationModel struct {
	SourceID    string `json:"Source_ID"`
	StationName string `json:"Station_Name"`
}

type AvailableServiceRequestModel struct {
	SourceStationID      string    `json:"sourceStation_Id"`
	DestinationStationID string    `json:"destinationStationId"`
	JourneyDate          time.Time `json:"journeyDate"`
}

type AvailableServiceResponseModel struct {
}
