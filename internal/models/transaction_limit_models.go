package models

import (
	"errors"
	"time"
)

var validLimitServices = map[string]bool{
	"PAYOUT": true,
	"DMT":    true,
	"AEPS":   true,
}

type TransactionLimitModel struct {
	LimitID     int64     `json:"limit_id,omitempty"`
	RetailerID  string    `json:"retailer_id"`
	LimitAmount float64   `json:"limit_amount"`
	Service     string    `json:"service"`
	CreatedAT   time.Time `json:"created_at,omitempty"`
	UpdatedAT   time.Time `json:"updated_at,omitempty"`
}

func (t *TransactionLimitModel) Validate() error {
	if t.RetailerID == "" {
		return errors.New("retailer_id is required")
	}
	if t.LimitAmount < 0 {
		return errors.New("limit_amount must be >= 0")
	}
	if !validLimitServices[t.Service] {
		return errors.New("service must be one of PAYOUT, DMT, AEPS")
	}
	return nil
}
