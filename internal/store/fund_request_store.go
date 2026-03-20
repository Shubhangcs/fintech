package store

import (
	"database/sql"
	"errors"
	"fmt"
	"time"

	"github.com/levionstudio/fintech/internal/models"
)

type PostgresFundRequestStore struct {
	db          *sql.DB
	walletStore WalletTransactionStore
}

func NewPostgresFundRequestStore(db *sql.DB, walletStore WalletTransactionStore) *PostgresFundRequestStore {
	return &PostgresFundRequestStore{db: db, walletStore: walletStore}
}

type FundRequestStore interface {
	MDRequestToAdmin(fr *models.FundRequestModel) error
	DistributorRequestToAdmin(fr *models.FundRequestModel) error
	DistributorRequestToMD(fr *models.FundRequestModel) error
	RetailerRequestToAdmin(fr *models.FundRequestModel) error
	RetailerRequestToMD(fr *models.FundRequestModel) error
	RetailerRequestToDistributor(fr *models.FundRequestModel) error
	ApproveFundRequest(fundRequestID int64) error
	RejectFundRequest(fundRequestID int64, rejectRemarks string) error
	GetFundRequestsByRequesterID(requesterID string, limit, offset int, startDate, endDate *time.Time) ([]models.FundRequestModel, error)
	GetFundRequestsByRequestToID(requestToID string, limit, offset int, startDate, endDate *time.Time) ([]models.FundRequestModel, error)
	GetAllFundRequests(limit, offset int, startDate, endDate *time.Time) ([]models.FundRequestModel, error)
}

// --- create implementations ---

func (fs *PostgresFundRequestStore) MDRequestToAdmin(fr *models.FundRequestModel) error {
	return fs.createRequest(fr)
}

func (fs *PostgresFundRequestStore) DistributorRequestToAdmin(fr *models.FundRequestModel) error {
	return fs.createRequest(fr)
}

func (fs *PostgresFundRequestStore) DistributorRequestToMD(fr *models.FundRequestModel) error {
	return fs.createRequest(fr)
}

func (fs *PostgresFundRequestStore) RetailerRequestToAdmin(fr *models.FundRequestModel) error {
	return fs.createRequest(fr)
}

func (fs *PostgresFundRequestStore) RetailerRequestToMD(fr *models.FundRequestModel) error {
	return fs.createRequest(fr)
}

func (fs *PostgresFundRequestStore) RetailerRequestToDistributor(fr *models.FundRequestModel) error {
	return fs.createRequest(fr)
}

func (fs *PostgresFundRequestStore) createRequest(fr *models.FundRequestModel) error {
	query := `
	INSERT INTO fund_requests (
		requester_id,
		request_to_id,
		amount,
		bank_name,
		request_date,
		utr_number,
		request_type,
		request_status,
		remarks
	) VALUES ($1, $2, $3, $4, $5, $6, $7, 'PENDING', $8)
	RETURNING fund_request_id, request_status, created_at, updated_at;
	`

	return fs.db.QueryRow(
		query,
		fr.RequesterID,
		fr.RequestToID,
		fr.Amount,
		fr.BankName,
		fr.RequestDate,
		fr.UTRNumber,
		fr.RequestType,
		fr.Remarks,
	).Scan(
		&fr.FundRequestID,
		&fr.RequestStatus,
		&fr.CreatedAT,
		&fr.UpdatedAT,
	)
}

// --- approve / reject ---

func (fs *PostgresFundRequestStore) ApproveFundRequest(fundRequestID int64) error {
	// 1. Fetch the fund request
	fr, err := fs.getFundRequestByID(fundRequestID)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return errors.New("fund request not found")
		}
		return err
	}

	if fr.RequestStatus != "PENDING" {
		return fmt.Errorf("fund request is already %s", fr.RequestStatus)
	}

	// 2. Resolve wallet tables from ID prefixes
	requestToWallet, err := walletInfoFromID(fr.RequestToID)
	if err != nil {
		return err
	}

	requesterWallet, err := walletInfoFromID(fr.RequesterID)
	if err != nil {
		return err
	}

	// 3. Begin transaction
	tx, err := fs.db.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()

	// 4. Atomically debit request_to — single UPDATE, checks balance in WHERE
	requestToBefore, requestToAfter, err := debitTx(tx, requestToWallet.table, requestToWallet.idCol, requestToWallet.balanceCol, fr.RequestToID, fr.Amount)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return checkExistsTx(tx, requestToWallet.table, requestToWallet.idCol, fr.RequestToID, "request_to user")
		}
		return err
	}

	// 5. Atomically credit requester
	requesterBefore, requesterAfter, err := creditTx(tx, requesterWallet.table, requesterWallet.idCol, requesterWallet.balanceCol, fr.RequesterID, fr.Amount)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return errors.New("requester not found")
		}
		return err
	}

	// 6. Update fund request status
	if err = fs.updateFundRequestStatusTx(tx, fundRequestID, "ACCEPTED", nil); err != nil {
		return err
	}

	refID := fmt.Sprintf("%d", fundRequestID)
	remarks := fmt.Sprintf("Fund request approved: %s", fr.Remarks)

	// 7. Wallet tx for request_to (debit)
	debitAmt := fr.Amount
	if err = fs.walletStore.CreateWalletTransactionTx(tx, &models.WalletTransactionModel{
		UserID: fr.RequestToID, ReferenceID: refID,
		DebitAmount: &debitAmt, BeforeBalance: requestToBefore, AfterBalance: requestToAfter,
		TransactionReason: "FUND_REQUEST", Remarks: remarks,
	}); err != nil {
		return err
	}

	// 8. Wallet tx for requester (credit)
	creditAmt := fr.Amount
	if err = fs.walletStore.CreateWalletTransactionTx(tx, &models.WalletTransactionModel{
		UserID: fr.RequesterID, ReferenceID: refID,
		CreditAmount: &creditAmt, BeforeBalance: requesterBefore, AfterBalance: requesterAfter,
		TransactionReason: "FUND_REQUEST", Remarks: remarks,
	}); err != nil {
		return err
	}

	return tx.Commit()
}

func (fs *PostgresFundRequestStore) RejectFundRequest(fundRequestID int64, rejectRemarks string) error {
	query := `
	UPDATE fund_requests
	SET request_status = 'REJECTED',
		reject_remarks = $1,
		updated_at     = CURRENT_TIMESTAMP
	WHERE fund_request_id = $2
	AND request_status    = 'PENDING';
	`

	res, err := fs.db.Exec(query, rejectRemarks, fundRequestID)
	if err != nil {
		return err
	}

	rowsAffected, err := res.RowsAffected()
	if err != nil {
		return err
	}
	if rowsAffected == 0 {
		return errors.New("fund request not found or already processed")
	}

	return nil
}

// --- get functions ---

// fundRequestSelectBase uses LATERAL subqueries instead of 8 LEFT JOINs.
const fundRequestSelectBase = `
SELECT
	fr.fund_request_id, fr.requester_id, fr.request_to_id, fr.amount, fr.bank_name,
	fr.request_date, fr.utr_number, fr.request_type, fr.request_status,
	fr.remarks, fr.reject_remarks, fr.created_at, fr.updated_at,
	COALESCE(q.name, '')  AS requester_name,
	q.business_name       AS requester_business_name,
	COALESCE(p.name, '')  AS request_to_name,
	p.business_name       AS request_to_business_name
FROM fund_requests fr
LEFT JOIN LATERAL (
	SELECT name, business_name FROM (
		SELECT admin_name AS name,            NULL::TEXT AS business_name              FROM admins            WHERE admin_id            = fr.requester_id
		UNION ALL
		SELECT master_distributor_name,       master_distributor_business_name         FROM master_distributors WHERE master_distributor_id = fr.requester_id
		UNION ALL
		SELECT distributor_name,              distributor_business_name                FROM distributors        WHERE distributor_id       = fr.requester_id
		UNION ALL
		SELECT retailer_name,                 retailer_business_name                   FROM retailers           WHERE retailer_id          = fr.requester_id
	) u LIMIT 1
) q ON TRUE
LEFT JOIN LATERAL (
	SELECT name, business_name FROM (
		SELECT admin_name AS name,            NULL::TEXT AS business_name              FROM admins            WHERE admin_id            = fr.request_to_id
		UNION ALL
		SELECT master_distributor_name,       master_distributor_business_name         FROM master_distributors WHERE master_distributor_id = fr.request_to_id
		UNION ALL
		SELECT distributor_name,              distributor_business_name                FROM distributors        WHERE distributor_id       = fr.request_to_id
		UNION ALL
		SELECT retailer_name,                 retailer_business_name                   FROM retailers           WHERE retailer_id          = fr.request_to_id
	) u LIMIT 1
) p ON TRUE
`

func (fs *PostgresFundRequestStore) GetFundRequestsByRequesterID(requesterID string, limit, offset int, startDate, endDate *time.Time) ([]models.FundRequestModel, error) {
	query := fundRequestSelectBase + `
	WHERE fr.requester_id = $1
	AND fr.created_at >= COALESCE($4, '-infinity'::TIMESTAMPTZ)
	AND fr.created_at <= COALESCE($5, 'infinity'::TIMESTAMPTZ)
	ORDER BY fr.created_at DESC
	LIMIT $2 OFFSET $3;
	`
	return scanFundRequests(fs.db, query, requesterID, limit, offset, startDate, endDate)
}

func (fs *PostgresFundRequestStore) GetFundRequestsByRequestToID(requestToID string, limit, offset int, startDate, endDate *time.Time) ([]models.FundRequestModel, error) {
	query := fundRequestSelectBase + `
	WHERE fr.request_to_id = $1
	AND fr.created_at >= COALESCE($4, '-infinity'::TIMESTAMPTZ)
	AND fr.created_at <= COALESCE($5, 'infinity'::TIMESTAMPTZ)
	ORDER BY fr.created_at DESC
	LIMIT $2 OFFSET $3;
	`
	return scanFundRequests(fs.db, query, requestToID, limit, offset, startDate, endDate)
}

func (fs *PostgresFundRequestStore) GetAllFundRequests(limit, offset int, startDate, endDate *time.Time) ([]models.FundRequestModel, error) {
	query := fundRequestSelectBase + `
	WHERE fr.created_at >= COALESCE($3, '-infinity'::TIMESTAMPTZ)
	AND fr.created_at <= COALESCE($4, 'infinity'::TIMESTAMPTZ)
	ORDER BY fr.created_at DESC
	LIMIT $1 OFFSET $2;
	`
	return scanFundRequests(fs.db, query, limit, offset, startDate, endDate)
}

// --- helpers ---

func (fs *PostgresFundRequestStore) getFundRequestByID(id int64) (*models.FundRequestModel, error) {
	query := `
	SELECT
		fund_request_id, requester_id, request_to_id, amount, bank_name,
		request_date, utr_number, request_type, request_status,
		remarks, reject_remarks, created_at, updated_at
	FROM fund_requests
	WHERE fund_request_id = $1;
	`

	var fr models.FundRequestModel
	err := fs.db.QueryRow(query, id).Scan(
		&fr.FundRequestID, &fr.RequesterID, &fr.RequestToID, &fr.Amount, &fr.BankName,
		&fr.RequestDate, &fr.UTRNumber, &fr.RequestType, &fr.RequestStatus,
		&fr.Remarks, &fr.RejectRemarks, &fr.CreatedAT, &fr.UpdatedAT,
	)

	return &fr, err
}

func (fs *PostgresFundRequestStore) updateFundRequestStatusTx(tx *sql.Tx, id int64, status string, rejectRemarks *string) error {
	query := `
	UPDATE fund_requests
	SET request_status = $1,
		reject_remarks = $2,
		updated_at     = CURRENT_TIMESTAMP
	WHERE fund_request_id = $3;
	`

	_, err := tx.Exec(query, status, rejectRemarks, id)
	return err
}

func scanFundRequests(db *sql.DB, query string, args ...any) ([]models.FundRequestModel, error) {
	rows, err := db.Query(query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	requests := []models.FundRequestModel{}
	for rows.Next() {
		var fr models.FundRequestModel
		if err = rows.Scan(
			&fr.FundRequestID, &fr.RequesterID, &fr.RequestToID, &fr.Amount, &fr.BankName,
			&fr.RequestDate, &fr.UTRNumber, &fr.RequestType, &fr.RequestStatus,
			&fr.Remarks, &fr.RejectRemarks, &fr.CreatedAT, &fr.UpdatedAT,
			&fr.RequesterName, &fr.RequesterBusinessName,
			&fr.RequestToName, &fr.RequestToBusinessName,
		); err != nil {
			return nil, err
		}
		requests = append(requests, fr)
	}

	return requests, rows.Err()
}
