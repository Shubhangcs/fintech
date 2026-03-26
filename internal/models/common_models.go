package models

type DropdownItem struct {
	ID   string `json:"id"`
	Name string `json:"name"`
}

type APIResponseModel struct {
	Error                 int    `json:"error"`
	Message               string `json:"msg"`
	Status                int    `json:"status"`
	OrderID               string `json:"orderid"`
	OperatorTransactionID string `json:"optransid"`
	PartnerRequestID      string `json:"partnerreqid"`
}
