package store

import (
	"database/sql"
	"errors"

	"github.com/levionstudio/fintech/internal/models"
)

type PostgresMasterDistributorStore struct {
	db *sql.DB
}

func NewPostgresMasterDistributorStore(db *sql.DB) *PostgresMasterDistributorStore {
	return &PostgresMasterDistributorStore{db: db}
}

type MasterDistributorStore interface {
	CreateMasterDistributor(md *models.MasterDistributorModel) error
	UpdateMasterDistributorDetails(md *models.MasterDistributorModel) error
	UpdateMasterDistributorPassword(md *models.MasterDistributorModel) error
	UpdateMasterDistributorMpin(md *models.MasterDistributorModel) error
	UpdateMasterDistributorKYCStatus(md *models.MasterDistributorModel) error
	UpdateMasterDistributorBlockStatus(md *models.MasterDistributorModel) error
	GetMasterDistributorByID(id string) (*models.MasterDistributorModel, error)
	GetMasterDistributorsByAdminID(adminID string, limit, offset int) ([]models.MasterDistributorModel, error)
	GetMasterDistributorDetailsForLogin(md *models.MasterDistributorModel) error
	GetMasterDistributorsByAdminIDForDropdown(adminID string) ([]models.DropdownItem, error)
	DeleteMasterDistributor(id string) error
	GetMasterDistributorWalletBalance(id string) (float64, error)
	UpdateMasterDistributorAadharImage(path, id string) error
	UpdateMasterDistributorPanImage(path, id string) error
	UpdateMasterDistributorImage(path, id string) error
}

// Create Master Distributor
func (ms *PostgresMasterDistributorStore) CreateMasterDistributor(md *models.MasterDistributorModel) error {
	query := `
	INSERT INTO master_distributors (
		admin_id,
		master_distributor_name,
		master_distributor_phone,
		master_distributor_email,
		master_distributor_password,
		master_distributor_aadhar_number,
		master_distributor_pan_number,
		master_distributor_date_of_birth,
		master_distributor_gender,
		master_distributor_city,
		master_distributor_state,
		master_distributor_address,
		master_distributor_pincode,
		master_distributor_business_name,
		master_distributor_business_type,
		master_distributor_gst_number
	) VALUES (
		$1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14, $15, $16
	)
	RETURNING master_distributor_id, master_distributor_mpin, master_distributor_wallet_balance, created_at, updated_at;
	`

	return ms.db.QueryRow(
		query,
		md.AdminID,
		md.MasterDistributorName,
		md.MasterDistributorPhone,
		md.MasterDistributorEmail,
		md.MasterDistributorPassword,
		md.MasterDistributorAadharNumber,
		md.MasterDistributorPanNumber,
		md.MasterDistributorDateOfBirth,
		md.MasterDistributorGender,
		md.MasterDistributorCity,
		md.MasterDistributorState,
		md.MasterDistributorAddress,
		md.MasterDistributorPincode,
		md.MasterDistributorBusinessName,
		md.MasterDistributorBusinessType,
		md.MasterDistributorGSTNumber,
	).Scan(
		&md.MasterDistributorID,
		&md.MasterDistributorMpin,
		&md.MasterDistributorWalletBalance,
		&md.CreatedAT,
		&md.UpdatedAT,
	)
}

// Update Master Distributor Details
func (ms *PostgresMasterDistributorStore) UpdateMasterDistributorDetails(md *models.MasterDistributorModel) error {
	query := `
	UPDATE master_distributors
	SET
		master_distributor_name          = COALESCE(NULLIF($1, ''), master_distributor_name),
		master_distributor_phone         = COALESCE(NULLIF($2, ''), master_distributor_phone),
		master_distributor_email         = COALESCE(NULLIF($3, ''), master_distributor_email),
		master_distributor_city          = COALESCE(NULLIF($4, ''), master_distributor_city),
		master_distributor_state         = COALESCE(NULLIF($5, ''), master_distributor_state),
		master_distributor_address       = COALESCE(NULLIF($6, ''), master_distributor_address),
		master_distributor_pincode       = COALESCE(NULLIF($7, ''), master_distributor_pincode),
		master_distributor_business_name = COALESCE(NULLIF($8, ''), master_distributor_business_name),
		master_distributor_business_type = COALESCE(NULLIF($9, ''), master_distributor_business_type),
		master_distributor_gst_number    = COALESCE($10, master_distributor_gst_number),
		updated_at                       = CURRENT_TIMESTAMP
	WHERE master_distributor_id = $11;
	`

	res, err := ms.db.Exec(
		query,
		md.MasterDistributorName,
		md.MasterDistributorPhone,
		md.MasterDistributorEmail,
		md.MasterDistributorCity,
		md.MasterDistributorState,
		md.MasterDistributorAddress,
		md.MasterDistributorPincode,
		md.MasterDistributorBusinessName,
		md.MasterDistributorBusinessType,
		md.MasterDistributorGSTNumber,
		md.MasterDistributorID,
	)
	if err != nil {
		return err
	}

	return checkRowsAffected(res)
}

// Update Master Distributor Password
func (ms *PostgresMasterDistributorStore) UpdateMasterDistributorPassword(md *models.MasterDistributorModel) error {
	query := `
	UPDATE master_distributors
	SET master_distributor_password = $1,
		updated_at = CURRENT_TIMESTAMP
	WHERE master_distributor_id = $2;
	`

	res, err := ms.db.Exec(query, md.MasterDistributorPassword, md.MasterDistributorID)
	if err != nil {
		return err
	}

	return checkRowsAffected(res)
}

// Update Master Distributor MPIN
func (ms *PostgresMasterDistributorStore) UpdateMasterDistributorMpin(md *models.MasterDistributorModel) error {
	query := `
	UPDATE master_distributors
	SET master_distributor_mpin = $1,
		updated_at = CURRENT_TIMESTAMP
	WHERE master_distributor_id = $2;
	`

	res, err := ms.db.Exec(query, md.MasterDistributorMpin, md.MasterDistributorID)
	if err != nil {
		return err
	}

	return checkRowsAffected(res)
}

// Update Master Distributor KYC Status
func (ms *PostgresMasterDistributorStore) UpdateMasterDistributorKYCStatus(md *models.MasterDistributorModel) error {
	query := `
	UPDATE master_distributors
	SET master_distributor_kyc_status = $1,
		updated_at = CURRENT_TIMESTAMP
	WHERE master_distributor_id = $2;
	`

	res, err := ms.db.Exec(query, md.MasterDistributorKYCStatus, md.MasterDistributorID)
	if err != nil {
		return err
	}

	return checkRowsAffected(res)
}

// Update Master Distributor Block Status
func (ms *PostgresMasterDistributorStore) UpdateMasterDistributorBlockStatus(md *models.MasterDistributorModel) error {
	query := `
	UPDATE master_distributors
	SET is_master_distributor_blocked = $1,
		updated_at = CURRENT_TIMESTAMP
	WHERE master_distributor_id = $2;
	`

	res, err := ms.db.Exec(query, md.IsMasterDistributorBlocked, md.MasterDistributorID)
	if err != nil {
		return err
	}

	return checkRowsAffected(res)
}

// Get Master Distributor By ID
func (ms *PostgresMasterDistributorStore) GetMasterDistributorByID(id string) (*models.MasterDistributorModel, error) {
	query := `
	SELECT
		master_distributor_id,
		admin_id,
		master_distributor_name,
		master_distributor_phone,
		master_distributor_email,
		master_distributor_aadhar_number,
		master_distributor_pan_number,
		master_distributor_date_of_birth,
		master_distributor_gender,
		master_distributor_city,
		master_distributor_state,
		master_distributor_address,
		master_distributor_pincode,
		master_distributor_business_name,
		master_distributor_business_type,
		master_distributor_kyc_status,
		master_distributor_gst_number,
		master_distributor_wallet_balance,
		is_master_distributor_blocked,
		master_distributor_aadhar_image,
		master_distributor_pan_image,
		master_distributor_image,
		created_at,
		updated_at
	FROM master_distributors
	WHERE master_distributor_id = $1;
	`

	var md models.MasterDistributorModel
	err := ms.db.QueryRow(query, id).Scan(
		&md.MasterDistributorID,
		&md.AdminID,
		&md.MasterDistributorName,
		&md.MasterDistributorPhone,
		&md.MasterDistributorEmail,
		&md.MasterDistributorAadharNumber,
		&md.MasterDistributorPanNumber,
		&md.MasterDistributorDateOfBirth,
		&md.MasterDistributorGender,
		&md.MasterDistributorCity,
		&md.MasterDistributorState,
		&md.MasterDistributorAddress,
		&md.MasterDistributorPincode,
		&md.MasterDistributorBusinessName,
		&md.MasterDistributorBusinessType,
		&md.MasterDistributorKYCStatus,
		&md.MasterDistributorGSTNumber,
		&md.MasterDistributorWalletBalance,
		&md.IsMasterDistributorBlocked,
		&md.MasterDistributorAadharImage,
		&md.MasterDistributorPanImage,
		&md.MasterDistributorImage,
		&md.CreatedAT,
		&md.UpdatedAT,
	)

	return &md, err
}

// Get Master Distributor By Admin ID
func (ms *PostgresMasterDistributorStore) GetMasterDistributorsByAdminID(adminID string, limit, offset int) ([]models.MasterDistributorModel, error) {
	query := `
	SELECT
		master_distributor_id,
		admin_id,
		master_distributor_name,
		master_distributor_phone,
		master_distributor_email,
		master_distributor_aadhar_number,
		master_distributor_pan_number,
		master_distributor_date_of_birth,
		master_distributor_gender,
		master_distributor_city,
		master_distributor_state,
		master_distributor_address,
		master_distributor_pincode,
		master_distributor_business_name,
		master_distributor_business_type,
		master_distributor_kyc_status,
		master_distributor_gst_number,
		master_distributor_wallet_balance,
		is_master_distributor_blocked,
		master_distributor_aadhar_image,
		master_distributor_pan_image,
		master_distributor_image,
		created_at,
		updated_at
	FROM master_distributors
	WHERE admin_id = $1
	ORDER BY created_at DESC
	LIMIT $2 OFFSET $3;
	`

	rows, err := ms.db.Query(query, adminID, limit, offset)
	if err != nil && errors.Is(err, sql.ErrNoRows) {
		return []models.MasterDistributorModel{}, nil
	}
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var mds []models.MasterDistributorModel
	for rows.Next() {
		var md models.MasterDistributorModel
		err = rows.Scan(
			&md.MasterDistributorID,
			&md.AdminID,
			&md.MasterDistributorName,
			&md.MasterDistributorPhone,
			&md.MasterDistributorEmail,
			&md.MasterDistributorAadharNumber,
			&md.MasterDistributorPanNumber,
			&md.MasterDistributorDateOfBirth,
			&md.MasterDistributorGender,
			&md.MasterDistributorCity,
			&md.MasterDistributorState,
			&md.MasterDistributorAddress,
			&md.MasterDistributorPincode,
			&md.MasterDistributorBusinessName,
			&md.MasterDistributorBusinessType,
			&md.MasterDistributorKYCStatus,
			&md.MasterDistributorGSTNumber,
			&md.MasterDistributorWalletBalance,
			&md.IsMasterDistributorBlocked,
			&md.MasterDistributorAadharImage,
			&md.MasterDistributorPanImage,
			&md.MasterDistributorImage,
			&md.CreatedAT,
			&md.UpdatedAT,
		)
		if err != nil {
			return nil, err
		}
		mds = append(mds, md)
	}

	return mds, rows.Err()
}

// Get Master Distributors By Admin ID For Dropdown
func (ms *PostgresMasterDistributorStore) GetMasterDistributorsByAdminIDForDropdown(adminID string) ([]models.DropdownItem, error) {
	query := `
	SELECT master_distributor_id, master_distributor_name
	FROM master_distributors
	WHERE admin_id = $1
	ORDER BY master_distributor_name;
	`
	return scanDropdown(ms.db, query, adminID)
}

// Get Master Distributor Details For Login
func (ms *PostgresMasterDistributorStore) GetMasterDistributorDetailsForLogin(md *models.MasterDistributorModel) error {
	query := `
	SELECT
		master_distributor_id,
		master_distributor_name
	FROM master_distributors
	WHERE master_distributor_id = $1
	AND master_distributor_password = $2
	AND is_master_distributor_blocked = FALSE;
	`

	err := ms.db.QueryRow(query, md.MasterDistributorID, md.MasterDistributorPassword).Scan(
		&md.MasterDistributorID,
		&md.MasterDistributorName,
	)
	if errors.Is(err, sql.ErrNoRows) {
		return errors.New("invalid credentials")
	}

	return err
}

// Delete Master Distributor
func (ms *PostgresMasterDistributorStore) DeleteMasterDistributor(id string) error {
	query := `
	DELETE FROM master_distributors
	WHERE master_distributor_id = $1;
	`

	res, err := ms.db.Exec(query, id)
	if err != nil {
		return err
	}

	return checkRowsAffected(res)
}

// Get Master Distributor Wallet Balance
func (ms *PostgresMasterDistributorStore) GetMasterDistributorWalletBalance(id string) (float64, error) {
	query := `
		SELECT 
			master_distributor_wallet_balance
		FROM master_distributors
		WHERE master_distributor_id = $1;
	`
	var balance float64
	err := ms.db.QueryRow(
		query,
		id,
	).Scan(
		&balance,
	)

	return balance, err
}

// Update Master Distributor Aadhar Image
func (ms *PostgresMasterDistributorStore) UpdateMasterDistributorAadharImage(path, id string) error {
	query := `
		UPDATE master_distributors
		SET master_distributor_aadhar_image = $1,
		updated_at = CURRENT_TIMESTAMP
		WHERE master_distributor_id = $2;
	`

	res, err := ms.db.Exec(query, path, id)
	if err != nil {
		return err
	}

	return checkRowsAffected(res)
}

// Update Master Distributor Pan Image
func (ms *PostgresMasterDistributorStore) UpdateMasterDistributorPanImage(path, id string) error {
	query := `
		UPDATE master_distributors
		SET master_distributor_pan_image = $1,
		updated_at = CURRENT_TIMESTAMP
		WHERE master_distributor_id = $2;
	`

	res, err := ms.db.Exec(query, path, id)
	if err != nil {
		return err
	}

	return checkRowsAffected(res)
}

// Update Master Distributor Image
func (ms *PostgresMasterDistributorStore) UpdateMasterDistributorImage(path, id string) error {
	query := `
		UPDATE master_distributors
		SET master_distributor_image = $1,
		updated_at = CURRENT_TIMESTAMP
		WHERE master_distributor_id = $2;
	`

	res, err := ms.db.Exec(query, path, id)
	if err != nil {
		return err
	}

	return checkRowsAffected(res)
}
