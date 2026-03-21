package models

import (
	"errors"
	"time"
)

type RevertTransactionModel struct {
	RevertTransactionID  int64     `json:"revert_transaction_id"`
	RevertByID           string    `json:"revert_by_id"`
	RevertOnID           string    `json:"revert_on_id"`
	Amount               float64   `json:"amount"`
	RevertStatus         string    `json:"revert_status"`
	Remarks              string    `json:"remarks"`
	RevertByName         string    `json:"revert_by_name,omitempty"`
	RevertByBusinessName *string   `json:"revert_by_business_name,omitempty"`
	RevertOnName         string    `json:"revert_on_name,omitempty"`
	RevertOnBusinessName *string   `json:"revert_on_business_name,omitempty"`
	CreatedAT            time.Time `json:"created_at"`
}

func (rt *RevertTransactionModel) Validate() error {
	if rt.RevertByID == "" {
		return errors.New("revert_by_id is required")
	}
	if rt.RevertOnID == "" {
		return errors.New("revert_on_id is required")
	}
	if rt.RevertByID == rt.RevertOnID {
		return errors.New("revert_by and revert_on cannot be the same")
	}
	if rt.Amount <= 0 {
		return errors.New("amount must be greater than 0")
	}
	if rt.Remarks == "" {
		return errors.New("remarks is required")
	}
	return nil
}
