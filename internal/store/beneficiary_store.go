package store

import (
	"database/sql"

	"github.com/levionstudio/fintech/internal/models"
	"github.com/levionstudio/fintech/internal/utils"
)

type PostgresBeneficiaryStore struct {
	db *sql.DB
}

func NewPostgresBeneficiaryStore(db *sql.DB) *PostgresBeneficiaryStore {
	return &PostgresBeneficiaryStore{db: db}
}

type BeneficiaryStore interface {
	CreateBeneficiary(b *models.BeneficiaryModel) error
	UpdateBeneficiary(b *models.BeneficiaryModel) error
	DeleteBeneficiary(beneficiaryID string) error
	GetBeneficiaries(mobileNumber string, p utils.PaginationParams) ([]models.BeneficiaryModel, error)
	VerifyBeneficiary(beneficiaryID string) error
}

// Create Beneficiary
func (bs *PostgresBeneficiaryStore) CreateBeneficiary(b *models.BeneficiaryModel) error {
	query := `
	INSERT INTO beneficiaries (mobile_number, bank_name, ifsc_code, account_number, beneficiary_name, beneficiary_phone)
	VALUES ($1, $2, $3, $4, $5, $6)
	RETURNING beneficiary_id, beneficiary_verified, created_at;
	`
	return bs.db.QueryRow(query,
		b.MobileNumber, b.BankName, b.IFSCCode,
		b.AccountNumber, b.BeneficiaryName, b.BeneficiaryPhone,
	).Scan(&b.BeneficiaryID, &b.BeneficiaryVerified, &b.CreatedAT)
}

// Update Beneficiary
func (bs *PostgresBeneficiaryStore) UpdateBeneficiary(b *models.BeneficiaryModel) error {
	query := `
	UPDATE beneficiaries
	SET mobile_number     = COALESCE(NULLIF($1, ''), mobile_number),
	    bank_name         = COALESCE(NULLIF($2, ''), bank_name),
	    ifsc_code         = COALESCE(NULLIF($3, ''), ifsc_code),
	    account_number    = COALESCE(NULLIF($4, ''), account_number),
	    beneficiary_name  = COALESCE(NULLIF($5, ''), beneficiary_name),
	    beneficiary_phone = COALESCE(NULLIF($6, ''), beneficiary_phone)
	WHERE beneficiary_id = $7;
	`
	res, err := bs.db.Exec(query,
		b.MobileNumber, b.BankName, b.IFSCCode,
		b.AccountNumber, b.BeneficiaryName, b.BeneficiaryPhone,
		b.BeneficiaryID,
	)
	if err != nil {
		return err
	}
	return checkRowsAffected(res)
}

// Delete Beneficiary
func (bs *PostgresBeneficiaryStore) DeleteBeneficiary(beneficiaryID string) error {
	res, err := bs.db.Exec(`DELETE FROM beneficiaries WHERE beneficiary_id = $1;`, beneficiaryID)
	if err != nil {
		return err
	}
	return checkRowsAffected(res)
}

// Verify Beneficiary
func (bs *PostgresBeneficiaryStore) VerifyBeneficiary(beneficiaryID string) error {
	res, err := bs.db.Exec(
		`UPDATE beneficiaries SET beneficiary_verified = TRUE WHERE beneficiary_id = $1;`,
		beneficiaryID,
	)
	if err != nil {
		return err
	}
	return checkRowsAffected(res)
}

// Get Beneficiaries
func (bs *PostgresBeneficiaryStore) GetBeneficiaries(mobileNumber string, p utils.PaginationParams) ([]models.BeneficiaryModel, error) {
	query := `
	SELECT beneficiary_id, mobile_number, bank_name, ifsc_code, account_number,
	       beneficiary_name, beneficiary_phone, beneficiary_verified, created_at
	FROM beneficiaries
	WHERE mobile_number = $1
	ORDER BY created_at DESC
	LIMIT $2 OFFSET $3;
	`
	rows, err := bs.db.Query(query, mobileNumber, p.Limit, p.Offset)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var beneficiaries []models.BeneficiaryModel
	for rows.Next() {
		var b models.BeneficiaryModel
		if err := rows.Scan(
			&b.BeneficiaryID, &b.MobileNumber, &b.BankName, &b.IFSCCode, &b.AccountNumber,
			&b.BeneficiaryName, &b.BeneficiaryPhone, &b.BeneficiaryVerified, &b.CreatedAT,
		); err != nil {
			return nil, err
		}
		beneficiaries = append(beneficiaries, b)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	if beneficiaries == nil {
		beneficiaries = []models.BeneficiaryModel{}
	}
	return beneficiaries, nil
}
