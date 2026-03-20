package models

import (
	"errors"
	"time"
)

type BeneficiaryModel struct {
	BeneficiaryID       string    `json:"beneficiary_id,omitempty"`
	MobileNumber        string    `json:"mobile_number"`
	BankName            string    `json:"bank_name"`
	IFSCCode            string    `json:"ifsc_code"`
	AccountNumber       string    `json:"account_number"`
	BeneficiaryName     string    `json:"beneficiary_name"`
	BeneficiaryPhone    string    `json:"beneficiary_phone"`
	BeneficiaryVerified bool      `json:"beneficiary_verified"`
	CreatedAT           time.Time `json:"created_at"`
}

// VerifyBeneficiaryRequest is the request body for the verify endpoint.
type VerifyBeneficiaryRequest struct {
	AccountNumber string `json:"account_number"`
	IFSCCode      string `json:"ifsc_code"`
}

func (v *VerifyBeneficiaryRequest) Validate() error {
	if v.AccountNumber == "" {
		return errors.New("account_number is required")
	}
	if v.IFSCCode == "" {
		return errors.New("ifsc_code is required")
	}
	return nil
}

// VerifyBeneficiaryResponse mirrors the Paysprint penny drop response.
type VerifyBeneficiaryResponse struct {
	Status       bool   `json:"status"`
	ResponseCode string `json:"response_code"`
	Message      string `json:"message"`
	Data         any    `json:"data"`
}

func (b *BeneficiaryModel) Validate() error {
	if b.MobileNumber == "" {
		return errors.New("mobile_number is required")
	}
	if b.BankName == "" {
		return errors.New("bank_name is required")
	}
	if b.IFSCCode == "" {
		return errors.New("ifsc_code is required")
	}
	if b.AccountNumber == "" {
		return errors.New("account_number is required")
	}
	if b.BeneficiaryName == "" {
		return errors.New("beneficiary_name is required")
	}
	if b.BeneficiaryPhone == "" {
		return errors.New("beneficiary_phone is required")
	}
	return nil
}
