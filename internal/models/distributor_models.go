package models

import (
	"errors"
	"time"

	"github.com/levionstudio/fintech/internal/utils"
)

type DistributorModel struct {
	DistributorID            string    `json:"distributor_id"`
	MasterDistributorID      string    `json:"master_distributor_id"`
	DistributorName          string    `json:"distributor_name"`
	DistributorPhone         string    `json:"distributor_phone"`
	DistributorEmail         string    `json:"distributor_email"`
	DistributorPassword      string    `json:"distributor_password"`
	DistributorAadharNumber  string    `json:"distributor_aadhar_number"`
	DistributorPanNumber     string    `json:"distributor_pan_number"`
	DistributorDateOfBirth   time.Time `json:"distributor_date_of_birth"`
	DistributorGender        string    `json:"distributor_gender"`
	DistributorCity          string    `json:"distributor_city"`
	DistributorState         string    `json:"distributor_state"`
	DistributorAddress       string    `json:"distributor_address"`
	DistributorPincode       string    `json:"distributor_pincode"`
	DistributorBusinessName  string    `json:"distributor_business_name"`
	DistributorBusinessType  string    `json:"distributor_business_type"`
	DistributorGSTNumber     *string   `json:"distributor_gst_number"`
	DistributorMpin          int       `json:"distributor_mpin"`
	DistributorKYCStatus     bool      `json:"distributor_kyc_status"`
	DistributorWalletBalance float64   `json:"distributor_wallet_balance"`
	DistributorAadharImage   *string   `json:"distributor_aadhar_image"`
	DistributorPanImage      *string   `json:"distributor_pan_image"`
	DistributorImage         *string   `json:"distributor_image"`
	IsDistributorBlocked     bool      `json:"is_distributor_blocked"`
	CreatedAT                time.Time `json:"created_at"`
	UpdatedAT                time.Time `json:"updated_at"`
}

func (d *DistributorModel) ValidateCreateDistributor() error {
	if d.MasterDistributorID == "" {
		return errors.New("master_distributor_id is required")
	}

	if d.DistributorName == "" {
		return errors.New("distributor name is required")
	}

	if d.DistributorPhone == "" || !utils.IsValid(utils.PhoneRegx, d.DistributorPhone) {
		return errors.New("distributor phone is empty or incorrect")
	}

	if d.DistributorEmail == "" || !utils.IsValid(utils.EmailRegx, d.DistributorEmail) {
		return errors.New("distributor email is empty or incorrect")
	}

	if d.DistributorPassword == "" || !utils.IsValidPassword(d.DistributorPassword) {
		return errors.New("distributor password is empty or does not meet strength requirements")
	}

	if d.DistributorAadharNumber == "" || !utils.IsValid(utils.AadharRegx, d.DistributorAadharNumber) {
		return errors.New("distributor aadhar number must be 12 digits")
	}

	if d.DistributorPanNumber == "" || !utils.IsValid(utils.PanRegx, d.DistributorPanNumber) {
		return errors.New("distributor PAN number is invalid")
	}

	if d.DistributorDateOfBirth.IsZero() {
		return errors.New("distributor date of birth is required")
	}

	if !validGenders[d.DistributorGender] {
		return errors.New("distributor gender must be MALE, FEMALE, or OTHER")
	}

	if d.DistributorCity == "" {
		return errors.New("distributor city is required")
	}

	if d.DistributorState == "" {
		return errors.New("distributor state is required")
	}

	if d.DistributorAddress == "" {
		return errors.New("distributor address is required")
	}

	if d.DistributorPincode == "" || !utils.IsValid(utils.PincodeRegx, d.DistributorPincode) {
		return errors.New("distributor pincode must be 6 digits")
	}

	if d.DistributorBusinessName == "" {
		return errors.New("distributor business name is required")
	}

	if d.DistributorBusinessType == "" {
		return errors.New("distributor business type is required")
	}

	return nil
}
