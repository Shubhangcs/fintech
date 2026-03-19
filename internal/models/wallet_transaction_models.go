package models

import (
	"errors"
	"time"
)

var validTransactionReasons = map[string]bool{
	"FUND_TRANSFER":           true,
	"FUND_REQUEST":            true,
	"MOBILE_RECHARGE":         true,
	"POSTPAID_MOBILE_RECHARGE": true,
	"MOBILE_RECHARGE_REFUND":  true,
	"DTH_RECHARGE_REFUND":     true,
	"PAYOUT_REFUND":           true,
	"DTH_RECHARGE":            true,
	"TOPUP":                   true,
	"REVERT":                  true,
	"PAYOUT":                  true,
}

type WalletTransactionModel struct {
	WalletTransactionID int64     `json:"wallet_transaction_id"`
	UserID              string    `json:"user_id"`
	ReferenceID         string    `json:"reference_id"`
	CreditAmount        *float64  `json:"credit_amount"`
	DebitAmount         *float64  `json:"debit_amount"`
	BeforeBalance       float64   `json:"before_balance"`
	AfterBalance        float64   `json:"after_balance"`
	TransactionReason   string    `json:"transaction_reason"`
	Remarks             string    `json:"remarks"`
	UserName            string    `json:"user_name,omitempty"`
	UserBusinessName    *string   `json:"user_business_name,omitempty"`
	CreatedAT           time.Time `json:"created_at"`
}

func (wt *WalletTransactionModel) ValidateCreateWalletTransaction() error {
	if wt.UserID == "" {
		return errors.New("user_id is required")
	}

	if wt.ReferenceID == "" {
		return errors.New("reference_id is required")
	}

	if wt.CreditAmount == nil && wt.DebitAmount == nil {
		return errors.New("either credit_amount or debit_amount is required")
	}

	if wt.CreditAmount != nil && wt.DebitAmount != nil {
		return errors.New("only one of credit_amount or debit_amount can be set")
	}

	if wt.CreditAmount != nil && *wt.CreditAmount <= 0 {
		return errors.New("credit_amount must be greater than 0")
	}

	if wt.DebitAmount != nil && *wt.DebitAmount <= 0 {
		return errors.New("debit_amount must be greater than 0")
	}

	if !validTransactionReasons[wt.TransactionReason] {
		return errors.New("invalid transaction_reason")
	}

	if wt.Remarks == "" {
		return errors.New("remarks is required")
	}

	return nil
}
