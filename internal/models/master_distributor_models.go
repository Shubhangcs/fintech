package models

import (
	"errors"
	"time"

	"github.com/levionstudio/fintech/internal/utils"
)

type MasterDistributorModel struct {
	MasterDistributorID            string    `json:"master_distributor_id"`
	AdminID                        string    `json:"admin_id"`
	MasterDistributorName          string    `json:"master_distributor_name"`
	MasterDistributorPhone         string    `json:"master_distributor_phone"`
	MasterDistributorEmail         string    `json:"master_distributor_email"`
	MasterDistributorPassword      string    `json:"master_distributor_password"`
	MasterDistributorAadharNumber  string    `json:"master_distributor_aadhar_number"`
	MasterDistributorPanNumber     string    `json:"master_distributor_pan_number"`
	MasterDistributorDateOfBirth   time.Time `json:"master_distributor_date_of_birth"`
	MasterDistributorGender        string    `json:"master_distributor_gender"`
	MasterDistributorCity          string    `json:"master_distributor_city"`
	MasterDistributorState         string    `json:"master_distributor_state"`
	MasterDistributorAddress       string    `json:"master_distributor_address"`
	MasterDistributorPincode       string    `json:"master_distributor_pincode"`
	MasterDistributorBusinessName  string    `json:"master_distributor_business_name"`
	MasterDistributorBusinessType  string    `json:"master_distributor_business_type"`
	MasterDistributorMpin          int       `json:"master_distributor_mpin"`
	MasterDistributorKYCStatus     bool      `json:"master_distributor_kyc_status"`
	MasterDistributorGSTNumber     *string   `json:"master_distributor_gst_number"`
	MasterDistributorWalletBalance float64   `json:"master_distributor_wallet_balance"`
	HoldAmount                     float64   `json:"hold_amount"`
	IsMasterDistributorBlocked     bool      `json:"is_master_distributor_blocked"`
	MasterDistributorAadharImage   *string   `json:"master_distributor_aadhar_image"`
	MasterDistributorPanImage      *string   `json:"master_distributor_pan_image"`
	MasterDistributorImage         *string   `json:"master_distributor_image"`
	CreatedAT                      time.Time `json:"created_at"`
	UpdatedAT                      time.Time `json:"updated_at"`
}

var validGenders = map[string]bool{"MALE": true, "FEMALE": true, "OTHER": true}

func (md *MasterDistributorModel) ValidateCreateMasterDistributor() error {
	if md.AdminID == "" {
		return errors.New("admin_id is required")
	}

	if md.MasterDistributorName == "" {
		return errors.New("master distributor name is required")
	}

	if md.MasterDistributorPhone == "" || !utils.IsValid(utils.PhoneRegx, md.MasterDistributorPhone) {
		return errors.New("master distributor phone is empty or incorrect")
	}

	if md.MasterDistributorEmail == "" || !utils.IsValid(utils.EmailRegx, md.MasterDistributorEmail) {
		return errors.New("master distributor email is empty or incorrect")
	}

	if md.MasterDistributorPassword == "" || !utils.IsValidPassword(md.MasterDistributorPassword) {
		return errors.New("master distributor password is empty or does not meet strength requirements")
	}

	if md.MasterDistributorAadharNumber == "" || !utils.IsValid(utils.AadharRegx, md.MasterDistributorAadharNumber) {
		return errors.New("master distributor aadhar number must be 12 digits")
	}

	if md.MasterDistributorPanNumber == "" || !utils.IsValid(utils.PanRegx, md.MasterDistributorPanNumber) {
		return errors.New("master distributor PAN number is invalid")
	}

	if md.MasterDistributorDateOfBirth.IsZero() {
		return errors.New("master distributor date of birth is required")
	}

	if !validGenders[md.MasterDistributorGender] {
		return errors.New("master distributor gender must be MALE, FEMALE, or OTHER")
	}

	if md.MasterDistributorCity == "" {
		return errors.New("master distributor city is required")
	}

	if md.MasterDistributorState == "" {
		return errors.New("master distributor state is required")
	}

	if md.MasterDistributorAddress == "" {
		return errors.New("master distributor address is required")
	}

	if md.MasterDistributorPincode == "" || !utils.IsValid(utils.PincodeRegx, md.MasterDistributorPincode) {
		return errors.New("master distributor pincode must be 6 digits")
	}

	if md.MasterDistributorBusinessName == "" {
		return errors.New("master distributor business name is required")
	}

	if md.MasterDistributorBusinessType == "" {
		return errors.New("master distributor business type is required")
	}

	return nil
}
