package store

import (
	"database/sql"

	"github.com/levionstudio/fintech/internal/models"
)

type PostgresBankStore struct {
	db *sql.DB
}

func NewPostgresBankStore(db *sql.DB) *PostgresBankStore {
	return &PostgresBankStore{db: db}
}

type BankStore interface {
	CreateBank(bank *models.BankModel) error
	UpdateBank(bankID int64, bank *models.BankModel) error
	DeleteBank(bankID int64) error
	GetAllBanks() ([]models.BankModel, error)

	CreateAdminBank(adminBank *models.AdminBankModel) error
	UpdateAdminBank(adminBankID int64, adminBank *models.AdminBankModel) error
	DeleteAdminBank(adminBankID int64) error
	GetAllAdminBanks() ([]models.AdminBankModel, error)
}

// --- bank ---

func (bs *PostgresBankStore) CreateBank(bank *models.BankModel) error {
	query := `
	INSERT INTO banks (bank_name, ifsc_code)
	VALUES ($1, $2)
	RETURNING bank_id;
	`
	return bs.db.QueryRow(query, bank.BankName, bank.IFSCCode).Scan(&bank.BankID)
}

func (bs *PostgresBankStore) UpdateBank(bankID int64, bank *models.BankModel) error {
	query := `
	UPDATE banks
	SET bank_name = COALESCE(NULLIF($1, ''), bank_name),
	    ifsc_code  = COALESCE(NULLIF($2, ''), ifsc_code)
	WHERE bank_id = $3;
	`
	res, err := bs.db.Exec(query, bank.BankName, bank.IFSCCode, bankID)
	if err != nil {
		return err
	}
	return checkRowsAffected(res)
}

func (bs *PostgresBankStore) DeleteBank(bankID int64) error {
	res, err := bs.db.Exec(`DELETE FROM banks WHERE bank_id = $1;`, bankID)
	if err != nil {
		return err
	}
	return checkRowsAffected(res)
}

func (bs *PostgresBankStore) GetAllBanks() ([]models.BankModel, error) {
	rows, err := bs.db.Query(`SELECT bank_id, bank_name, ifsc_code FROM banks ORDER BY bank_name;`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var banks []models.BankModel
	for rows.Next() {
		var b models.BankModel
		if err := rows.Scan(&b.BankID, &b.BankName, &b.IFSCCode); err != nil {
			return nil, err
		}
		banks = append(banks, b)
	}
	return banks, rows.Err()
}

// --- admin bank ---

func (bs *PostgresBankStore) CreateAdminBank(adminBank *models.AdminBankModel) error {
	query := `
	INSERT INTO admin_banks (admin_id, admin_bank_name, admin_bank_account_number, admin_bank_ifsc_code)
	VALUES ($1, $2, $3, $4)
	RETURNING admin_bank_id;
	`
	return bs.db.QueryRow(query,
		adminBank.AdminID,
		adminBank.AdminBankName,
		adminBank.AdminBankAccountNumber,
		adminBank.AdminBankIFSCCode,
	).Scan(&adminBank.AdminBankID)
}

func (bs *PostgresBankStore) UpdateAdminBank(adminBankID int64, adminBank *models.AdminBankModel) error {
	query := `
	UPDATE admin_banks
	SET admin_bank_name           = COALESCE(NULLIF($1, ''), admin_bank_name),
	    admin_bank_account_number = COALESCE(NULLIF($2, ''), admin_bank_account_number),
	    admin_bank_ifsc_code      = COALESCE(NULLIF($3, ''), admin_bank_ifsc_code)
	WHERE admin_bank_id = $4;
	`
	res, err := bs.db.Exec(query,
		adminBank.AdminBankName,
		adminBank.AdminBankAccountNumber,
		adminBank.AdminBankIFSCCode,
		adminBankID,
	)
	if err != nil {
		return err
	}
	return checkRowsAffected(res)
}

func (bs *PostgresBankStore) DeleteAdminBank(adminBankID int64) error {
	res, err := bs.db.Exec(`DELETE FROM admin_banks WHERE admin_bank_id = $1;`, adminBankID)
	if err != nil {
		return err
	}
	return checkRowsAffected(res)
}

func (bs *PostgresBankStore) GetAllAdminBanks() ([]models.AdminBankModel, error) {
	query := `
	SELECT admin_bank_id, admin_id, admin_bank_name, admin_bank_account_number, admin_bank_ifsc_code
	FROM admin_banks
	ORDER BY admin_bank_name;
	`
	rows, err := bs.db.Query(query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var banks []models.AdminBankModel
	for rows.Next() {
		var ab models.AdminBankModel
		if err := rows.Scan(
			&ab.AdminBankID, &ab.AdminID,
			&ab.AdminBankName, &ab.AdminBankAccountNumber, &ab.AdminBankIFSCCode,
		); err != nil {
			return nil, err
		}
		banks = append(banks, ab)
	}
	return banks, rows.Err()
}
