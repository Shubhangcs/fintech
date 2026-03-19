package models

import (
	"errors"
	"time"

	"github.com/levionstudio/fintech/internal/utils"
)

type RetailerModel struct {
	RetailerID            string    `json:"retailer_id"`
	DistributorID         string    `json:"distributor_id"`
	RetailerName          string    `json:"retailer_name"`
	RetailerPhone         string    `json:"retailer_phone"`
	RetailerEmail         string    `json:"retailer_email"`
	RetailerPassword      string    `json:"retailer_password"`
	RetailerAadharNumber  string    `json:"retailer_aadhar_number"`
	RetailerPanNumber     string    `json:"retailer_pan_number"`
	RetailerDateOfBirth   time.Time `json:"retailer_date_of_birth"`
	RetailerGender        string    `json:"retailer_gender"`
	RetailerCity          string    `json:"retailer_city"`
	RetailerState         string    `json:"retailer_state"`
	RetailerAddress       string    `json:"retailer_address"`
	RetailerPincode       string    `json:"retailer_pincode"`
	RetailerBusinessName  string    `json:"retailer_business_name"`
	RetailerBusinessType  string    `json:"retailer_business_type"`
	RetailerGSTNumber     *string   `json:"retailer_gst_number"`
	RetailerMpin          int       `json:"retailer_mpin"`
	RetailerKYCStatus     bool      `json:"retailer_kyc_status"`
	RetailerWalletBalance float64   `json:"retailer_wallet_balance"`
	IsRetailerBlocked     bool      `json:"is_retailer_blocked"`
	RetailerAadharImage   *string   `json:"retailer_aadhar_image"`
	RetailerPanImage      *string   `json:"retailer_pan_image"`
	RetailerImage         *string   `json:"retailer_image"`
	CreatedAT             time.Time `json:"created_at"`
	UpdatedAT             time.Time `json:"updated_at"`
}

func (re *RetailerModel) ValidateCreateRetailer() error {
	if re.DistributorID == "" {
		return errors.New("distributor_id is required")
	}

	if re.RetailerName == "" {
		return errors.New("retailer name is required")
	}

	if re.RetailerPhone == "" || !utils.IsValid(utils.PhoneRegx, re.RetailerPhone) {
		return errors.New("retailer phone is empty or incorrect")
	}

	if re.RetailerEmail == "" || !utils.IsValid(utils.EmailRegx, re.RetailerEmail) {
		return errors.New("retailer email is empty or incorrect")
	}

	if re.RetailerPassword == "" || !utils.IsValidPassword(re.RetailerPassword) {
		return errors.New("retailer password is empty or does not meet strength requirements")
	}

	if re.RetailerAadharNumber == "" || !utils.IsValid(utils.AadharRegx, re.RetailerAadharNumber) {
		return errors.New("retailer aadhar number must be 12 digits")
	}

	if re.RetailerPanNumber == "" || !utils.IsValid(utils.PanRegx, re.RetailerPanNumber) {
		return errors.New("retailer PAN number is invalid")
	}

	if re.RetailerDateOfBirth.IsZero() {
		return errors.New("retailer date of birth is required")
	}

	if !validGenders[re.RetailerGender] {
		return errors.New("retailer gender must be MALE, FEMALE, or OTHER")
	}

	if re.RetailerCity == "" {
		return errors.New("retailer city is required")
	}

	if re.RetailerState == "" {
		return errors.New("retailer state is required")
	}

	if re.RetailerAddress == "" {
		return errors.New("retailer address is required")
	}

	if re.RetailerPincode == "" || !utils.IsValid(utils.PincodeRegx, re.RetailerPincode) {
		return errors.New("retailer pincode must be 6 digits")
	}

	if re.RetailerBusinessName == "" {
		return errors.New("retailer business name is required")
	}

	if re.RetailerBusinessType == "" {
		return errors.New("retailer business type is required")
	}

	return nil
}
