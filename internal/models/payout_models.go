package models

import (
	"errors"
	"time"
)

type CreatePayoutRequest struct {
	RetailerID      string  `json:"retailer_id"`
	MobileNumber    string  `json:"mobile_number"`
	BankName        string  `json:"bank_name"`
	BeneficiaryName string  `json:"beneficiary_name"`
	AccountNumber   string  `json:"account_number"`
	IFSCCode        string  `json:"ifsc_code"`
	Amount          float64 `json:"amount"`
	TransferType    int     `json:"transfer_type"` // 5=IMPS 6=NEFT
}

func (r *CreatePayoutRequest) Validate() error {
	if r.RetailerID == "" {
		return errors.New("retailer_id is required")
	}
	if r.MobileNumber == "" {
		return errors.New("mobile_number is required")
	}
	if r.BankName == "" {
		return errors.New("bank_name is required")
	}
	if r.BeneficiaryName == "" {
		return errors.New("beneficiary_name is required")
	}
	if r.AccountNumber == "" {
		return errors.New("account_number is required")
	}
	if r.IFSCCode == "" {
		return errors.New("ifsc_code is required")
	}
	if r.Amount < 1000 {
		return errors.New("minimum amount is 1000")
	}
	if r.TransferType != 5 && r.TransferType != 6 {
		return errors.New("transfer_type must be 5 (IMPS) or 6 (NEFT)")
	}
	return nil
}

// PayoutCommision holds the calculated commision amounts for a payout.
type PayoutCommision struct {
	Total             float64
	Admin             float64
	MasterDistributor float64
	Distributor       float64
	Retailer          float64
}

type PayoutTransactionModel struct {
	PayoutTransactionID         string    `json:"payout_transaction_id"`
	PartnerRequestID            string    `json:"partner_request_id"`
	OperatorTransactionID       string    `json:"operator_transaction_id"`
	OrderID                     string    `json:"order_id"`
	RetailerID                  string    `json:"retailer_id"`
	RetailerName                string    `json:"retailer_name,omitempty"`
	RetailerBusinessName        *string   `json:"retailer_business_name,omitempty"`
	MobileNumber                string    `json:"mobile_number"`
	BankName                    string    `json:"bank_name"`
	BeneficiaryName             string    `json:"beneficiary_name"`
	AccountNumber               string    `json:"account_number"`
	IFSCCode                    string    `json:"ifsc_code"`
	Amount                      float64   `json:"amount"`
	TransferType                string    `json:"transfer_type"`
	AdminCommision             float64   `json:"admin_commision"`
	MasterDistributorCommision float64   `json:"master_distributor_commision"`
	DistributorCommision       float64   `json:"distributor_commision"`
	RetailerCommision          float64   `json:"retailer_commision"`
	BeforeBalance               float64   `json:"before_balance"`
	AfterBalance                float64   `json:"after_balance"`
	TransactionStatus           string    `json:"payout_transaction_status"`
	CreatedAT                   time.Time `json:"created_at"`
	UpdatedAT                   time.Time `json:"updated_at"`
}

type UpdatePayoutTransactionRequest struct {
	OperatorTransactionID string `json:"operator_transaction_id"`
	OrderID               string `json:"order_id"`
	Status                string `json:"payout_transaction_status"`
}
