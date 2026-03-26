package models

import (
	"errors"
	"time"
)

var validRechargeStatuses = map[string]bool{
	"SUCCESS": true,
	"PENDING": true,
	"FAILED":  true,
	"REFUND":  true,
}

var validateRechargeType = map[string]bool{
	"PREPAID":  true,
	"POSTPAID": true,
}

func IsValidRechargeStatus(status string) bool {
	return validRechargeStatuses[status]
}

type MobileRechargeModel struct {
	MobileRechargeTransactionID int64     `json:"mobile_recharge_transaction_id"`
	RetailerID                  string    `json:"retailer_id"`
	PartnerRequestID            string    `json:"partner_request_id"`
	MobileNumber                string    `json:"mobile_number"`
	OperatorName                string    `json:"operator_name"`
	CircleName                  string    `json:"circle_name"`
	OperatorCode                int       `json:"operator_code"`
	CircleCode                  int       `json:"circle_code"`
	Amount                      float64   `json:"amount"`
	Commision                   float64   `json:"commision"`
	RechargeType                string    `json:"recharge_type"`
	OperatorTransactionID       string    `json:"operator_transaction_id"`
	OrderID                     string    `json:"order_id"`
	RechargeStatus              string    `json:"recharge_status"`
	CreatedAt                   time.Time `json:"created_at"`
	RetailerName                string    `json:"retailer_name,omitempty"`
	RetailerBusinessName        *string   `json:"retailer_business_name,omitempty"`
	BeforeBalance               float64   `json:"before_balance"`
	AfterBalance                float64   `json:"after_balance"`
}

func (mr *MobileRechargeModel) ValidateInitializeMobileRecharge() error {
	if mr.RetailerID == "" {
		return errors.New("retailer_id is required")
	}
	if mr.MobileNumber == "" {
		return errors.New("mobile_number is required")
	}
	if mr.OperatorCode == 0 {
		return errors.New("operator_code is required")
	}
	if mr.CircleCode == 0 {
		return errors.New("circle_code is required")
	}
	if mr.Amount <= 0 {
		return errors.New("amount must be greater than 0")
	}
	if mr.OperatorName == "" {
		return errors.New("operator_name is required")
	}
	if mr.CircleName == "" {
		return errors.New("circle_name is required")
	}
	if !validateRechargeType[mr.RechargeType] {
		return errors.New("recharge_type must be PREPAID or POSTPAID")
	}
	return nil
}

type PrepaidPlanFetchResponseModel struct {
	Error    int    `json:"error"`
	Message  string `json:"msg"`
	Status   int    `json:"status"`
	PlanData any    `json:"planData"`
}

type PostpaidBillFetchRequest struct {
	MobileNumber string `json:"mobile_no"`
	OperatorCode int    `json:"operator_code"`
}

type PostpaidBillFetchResponse struct {
	Error      int    `json:"error"`
	Status     int    `json:"status"`
	Message    string `json:"msg"`
	BillAmount any    `json:"billAmount"`
}

type MobileRechargeCircleModel struct {
	CircleCode int    `json:"circle_code"`
	CircleName string `json:"circle_name"`
}

type MobileRechargeOperatorModel struct {
	OperatorCode int    `json:"operator_code"`
	OperatorName string `json:"operator_name"`
}
