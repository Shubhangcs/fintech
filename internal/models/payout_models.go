package models

import (
	"errors"
	"time"
)

var validPayoutStatuses = map[string]bool{
	"SUCCESS": true,
	"PENDING": true,
	"FAILED":  true,
	"REFUND":  true,
}

var validTransferTypes = map[string]bool{
	"IMPS": true,
	"NEFT": true,
}

type PayoutTransactionModel struct {
	PayoutTransactionID        string    `json:"payout_transaction_id"`
	PartnerRequestID           string    `json:"partner_request_id"`
	OperatorTransactionID      string    `json:"operator_transaction_id"`
	RetailerID                 string    `json:"retailer_id"`
	OrderID                    string    `json:"order_id"`
	MobileNumber               string    `json:"mobile_number"`
	BankName                   string    `json:"bank_name"`
	BeneficiaryName            string    `json:"beneficiary_name"`
	AccountNumber              string    `json:"account_number"`
	IFSCCode                   string    `json:"ifsc_code"`
	Amount                     float64   `json:"amount"`
	TransferType               string    `json:"transfer_type"`
	AdminCommision             float64   `json:"admin_commision"`
	MasterDistributorCommision float64   `json:"master_distributor_commision"`
	DistributorCommision       float64   `json:"distributor_commision"`
	RetailerCommision          float64   `json:"retailer_commision"`
	PayoutTransactionStatus    string    `json:"payout_transaction_status"`
	RetailerName               string    `json:"retailer_name,omitempty"`
	RetailerBusinessName       *string   `json:"retailer_business_name,omitempty"`
	BeforeBalance              float64   `json:"before_balance"`
	AfterBalance               float64   `json:"after_balance"`
	CreatedAT                  time.Time `json:"created_at"`
	UpdatedAT                  time.Time `json:"updated_at"`
}

func IsValidPayoutStatus(status string) bool {
	return validPayoutStatuses[status]
}

func (pt *PayoutTransactionModel) ValidateInitilizePayout() error {
	if pt.RetailerID == "" {
		return errors.New("retailer_id is required")
	}
	if pt.MobileNumber == "" {
		return errors.New("mobile_number is required")
	}
	if pt.BankName == "" {
		return errors.New("bank_name is required")
	}
	if pt.BeneficiaryName == "" {
		return errors.New("beneficiary_name is required")
	}
	if pt.AccountNumber == "" {
		return errors.New("account_number is required")
	}
	if pt.IFSCCode == "" {
		return errors.New("ifsc_code is required")
	}
	if pt.Amount <= 0 {
		return errors.New("amount must be greater than 0")
	}
	if !validTransferTypes[pt.TransferType] {
		return errors.New("transfer_type must be IMPS or NEFT")
	}
	return nil
}

type FinilizePayoutModel struct {
	PayoutTransactionID   string  `json:"payout_transaction_id"`
	OperatorTransactionID *string `json:"optransid"`
	OrderID               string  `json:"orderid"`
	Status                string  `json:"status"`
}

func (pt *FinilizePayoutModel) ValidateFinilizePayoutModel() error {
	if pt.PayoutTransactionID == "" {
		return errors.New("payout transaction id is required")
	}
	if pt.Status == "" || !validPayoutStatuses[pt.Status] {
		return errors.New("invalid payout status")
	}
	return nil
}

type PayoutAPIResponseModel struct {
	Error                 int    `json:"error"`
	Message               string `json:"msg"`
	Status                int    `json:"status"`
	OrderID               string `json:"orderid"`
	OperatorTransactionID string `json:"optransid"`
	PartnerRequestID      string `json:"partnerreqid"`
}
