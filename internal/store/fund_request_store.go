package store

import (
	"database/sql"
	"errors"
	"fmt"

	"github.com/levionstudio/fintech/internal/models"
	"github.com/levionstudio/fintech/internal/utils"
)

type PostgresFundRequestStore struct {
	db          *sql.DB
	walletStore WalletTransactionStore
}

func NewPostgresFundRequestStore(db *sql.DB, walletStore WalletTransactionStore) *PostgresFundRequestStore {
	return &PostgresFundRequestStore{db: db, walletStore: walletStore}
}

type FundRequestStore interface {
	CreateFundRequest(fr *models.FundRequestModel) error
	ApproveFundRequest(fundRequestID int64) error
	RejectFundRequest(fr *models.FundRequestModel) error
	UploadFundRequestRecipt(id int64, recipt string) error
	GetFundRequestsByRequesterID(requesterID string, p utils.QueryParams) ([]models.FundRequestModel, error)
	GetFundRequestsByRequestToID(requestToID string, p utils.QueryParams) ([]models.FundRequestModel, error)
	GetAllFundRequests(p utils.QueryParams) ([]models.FundRequestModel, error)
}

// Create Fund Request
func (fs *PostgresFundRequestStore) CreateFundRequest(fr *models.FundRequestModel) error {
	const q = `
	INSERT INTO fund_requests (
		requester_id, request_to_id, amount, bank_name,
		request_date, utr_number, request_type, request_status, remarks
	) VALUES ($1, $2, $3, $4, $5, $6, $7, 'PENDING', $8)
	RETURNING fund_request_id, request_status, created_at, updated_at;
	`
	return fs.db.QueryRow(q,
		fr.RequesterID, fr.RequestToID, fr.Amount, fr.BankName,
		fr.RequestDate, fr.UTRNumber, fr.RequestType, fr.Remarks,
	).Scan(&fr.FundRequestID, &fr.RequestStatus, &fr.CreatedAT, &fr.UpdatedAT)
}

// Approve Fund Request
func (fs *PostgresFundRequestStore) ApproveFundRequest(fundRequestID int64) error {
	tx, err := fs.db.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()

	var fr models.FundRequestModel
	err = tx.QueryRow(`
		SELECT fund_request_id, requester_id, request_to_id, amount, request_status, remarks
		FROM fund_requests WHERE fund_request_id = $1 FOR UPDATE
	`, fundRequestID).Scan(
		&fr.FundRequestID, &fr.RequesterID, &fr.RequestToID,
		&fr.Amount, &fr.RequestStatus, &fr.Remarks,
	)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return errors.New("fund request not found")
		}
		return err
	}
	if fr.RequestStatus != "PENDING" {
		return fmt.Errorf("fund request is already %s", fr.RequestStatus)
	}

	requestToInfo, err := getUserTableInfo(fr.RequestToID)
	if err != nil {
		return err
	}
	requesterInfo, err := getUserTableInfo(fr.RequesterID)
	if err != nil {
		return err
	}

	refID := fmt.Sprintf("%d", fundRequestID)
	remarks := fmt.Sprintf("Fund request approved: %s", fr.Remarks)

	// Debit request_to — atomically checks balance, also creates wallet transaction entry
	if err = debitTx(tx, transaction{
		UserID: fr.RequestToID, ReferenceID: refID,
		Amount: fr.Amount, Reason: "FUND_REQUEST", Remarks: remarks,
		userTableInfo: *requestToInfo,
	}, fs.walletStore); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return checkExistsTx(tx, requestToInfo.TableName, requestToInfo.IDColumnName, fr.RequestToID, "request_to user")
		}
		return err
	}

	// Credit requester — also creates wallet transaction entry
	if err = creditTx(tx, transaction{
		UserID: fr.RequesterID, ReferenceID: refID,
		Amount: fr.Amount, Reason: "FUND_REQUEST", Remarks: remarks,
		userTableInfo: *requesterInfo,
	}, fs.walletStore); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return errors.New("requester not found")
		}
		return err
	}

	// Updating The Fund Request Status To Accepted
	res, err := tx.Exec(`
		UPDATE fund_requests SET request_status = 'ACCEPTED', updated_at = CURRENT_TIMESTAMP
		WHERE fund_request_id = $1 AND request_status = 'PENDING'
	`, fundRequestID)
	if err != nil {
		return err
	}
	n, err := res.RowsAffected()
	if err != nil {
		return err
	}
	if n == 0 {
		return errors.New("fund request not found or already processed")
	}

	return tx.Commit()
}

// Upload Fund Request Recipt
func (fs *PostgresFundRequestStore) UploadFundRequestRecipt(id int64, recipt string) error {
	res, err := fs.db.Exec(`
		UPDATE fund_requests SET recipt = $1, updated_at = CURRENT_TIMESTAMP
		WHERE fund_request_id = $2 AND request_type = 'NORMAL'
	`, recipt, id)
	if err != nil {
		return err
	}
	n, err := res.RowsAffected()
	if err != nil {
		return err
	}
	if n == 0 {
		return errors.New("fund request not found or not a NORMAL type request")
	}
	return nil
}

// Reject Fund Request
func (fs *PostgresFundRequestStore) RejectFundRequest(fr *models.FundRequestModel) error {
	res, err := fs.db.Exec(`
		UPDATE fund_requests
		SET request_status = 'REJECTED', reject_remarks = $1, updated_at = CURRENT_TIMESTAMP
		WHERE fund_request_id = $2 AND request_status = 'PENDING'
	`, fr.RejectRemarks, fr.FundRequestID)
	if err != nil {
		return err
	}
	n, err := res.RowsAffected()
	if err != nil {
		return err
	}
	if n == 0 {
		return errors.New("fund request not found or already processed")
	}
	return nil
}

// fundRequestSelectBase uses LATERAL subqueries instead of 8 LEFT JOINs.
const fundRequestSelectBase = `
SELECT
	fr.fund_request_id, fr.requester_id, fr.request_to_id, fr.amount, fr.bank_name,
	fr.request_date, fr.utr_number, fr.request_type, fr.request_status,
	fr.remarks, fr.reject_remarks, fr.recipt, fr.created_at, fr.updated_at,
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

// Get Fund Requests By Requester ID
func (fs *PostgresFundRequestStore) GetFundRequestsByRequesterID(requesterID string, p utils.QueryParams) ([]models.FundRequestModel, error) {
	q := fundRequestSelectBase + `
	WHERE fr.requester_id = $1
	AND fr.created_at >= COALESCE($4, '-infinity'::TIMESTAMPTZ)
	AND fr.created_at <= COALESCE($5, 'infinity'::TIMESTAMPTZ)
	ORDER BY fr.created_at DESC
	LIMIT $2 OFFSET $3;
	`
	return scanFundRequests(fs.db, q, requesterID, p.Limit, p.Offset, p.StartDate, p.EndDate)
}

// Get Fund Requests By Request To ID
func (fs *PostgresFundRequestStore) GetFundRequestsByRequestToID(requestToID string, p utils.QueryParams) ([]models.FundRequestModel, error) {
	q := fundRequestSelectBase + `
	WHERE fr.request_to_id = $1
	AND fr.created_at >= COALESCE($4, '-infinity'::TIMESTAMPTZ)
	AND fr.created_at <= COALESCE($5, 'infinity'::TIMESTAMPTZ)
	ORDER BY fr.created_at DESC
	LIMIT $2 OFFSET $3;
	`
	return scanFundRequests(fs.db, q, requestToID, p.Limit, p.Offset, p.StartDate, p.EndDate)
}

// Get All Fund Requests
func (fs *PostgresFundRequestStore) GetAllFundRequests(p utils.QueryParams) ([]models.FundRequestModel, error) {
	q := fundRequestSelectBase + `
	WHERE fr.created_at >= COALESCE($3, '-infinity'::TIMESTAMPTZ)
	AND fr.created_at <= COALESCE($4, 'infinity'::TIMESTAMPTZ)
	ORDER BY fr.created_at DESC
	LIMIT $1 OFFSET $2;
	`
	return scanFundRequests(fs.db, q, p.Limit, p.Offset, p.StartDate, p.EndDate)
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
			&fr.Remarks, &fr.RejectRemarks, &fr.Recipt, &fr.CreatedAT, &fr.UpdatedAT,
			&fr.RequesterName, &fr.RequesterBusinessName,
			&fr.RequestToName, &fr.RequestToBusinessName,
		); err != nil {
			return nil, err
		}
		requests = append(requests, fr)
	}
	return requests, rows.Err()
}
