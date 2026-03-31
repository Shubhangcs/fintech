package models

type ApiDownModel struct {
	ApiDownID   int64  `json:"api_down_id,omitempty"`
	ServiceName string `json:"service_name"`
	Status      bool   `json:"status"`
	CreatedAt   string `json:"created_at,omitempty"`
	UpdatedAt   string `json:"updated_at,omitempty"`
}

const (
	ServicePayout          = "payout"
	ServiceMobileRecharge  = "mobile_recharge"
	ServiceDTHRecharge     = "dth_recharge"
	ServiceElectricityBill = "electricity_bill"
	ServiceBusTicket       = "bus_ticket"
)
