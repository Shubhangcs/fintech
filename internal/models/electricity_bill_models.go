package models

import (
	"errors"
	"time"
)

type ElectricityBillModel struct {
	ElectricityBillTransactionID int64     `json:"electricity_bill_transaction_id"`
	RetailerID                   string    `json:"retailer_id"`
	OrderID                      *string   `json:"order_id"`
	OperatorTransactionID        *string   `json:"operator_transaction_id"`
	PartnerRequestID             string    `json:"partner_request_id"`
	CustomerID                   string    `json:"customer_id"`
	Amount                       float64   `json:"amount"`
	OperatorCode                 int       `json:"operator_code"`
	OperatorName                 string    `json:"operator_name"`
	CustomerEmail                string    `json:"customer_email"`
	Commision                    float64   `json:"commision"`
	TransactionStatus            string    `json:"transaction_status"`
	CreatedAt                    time.Time `json:"created_at"`
	RetailerName                 string    `json:"retailer_name,omitempty"`
	RetailerBusinessName         *string   `json:"retailer_business_name,omitempty"`
	BeforeBalance                float64   `json:"before_balance"`
	AfterBalance                 float64   `json:"after_balance"`
}

func (eb *ElectricityBillModel) Validate() error {
	if eb.RetailerID == "" {
		return errors.New("retailer_id is required")
	}
	if eb.CustomerID == "" {
		return errors.New("customer_id is required")
	}
	if eb.OperatorCode == 0 {
		return errors.New("operator_code is required")
	}
	if eb.OperatorName == "" {
		return errors.New("operator_name is required")
	}
	if eb.CustomerEmail == "" {
		return errors.New("customer_email is required")
	}
	if eb.Amount <= 0 {
		return errors.New("amount must be greater than 0")
	}
	return nil
}

type ElectricityOperatorModel struct {
	OperatorCode int    `json:"operator_code"`
	OperatorName string `json:"operator_name"`
}

type ElectricityBillFetchRequest struct {
	ConsumerID   string `json:"consumer_id"`
	OperatorCode int    `json:"operator_code"`
}

type ElectricityBillFetchResponse struct {
	Error      int    `json:"error"`
	Status     int    `json:"status"`
	Message    string `json:"msg"`
	BillAmount any    `json:"billAmount"`
}
