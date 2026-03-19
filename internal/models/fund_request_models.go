package models

import (
	"errors"
	"time"
)

type FundRequestModel struct {
	FundRequestID          int64     `json:"fund_request_id"`
	RequesterID            string    `json:"requester_id"`
	RequestToID            string    `json:"request_to_id"`
	Amount                 float64   `json:"amount"`
	BankName               *string   `json:"bank_name"`
	RequestDate            time.Time `json:"request_date"`
	UTRNumber              *string   `json:"utr_number"`
	RequestType            string    `json:"request_type"`
	RequestStatus          string    `json:"request_status"`
	Remarks                string    `json:"remarks"`
	RejectRemarks          *string   `json:"reject_remarks"`
	RequesterName          string    `json:"requester_name,omitempty"`
	RequesterBusinessName  *string   `json:"requester_business_name,omitempty"`
	RequestToName          string    `json:"request_to_name,omitempty"`
	RequestToBusinessName  *string   `json:"request_to_business_name,omitempty"`
	CreatedAT              time.Time `json:"created_at"`
	UpdatedAT              time.Time `json:"updated_at"`
}

func (fr *FundRequestModel) Validate() error {
	if fr.RequesterID == "" {
		return errors.New("requester_id is required")
	}

	if fr.RequestToID == "" {
		return errors.New("request_to_id is required")
	}

	if fr.RequesterID == fr.RequestToID {
		return errors.New("requester and request_to cannot be the same")
	}

	if fr.Amount <= 0 {
		return errors.New("amount must be greater than 0")
	}

	if fr.RequestDate.IsZero() {
		return errors.New("request_date is required")
	}

	if fr.RequestType != "NORMAL" && fr.RequestType != "ADVANCE" {
		return errors.New("request_type must be NORMAL or ADVANCE")
	}

	return nil
}
