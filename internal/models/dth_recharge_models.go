package models

import (
	"errors"
	"time"
)

type DTHRechargeModel struct {
	DTHTransactionID     int64     `json:"dth_transaction_id"`
	RetailerID           string    `json:"retailer_id"`
	PartnerRequestID     string    `json:"partner_request_id"`
	CustomerID           string    `json:"customer_id"`
	OperatorName         string    `json:"operator_name"`
	OperatorCode         int       `json:"operator_code"`
	Amount               float64   `json:"amount"`
	Commision            float64   `json:"commision"`
	Status               string    `json:"status"`
	CreatedAt            time.Time `json:"created_at"`
	RetailerName         string    `json:"retailer_name,omitempty"`
	RetailerBusinessName *string   `json:"retailer_business_name,omitempty"`
	BeforeBalance        float64   `json:"before_balance"`
	AfterBalance         float64   `json:"after_balance"`
}

func (dr *DTHRechargeModel) Validate() error {
	if dr.RetailerID == "" {
		return errors.New("retailer_id is required")
	}
	if dr.CustomerID == "" {
		return errors.New("customer_id is required")
	}
	if dr.OperatorCode == 0 {
		return errors.New("operator_code is required")
	}
	if dr.OperatorName == "" {
		return errors.New("operator_name is required")
	}
	if dr.Amount <= 0 {
		return errors.New("amount must be greater than 0")
	}
	return nil
}

type DTHRechargeOperatorModel struct {
	OperatorCode int    `json:"operator_code"`
	OperatorName string `json:"operator_name"`
}
