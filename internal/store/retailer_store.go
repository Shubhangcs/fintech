package store

import (
	"database/sql"
	"errors"

	"github.com/levionstudio/fintech/internal/models"
)

type PostgresRetailerStore struct {
	db *sql.DB
}

func NewPostgresRetailerStore(db *sql.DB) *PostgresRetailerStore {
	return &PostgresRetailerStore{db: db}
}

type RetailerStore interface {
	CreateRetailer(re *models.RetailerModel) error
	UpdateRetailerDetails(re *models.RetailerModel) error
	UpdateRetailerPassword(re *models.RetailerModel) error
	UpdateRetailerMpin(re *models.RetailerModel) error
	UpdateRetailerKYCStatus(re *models.RetailerModel) error
	UpdateRetailerBlockStatus(re *models.RetailerModel) error
	GetRetailerByID(id string) (*models.RetailerModel, error)
	GetRetailersByDistributorID(distributorID string, limit, offset int) ([]models.RetailerModel, error)
	GetRetailersByMasterDistributorID(masterDistributorID string, limit, offset int) ([]models.RetailerModel, error)
	GetRetailersByAdminID(adminID string, limit, offset int) ([]models.RetailerModel, error)
	GetRetailerDetailsForLogin(re *models.RetailerModel) error
	GetRetailersByDistributorIDForDropdown(distributorID string) ([]models.DropdownItem, error)
	GetRetailersByMasterDistributorIDForDropdown(mdID string) ([]models.DropdownItem, error)
	GetRetailersByAdminIDForDropdown(adminID string) ([]models.DropdownItem, error)
	ChangeRetailersDistributor(retailerID, distributorID string) error
	DeleteRetailer(id string) error
	UpdateRetailerAadharImage(path, id string) error
	UpdateRetailerPanImage(path, id string) error
	UpdateRetailerImage(path, id string) error
	GetRetailerWalletBalance(id string) (float64, error)
}

// Create Retailer
func (rs *PostgresRetailerStore) CreateRetailer(re *models.RetailerModel) error {
	query := `
	INSERT INTO retailers (
		distributor_id,
		retailer_name,
		retailer_phone,
		retailer_email,
		retailer_password,
		retailer_aadhar_number,
		retailer_pan_number,
		retailer_date_of_birth,
		retailer_gender,
		retailer_city,
		retailer_state,
		retailer_address,
		retailer_pincode,
		retailer_business_name,
		retailer_business_type,
		retailer_gst_number
	) VALUES (
		$1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14, $15, $16
	)
	RETURNING retailer_id, retailer_mpin, retailer_wallet_balance, created_at, updated_at;
	`

	return rs.db.QueryRow(
		query,
		re.DistributorID,
		re.RetailerName,
		re.RetailerPhone,
		re.RetailerEmail,
		re.RetailerPassword,
		re.RetailerAadharNumber,
		re.RetailerPanNumber,
		re.RetailerDateOfBirth,
		re.RetailerGender,
		re.RetailerCity,
		re.RetailerState,
		re.RetailerAddress,
		re.RetailerPincode,
		re.RetailerBusinessName,
		re.RetailerBusinessType,
		re.RetailerGSTNumber,
	).Scan(
		&re.RetailerID,
		&re.RetailerMpin,
		&re.RetailerWalletBalance,
		&re.CreatedAT,
		&re.UpdatedAT,
	)
}

// Update Retailer Details
func (rs *PostgresRetailerStore) UpdateRetailerDetails(re *models.RetailerModel) error {
	query := `
	UPDATE retailers
	SET
		retailer_name          = COALESCE(NULLIF($1, ''), retailer_name),
		retailer_phone         = COALESCE(NULLIF($2, ''), retailer_phone),
		retailer_email         = COALESCE(NULLIF($3, ''), retailer_email),
		retailer_city          = COALESCE(NULLIF($4, ''), retailer_city),
		retailer_state         = COALESCE(NULLIF($5, ''), retailer_state),
		retailer_address       = COALESCE(NULLIF($6, ''), retailer_address),
		retailer_pincode       = COALESCE(NULLIF($7, ''), retailer_pincode),
		retailer_business_name = COALESCE(NULLIF($8, ''), retailer_business_name),
		retailer_business_type = COALESCE(NULLIF($9, ''), retailer_business_type),
		retailer_gst_number    = COALESCE($10, retailer_gst_number),
		updated_at             = CURRENT_TIMESTAMP
	WHERE retailer_id = $11;
	`

	res, err := rs.db.Exec(
		query,
		re.RetailerName,
		re.RetailerPhone,
		re.RetailerEmail,
		re.RetailerCity,
		re.RetailerState,
		re.RetailerAddress,
		re.RetailerPincode,
		re.RetailerBusinessName,
		re.RetailerBusinessType,
		re.RetailerGSTNumber,
		re.RetailerID,
	)
	if err != nil {
		return err
	}

	return checkRowsAffected(res)
}

// Update Retailer Password
func (rs *PostgresRetailerStore) UpdateRetailerPassword(re *models.RetailerModel) error {
	query := `
	UPDATE retailers
	SET retailer_password = $1,
		updated_at        = CURRENT_TIMESTAMP
	WHERE retailer_id     = $2;
	`

	res, err := rs.db.Exec(query, re.RetailerPassword, re.RetailerID)
	if err != nil {
		return err
	}

	return checkRowsAffected(res)
}

// Update Retailer MPIN
func (rs *PostgresRetailerStore) UpdateRetailerMpin(re *models.RetailerModel) error {
	query := `
	UPDATE retailers
	SET retailer_mpin = $1,
		updated_at    = CURRENT_TIMESTAMP
	WHERE retailer_id = $2;
	`

	res, err := rs.db.Exec(query, re.RetailerMpin, re.RetailerID)
	if err != nil {
		return err
	}

	return checkRowsAffected(res)
}

// Update Retailer KYC Status
func (rs *PostgresRetailerStore) UpdateRetailerKYCStatus(re *models.RetailerModel) error {
	query := `
	UPDATE retailers
	SET retailer_kyc_status = $1,
		updated_at          = CURRENT_TIMESTAMP
	WHERE retailer_id       = $2;
	`

	res, err := rs.db.Exec(query, re.RetailerKYCStatus, re.RetailerID)
	if err != nil {
		return err
	}

	return checkRowsAffected(res)
}

// Update Retailer Block Status
func (rs *PostgresRetailerStore) UpdateRetailerBlockStatus(re *models.RetailerModel) error {
	query := `
	UPDATE retailers
	SET is_retailer_blocked = $1,
		updated_at          = CURRENT_TIMESTAMP
	WHERE retailer_id       = $2;
	`

	res, err := rs.db.Exec(query, re.IsRetailerBlocked, re.RetailerID)
	if err != nil {
		return err
	}

	return checkRowsAffected(res)
}

// Get Retailer By ID
func (rs *PostgresRetailerStore) GetRetailerByID(id string) (*models.RetailerModel, error) {
	query := `
	SELECT
		retailer_id,
		distributor_id,
		retailer_name,
		retailer_phone,
		retailer_email,
		retailer_password,
		retailer_mpin,
		retailer_aadhar_number,
		retailer_pan_number,
		retailer_date_of_birth,
		retailer_gender,
		retailer_city,
		retailer_state,
		retailer_address,
		retailer_pincode,
		retailer_business_name,
		retailer_business_type,
		retailer_gst_number,
		retailer_kyc_status,
		retailer_wallet_balance,
		is_retailer_blocked,
		retailer_aadhar_image,
		retailer_pan_image,
		retailer_image,
		created_at,
		updated_at
	FROM retailers
	WHERE retailer_id = $1;
	`

	var re models.RetailerModel
	err := rs.db.QueryRow(query, id).Scan(
		&re.RetailerID,
		&re.DistributorID,
		&re.RetailerName,
		&re.RetailerPhone,
		&re.RetailerEmail,
		&re.RetailerPassword,
		&re.RetailerMpin,
		&re.RetailerAadharNumber,
		&re.RetailerPanNumber,
		&re.RetailerDateOfBirth,
		&re.RetailerGender,
		&re.RetailerCity,
		&re.RetailerState,
		&re.RetailerAddress,
		&re.RetailerPincode,
		&re.RetailerBusinessName,
		&re.RetailerBusinessType,
		&re.RetailerGSTNumber,
		&re.RetailerKYCStatus,
		&re.RetailerWalletBalance,
		&re.IsRetailerBlocked,
		&re.RetailerAadharImage,
		&re.RetailerPanImage,
		&re.RetailerImage,
		&re.CreatedAT,
		&re.UpdatedAT,
	)

	return &re, err
}

// Get Retailers By Distributor ID
func (rs *PostgresRetailerStore) GetRetailersByDistributorID(distributorID string, limit, offset int) ([]models.RetailerModel, error) {
	query := `
	SELECT
		retailer_id, distributor_id, retailer_name, retailer_phone, retailer_email,
		retailer_password, retailer_mpin,
		retailer_aadhar_number, retailer_pan_number, retailer_date_of_birth, retailer_gender,
		retailer_city, retailer_state, retailer_address, retailer_pincode,
		retailer_business_name, retailer_business_type, retailer_gst_number,
		retailer_kyc_status, retailer_wallet_balance,
		is_retailer_blocked, retailer_aadhar_image, retailer_pan_image,
		retailer_image, created_at, updated_at
	FROM retailers
	WHERE distributor_id = $1
	ORDER BY created_at DESC
	LIMIT $2 OFFSET $3;
	`

	return scanRetailers(rs.db, query, distributorID, limit, offset)
}

// Get Retailers By Master Distributor ID
func (rs *PostgresRetailerStore) GetRetailersByMasterDistributorID(masterDistributorID string, limit, offset int) ([]models.RetailerModel, error) {
	query := `
	SELECT
		re.retailer_id, re.distributor_id, re.retailer_name, re.retailer_phone, re.retailer_email,
		re.retailer_password, re.retailer_mpin,
		re.retailer_aadhar_number, re.retailer_pan_number, re.retailer_date_of_birth, re.retailer_gender,
		re.retailer_city, re.retailer_state, re.retailer_address, re.retailer_pincode,
		re.retailer_business_name, re.retailer_business_type, re.retailer_gst_number,
		re.retailer_kyc_status, re.retailer_wallet_balance,
		re.is_retailer_blocked, re.retailer_aadhar_image, re.retailer_pan_image,
		re.retailer_image, re.created_at, re.updated_at
	FROM retailers re
	JOIN distributors d ON re.distributor_id = d.distributor_id
	WHERE d.master_distributor_id = $1
	ORDER BY re.created_at DESC
	LIMIT $2 OFFSET $3;
	`

	return scanRetailers(rs.db, query, masterDistributorID, limit, offset)
}

// Get Retailers By Admin ID
func (rs *PostgresRetailerStore) GetRetailersByAdminID(adminID string, limit, offset int) ([]models.RetailerModel, error) {
	query := `
	SELECT
		re.retailer_id, re.distributor_id, re.retailer_name, re.retailer_phone, re.retailer_email,
		re.retailer_password, re.retailer_mpin,
		re.retailer_aadhar_number, re.retailer_pan_number, re.retailer_date_of_birth, re.retailer_gender,
		re.retailer_city, re.retailer_state, re.retailer_address, re.retailer_pincode,
		re.retailer_business_name, re.retailer_business_type, re.retailer_gst_number,
		re.retailer_kyc_status, re.retailer_wallet_balance,
		re.is_retailer_blocked, re.retailer_aadhar_image, re.retailer_pan_image,
		re.retailer_image, re.created_at, re.updated_at
	FROM retailers re
	JOIN distributors d ON re.distributor_id = d.distributor_id
	JOIN master_distributors md ON d.master_distributor_id = md.master_distributor_id
	WHERE md.admin_id = $1
	ORDER BY re.created_at DESC
	LIMIT $2 OFFSET $3;
	`

	return scanRetailers(rs.db, query, adminID, limit, offset)
}

// Get Retailer Details For Login
func (rs *PostgresRetailerStore) GetRetailerDetailsForLogin(re *models.RetailerModel) error {
	query := `
	SELECT
		retailer_id,
		retailer_name
	FROM retailers
	WHERE retailer_id = $1
	AND retailer_password = $2
	AND is_retailer_blocked = FALSE;
	`

	err := rs.db.QueryRow(query, re.RetailerID, re.RetailerPassword).Scan(
		&re.RetailerID,
		&re.RetailerName,
	)
	if errors.Is(err, sql.ErrNoRows) {
		return errors.New("invalid credentials")
	}

	return err
}

// Get Retailers By Distributor ID For Dropdown
func (rs *PostgresRetailerStore) GetRetailersByDistributorIDForDropdown(distributorID string) ([]models.DropdownItem, error) {
	query := `
	SELECT retailer_id, retailer_name
	FROM retailers
	WHERE distributor_id = $1
	ORDER BY retailer_name;
	`

	return scanDropdown(rs.db, query, distributorID)
}

// Get Retailers By Master Distributor ID For Dropdown
func (rs *PostgresRetailerStore) GetRetailersByMasterDistributorIDForDropdown(mdID string) ([]models.DropdownItem, error) {
	query := `
	SELECT re.retailer_id, re.retailer_name
	FROM retailers re
	JOIN distributors d ON re.distributor_id = d.distributor_id
	WHERE d.master_distributor_id = $1
	ORDER BY re.retailer_name;
	`

	return scanDropdown(rs.db, query, mdID)
}

// Get Retailers By Admin ID For Dropdown
func (rs *PostgresRetailerStore) GetRetailersByAdminIDForDropdown(adminID string) ([]models.DropdownItem, error) {
	query := `
	SELECT re.retailer_id, re.retailer_name
	FROM retailers re
	JOIN distributors d ON re.distributor_id = d.distributor_id
	JOIN master_distributors md ON d.master_distributor_id = md.master_distributor_id
	WHERE md.admin_id = $1
	ORDER BY re.retailer_name;
	`

	return scanDropdown(rs.db, query, adminID)
}

// Delete Retailer
func (rs *PostgresRetailerStore) DeleteRetailer(id string) error {
	query := `DELETE FROM retailers WHERE retailer_id = $1;`

	res, err := rs.db.Exec(query, id)
	if err != nil {
		return err
	}

	return checkRowsAffected(res)
}

func scanRetailers(db *sql.DB, query string, args ...any) ([]models.RetailerModel, error) {
	rows, err := db.Query(query, args...)
	if err != nil && errors.Is(err, sql.ErrNoRows) {
		return []models.RetailerModel{}, nil
	}
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var retailers []models.RetailerModel
	for rows.Next() {
		var re models.RetailerModel
		err = rows.Scan(
			&re.RetailerID,
			&re.DistributorID,
			&re.RetailerName,
			&re.RetailerPhone,
			&re.RetailerEmail,
			&re.RetailerPassword,
			&re.RetailerMpin,
			&re.RetailerAadharNumber,
			&re.RetailerPanNumber,
			&re.RetailerDateOfBirth,
			&re.RetailerGender,
			&re.RetailerCity,
			&re.RetailerState,
			&re.RetailerAddress,
			&re.RetailerPincode,
			&re.RetailerBusinessName,
			&re.RetailerBusinessType,
			&re.RetailerGSTNumber,
			&re.RetailerKYCStatus,
			&re.RetailerWalletBalance,
			&re.IsRetailerBlocked,
			&re.RetailerAadharImage,
			&re.RetailerPanImage,
			&re.RetailerImage,
			&re.CreatedAT,
			&re.UpdatedAT,
		)
		if err != nil {
			return nil, err
		}
		retailers = append(retailers, re)
	}

	return retailers, rows.Err()
}

// Change Retailers Distributor
func (rs *PostgresRetailerStore) ChangeRetailersDistributor(retailerID, distributorID string) error {
	res, err := rs.db.Exec(`
		UPDATE retailers
		SET distributor_id = $1, 
			updated_at = CURRENT_TIMESTAMP
		WHERE retailer_id = $2
	`, distributorID, retailerID)
	if err != nil {
		return err
	}
	return checkRowsAffected(res)
}

// Update Retailer Aadhar Image
func (rs *PostgresRetailerStore) UpdateRetailerAadharImage(path, id string) error {
	query := `
		UPDATE retailers
		SET retailer_aadhar_image = $1,
			updated_at = CURRENT_TIMESTAMP
		WHERE retailer_id = $2;
	`
	res, err := rs.db.Exec(query, path, id)
	if err != nil {
		return err
	}

	return checkRowsAffected(res)
}

// Update Retailer Pan Image
func (rs *PostgresRetailerStore) UpdateRetailerPanImage(path, id string) error {
	query := `
		UPDATE retailers
		SET retailer_pan_image = $1,
			updated_at = CURRENT_TIMESTAMP
		WHERE retailer_id = $2;
	`
	res, err := rs.db.Exec(query, path, id)
	if err != nil {
		return err
	}

	return checkRowsAffected(res)
}

// Update Retailer Image
func (rs *PostgresRetailerStore) UpdateRetailerImage(path, id string) error {
	query := `
		UPDATE retailers
		SET retailer_image = $1,
			updated_at = CURRENT_TIMESTAMP
		WHERE retailer_id = $2;
	`
	res, err := rs.db.Exec(query, path, id)
	if err != nil {
		return err
	}

	return checkRowsAffected(res)
}

// Get Retailer Wallet Balance
func (rs *PostgresRetailerStore) GetRetailerWalletBalance(id string) (float64, error) {
	query := `
		SELECT 
			retailer_wallet_balance
		FROM retailers 
		WHERE retailer_id = $1;
	`
	var balance float64
	err := rs.db.QueryRow(
		query,
		id,
	).Scan(
		&balance,
	)
	return balance, err
}
