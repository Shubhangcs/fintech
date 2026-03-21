package models

import (
	"errors"
	"time"
)

type FundTransferModel struct {
	FundTransferID             int64     `json:"fund_transfer_id"`
	FundTransfererID           string    `json:"fund_transferer_id"`
	FundReceiverID             string    `json:"fund_receiver_id"`
	Amount                     float64   `json:"amount"`
	FundTransferStatus         string    `json:"fund_transfer_status"`
	Remarks                    string    `json:"remarks"`
	BeforeBalance              float64   `json:"before_balance"`
	AfterBalance               float64   `json:"after_balance"`
	TransfererName             string    `json:"transferer_name,omitempty"`
	TransfererBusinessName     *string   `json:"transferer_business_name,omitempty"`
	ReceiverName               string    `json:"receiver_name,omitempty"`
	ReceiverBusinessName       *string   `json:"receiver_business_name,omitempty"`
	CreatedAT                  time.Time `json:"created_at"`
}

func (ft *FundTransferModel) Validate() error {
	if ft.FundTransfererID == "" {
		return errors.New("fund_transferer_id is required")
	}

	if ft.FundReceiverID == "" {
		return errors.New("fund_receiver_id is required")
	}

	if ft.FundTransfererID == ft.FundReceiverID {
		return errors.New("sender and receiver cannot be the same")
	}

	if ft.Amount <= 0 {
		return errors.New("amount must be greater than 0")
	}

	return nil
}
