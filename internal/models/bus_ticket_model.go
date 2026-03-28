package models

type BusStationModel struct {
	Success  int            `json:"success"`
	Stations []StationModel `json:"stations"`
}

type StationModel struct {
	SourceID    string `json:"Source_ID"`
	StationName string `json:"Station_Name"`
}


