package store

import (
	"database/sql"
	"errors"
	"fmt"
	"time"

	"github.com/levionstudio/fintech/internal/models"
)

type PostgresFundRequestStore struct {
	db *sql.DB
}

func NewPostgresFundRequestStore(db *sql.DB) *PostgresFundRequestStore {
	return &PostgresFundRequestStore{db: db}
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
	requesterWallet, err := walletInfoFromID(fr.RequesterID)
	if err != nil {
		return err
	}

	requestToWallet, err := walletInfoFromID(fr.RequestToID)
	if err != nil {
		return err
	}

	// 3. Begin transaction
	tx, err := fs.db.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()

	// 4. Get balances
	requestToBefore, err := getBalanceTx(tx, requestToWallet.table, requestToWallet.idCol, requestToWallet.balanceCol, fr.RequestToID)
	if err != nil {
		return errors.New("request_to user not found")
	}

	if requestToBefore < fr.Amount {
		return errors.New("insufficient balance")
	}

	requesterBefore, err := getBalanceTx(tx, requesterWallet.table, requesterWallet.idCol, requesterWallet.balanceCol, fr.RequesterID)
	if err != nil {
		return errors.New("requester not found")
	}

	requestToAfter := requestToBefore - fr.Amount
	requesterAfter := requesterBefore + fr.Amount

	// 5. Deduct from request_to
	if err = updateBalanceTx(tx, requestToWallet.table, requestToWallet.idCol, requestToWallet.balanceCol, fr.RequestToID, requestToAfter); err != nil {
		return err
	}

	// 6. Credit requester
	if err = updateBalanceTx(tx, requesterWallet.table, requesterWallet.idCol, requesterWallet.balanceCol, fr.RequesterID, requesterAfter); err != nil {
		return err
	}

	// 7. Update fund request status
	if err = fs.updateFundRequestStatusTx(tx, fundRequestID, "ACCEPTED", nil); err != nil {
		return err
	}

	refID := fmt.Sprintf("%d", fundRequestID)
	remarks := fmt.Sprintf("Fund request approved: %s", fr.Remarks)

	// 8. Wallet tx for request_to (debit)
	debitAmt := fr.Amount
	if err = insertWalletTransactionTx(tx, fr.RequestToID, refID, nil, &debitAmt, requestToBefore, requestToAfter, "FUND_REQUEST", remarks); err != nil {
		return err
	}

	// 9. Wallet tx for requester (credit)
	creditAmt := fr.Amount
	if err = insertWalletTransactionTx(tx, fr.RequesterID, refID, &creditAmt, nil, requesterBefore, requesterAfter, "FUND_REQUEST", remarks); err != nil {
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

// fundRequestSelectBase selects all columns including user names for both parties.
const fundRequestSelectBase = `
SELECT
	fr.fund_request_id, fr.requester_id, fr.request_to_id, fr.amount, fr.bank_name,
	fr.request_date, fr.utr_number, fr.request_type, fr.request_status,
	fr.remarks, fr.reject_remarks, fr.created_at, fr.updated_at,
	COALESCE(qa.admin_name, qmd.master_distributor_name, qd.distributor_name, qr.retailer_name, '') AS requester_name,
	COALESCE(qmd.master_distributor_business_name, qd.distributor_business_name, qr.retailer_business_name) AS requester_business_name,
	COALESCE(pa.admin_name, pmd.master_distributor_name, pd.distributor_name, pr.retailer_name, '') AS request_to_name,
	COALESCE(pmd.master_distributor_business_name, pd.distributor_business_name, pr.retailer_business_name) AS request_to_business_name
FROM fund_requests fr
LEFT JOIN admins qa               ON fr.requester_id = qa.admin_id
LEFT JOIN master_distributors qmd ON fr.requester_id = qmd.master_distributor_id
LEFT JOIN distributors qd         ON fr.requester_id = qd.distributor_id
LEFT JOIN retailers qr            ON fr.requester_id = qr.retailer_id
LEFT JOIN admins pa               ON fr.request_to_id = pa.admin_id
LEFT JOIN master_distributors pmd ON fr.request_to_id = pmd.master_distributor_id
LEFT JOIN distributors pd         ON fr.request_to_id = pd.distributor_id
LEFT JOIN retailers pr            ON fr.request_to_id = pr.retailer_id
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

	var requests []models.FundRequestModel
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
