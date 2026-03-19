package models

import "errors"

type BankModel struct {
	BankID   int64  `json:"bank_id,omitempty"`
	BankName string `json:"bank_name"`
	IFSCCode string `json:"ifsc_code"`
}

func (b *BankModel) Validate() error {
	if b.BankName == "" {
		return errors.New("bank_name is required")
	}
	if b.IFSCCode == "" {
		return errors.New("ifsc_code is required")
	}
	return nil
}

type AdminBankModel struct {
	AdminBankID            int64  `json:"admin_bank_id,omitempty"`
	AdminID                string `json:"admin_id"`
	AdminBankName          string `json:"admin_bank_name"`
	AdminBankAccountNumber string `json:"admin_bank_account_number"`
	AdminBankIFSCCode      string `json:"admin_bank_ifsc_code"`
}

func (ab *AdminBankModel) Validate() error {
	if ab.AdminID == "" {
		return errors.New("admin_id is required")
	}
	if ab.AdminBankName == "" {
		return errors.New("admin_bank_name is required")
	}
	if ab.AdminBankAccountNumber == "" {
		return errors.New("admin_bank_account_number is required")
	}
	if ab.AdminBankIFSCCode == "" {
		return errors.New("admin_bank_ifsc_code is required")
	}
	return nil
}
