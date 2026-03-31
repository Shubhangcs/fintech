package models

type BusStationResponseModel struct {
	Success  int               `json:"success"`
	Stations []BusStationModel `json:"stations"`
}

type BusStationModel struct {
	SourceID    string `json:"Source_ID"`
	StationName string `json:"Station_Name"`
}

type BusOperatorResponseModel struct {
	Error        int                `json:"error"`
	Message      string             `json:"msg"`
	Status       int                `json:"status"`
	OperatorInfo []BusOperatorModel `json:"operatorinfo"`
}

type BusOperatorModel struct {
	OperatorID          string `json:"operator_id"`
	OperatorName        string `json:"operater_name"`
	PartialCancellation string `json:"partialCancellation"`
}

type BusSeatingLayoutRequestModel struct {
	OperatorID           string `json:"operator_id"`
	ServiceID            string `json:"service_id"`
	SourceStationID      string `json:"source_station_id"`
	DestinationStationID string `json:"destination_station_id"`
	JourneyDate          string `json:"journey_date"`
	LayoutID             int    `json:"layout_id"`
	SeatFare             int    `json:"seat_fare"`
	PartnerRequestID     string `json:"partner_request_id"`
}

type BusAvailableServiceRequestModel struct {
	SourceStationID      string `json:"source_station_id"`
	DestinationStationID string `json:"destination_station_id"`
	JourneyDate          string `json:"journey_date"`
}

type BusAvailableServiceResponseModel struct {
	Error    int                   `json:"error"`
	Message  string                `json:"msg"`
	Status   int                   `json:"status"`
	Services []BusAvailableService `json:"services"`
}

type BusAvailableService struct {
	OperatorID         string   `json:"operatorId"`
	ServiceKey         string   `json:"Service_key"`
	ServiceName        string   `json:"Service_Name"`
	ServiceNumber      string   `json:"Service_Number"`
	TravelerAgentName  string   `json:"Traveler_Agent_Name"`
	BusTypeName        string   `json:"Bus_Type_Name"`
	StartTime          string   `json:"Start_time"`
	ArrTime            string   `json:"Arr_Time"`
	TravelTime         string   `json:"TravelTime"`
	SourceID           int      `json:"Source_ID"`
	DestinationID      int      `json:"Destination_ID"`
	Fare               float64  `json:"Fare"`
	AvailableSeats     string   `json:"available_seats"`
	JDate              string   `json:"jdate"`
	BusStartDate       string   `json:"BUS_START_DATE"`
	LayoutID           int      `json:"layout_id"`
	Amenities          string   `json:"Amenities"`
	BoardingInfo       []string `json:"boarding_info"`
	DroppingInfo       []string `json:"dropping_info"`
	CancellationPolicy string   `json:"Cancellationpolicy"`
	BusType            string   `json:"bus_type"`
	IsBordDropFirst    string   `json:"isBordDropFirst"`
	IsSingleLady       string   `json:"isSingleLady"`
	AllowedConcessions []any    `json:"allowedConcessions"`
}
