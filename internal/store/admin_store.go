package store

import (
	"database/sql"
	"errors"

	"github.com/levionstudio/fintech/internal/models"
)

type PostgresAdminStore struct {
	db *sql.DB
}

func NewPostgresAdminStore(db *sql.DB) *PostgresAdminStore {
	return &PostgresAdminStore{
		db: db,
	}
}

type AdminStore interface {
	CreateAdmin(admin *models.AdminModel) error
	UpdateAdminDetails(admin *models.AdminModel) error
	UpdateAdminPassword(admin *models.AdminModel) error
	UpdateAdminWalletBalance(admin *models.AdminModel) error
	DeleteAdmin(id string) error
	GetAdminByID(id string) (*models.AdminModel, error)
	GetAdminDetailsForLogin(admin *models.AdminModel) error
	GetAdmins(limit, offset int) ([]models.AdminModel, error)
	GetAdminsForDropdown() ([]models.DropdownItem, error)
	GetAdminWalletBalance(id string) (float64, error)
}

// Create Admin
func (as *PostgresAdminStore) CreateAdmin(admin *models.AdminModel) error {
	query := `
	INSERT INTO admins(
		admin_name,
		admin_email,
		admin_phone,
		admin_password
	) VALUES (
		$1, $2, $3, $4 
	)
	RETURNING admin_id, created_at, updated_at;
	`

	err := as.db.QueryRow(
		query,
		admin.AdminName,
		admin.AdminEmail,
		admin.AdminPhone,
		admin.AdminPassword,
	).Scan(
		&admin.AdminID,
		&admin.CreatedAT,
		&admin.UpdatedAT,
	)

	return err
}

// Update Admin Details
func (as *PostgresAdminStore) UpdateAdminDetails(admin *models.AdminModel) error {
	query := `
	UPDATE admins
	SET admin_name  = COALESCE(NULLIF($1, ''), admin_name),
	admin_email     = COALESCE(NULLIF($2, ''), admin_email),
	admin_phone     = COALESCE(NULLIF($3, ''), admin_phone),
	updated_at      = CURRENT_TIMESTAMP
	WHERE admin_id  = $4;
	`

	res, err := as.db.Exec(query, admin.AdminName, admin.AdminEmail, admin.AdminPhone, admin.AdminID)
	if err != nil {
		return err
	}

	return checkRowsAffected(res)
}

// Update Admin Password
func (as *PostgresAdminStore) UpdateAdminPassword(admin *models.AdminModel) error {
	query := `
	UPDATE admins
	SET admin_password = $1,
	updated_at         = CURRENT_TIMESTAMP
	WHERE admin_id     = $2;
	`

	res, err := as.db.Exec(query, admin.AdminPassword, admin.AdminID)
	if err != nil {
		return err
	}

	return checkRowsAffected(res)
}

// Update Admin Wallet Balance
func (as *PostgresAdminStore) UpdateAdminWalletBalance(admin *models.AdminModel) error {
	query := `
	UPDATE admins
	SET admin_wallet_balance = admin_wallet_balance + $1,
	updated_at = CURRENT_TIMESTAMP
	WHERE admin_id = $2;
	`

	res, err := as.db.Exec(query, admin.AdminWalletBalance, admin.AdminID)
	if err != nil {
		return err
	}

	return checkRowsAffected(res)
}

// Delete Admin
func (as *PostgresAdminStore) DeleteAdmin(id string) error {
	query := `
	DELETE FROM admins
	WHERE admin_id = $1;
	`

	res, err := as.db.Exec(query, id)
	if err != nil {
		return err
	}

	return checkRowsAffected(res)
}

// Get Admin By ID
func (as *PostgresAdminStore) GetAdminByID(id string) (*models.AdminModel, error) {
	query := `
	SELECT 
		admin_id,
		admin_name,
		admin_email,
		admin_phone,
		admin_password,
		admin_wallet_balance,
		created_at,
		updated_at
	FROM admins
	WHERE admin_id = $1;
	`
	var admin models.AdminModel
	err := as.db.QueryRow(query, id).Scan(
		&admin.AdminID,
		&admin.AdminName,
		&admin.AdminEmail,
		&admin.AdminPhone,
		&admin.AdminPassword,
		&admin.AdminWalletBalance,
		&admin.CreatedAT,
		&admin.UpdatedAT,
	)

	return &admin, err
}

// Get Admin Details For Login
func (as *PostgresAdminStore) GetAdminDetailsForLogin(admin *models.AdminModel) error {
	query := `
	SELECT
		admin_id,
		admin_name
	FROM admins
	WHERE admin_id = $1
	AND admin_password = $2;
	`

	err := as.db.QueryRow(query, admin.AdminID, admin.AdminPassword).Scan(
		&admin.AdminID,
		&admin.AdminName,
	)
	if err != nil && errors.Is(err, sql.ErrNoRows) {
		return errors.New("invalid credentials")
	}

	return err
}

// Get Admins For Dropdown Handler
func (as *PostgresAdminStore) GetAdminsForDropdown() ([]models.DropdownItem, error) {
	query := `SELECT admin_id, admin_name FROM admins ORDER BY admin_name;`
	return scanDropdown(as.db, query)
}

// Get Admins
func (as *PostgresAdminStore) GetAdmins(limit, offset int) ([]models.AdminModel, error) {
	query := `
	SELECT 
		admin_id,
		admin_name,
		admin_email,
		admin_phone,
		admin_password,
		admin_wallet_balance,
		created_at,
		updated_at
	FROM admins
	ORDER BY created_at DESC
	LIMIT $1 OFFSET $2;
	`

	res, err := as.db.Query(query, limit, offset)

	if err != nil && errors.Is(err, sql.ErrNoRows) {
		return []models.AdminModel{}, nil
	}

	if err != nil {
		return nil, err
	}
	defer res.Close()

	var admins []models.AdminModel
	for res.Next() {
		var a models.AdminModel
		err = res.Scan(
			&a.AdminID,
			&a.AdminName,
			&a.AdminEmail,
			&a.AdminPhone,
			&a.AdminPassword,
			&a.AdminWalletBalance,
			&a.CreatedAT,
			&a.UpdatedAT,
		)

		if err != nil {
			return nil, err
		}

		admins = append(admins, a)
	}

	return admins, res.Err()
}

// Get Admin Wallet Balance
func (as *PostgresAdminStore) GetAdminWalletBalance(id string) (float64, error) {
	query := `
		SELECT 
			admin_wallet_balance
		FROM admins
		WHERE admin_id = $1;
	`
	var balance float64
	err := as.db.QueryRow(
		query,
		id,
	).Scan(
		&balance,
	)

	return balance, err
}
