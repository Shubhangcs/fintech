package models

import (
	"errors"
	"time"

	"github.com/levionstudio/fintech/internal/utils"
)

type AdminModel struct {
	AdminID            string    `json:"admin_id"`
	AdminName          string    `json:"admin_name"`
	AdminEmail         string    `json:"admin_email"`
	AdminPhone         string    `json:"admin_phone"`
	AdminPassword      string    `json:"admin_password"`
	AdminWalletBalance float64   `json:"admin_wallet_balance"`
	CreatedAT          time.Time `json:"created_at"`
	UpdatedAT          time.Time `json:"updated_at"`
}

func (am *AdminModel) ValidateCreateAdmin() error {
	if am.AdminName == "" {
		return errors.New("invalid request format, admin name is required")
	}

	if am.AdminEmail == "" || !utils.IsValid(utils.EmailRegx, am.AdminEmail) {
		return errors.New("invalid request format, admin email is empty or incorrect")
	}

	if am.AdminPhone == "" || !utils.IsValid(utils.PhoneRegx, am.AdminPhone) {
		return errors.New("invalid request format, admin phone is empty or incorrect")
	}

	if am.AdminPassword == "" || !utils.IsValidPassword(am.AdminPassword) {
		return errors.New("invalid request format, admin password is empty or incorrect")
	}

	return nil
}
