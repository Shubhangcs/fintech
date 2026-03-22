package store

import (
	"database/sql"
	"errors"
	"fmt"

	"github.com/levionstudio/fintech/internal/models"
	"github.com/levionstudio/fintech/internal/utils"
)

type PostgresFundTransferStore struct {
	db          *sql.DB
	walletStore WalletTransactionStore
}

func NewPostgresFundTransferStore(db *sql.DB, walletStore WalletTransactionStore) *PostgresFundTransferStore {
	return &PostgresFundTransferStore{
		db:          db,
		walletStore: walletStore,
	}
}

type FundTransferStore interface {
	AdminToMD(ft *models.FundTransferModel) error
	AdminToDistributor(ft *models.FundTransferModel) error
	AdminToRetailer(ft *models.FundTransferModel) error
	MDToDistributor(ft *models.FundTransferModel) error
	MDToRetailer(ft *models.FundTransferModel) error
	DistributorToRetailer(ft *models.FundTransferModel) error
	GetFundTransfersByTransfererID(transfererID string, p utils.QueryParams) ([]models.FundTransferModel, error)
	GetFundTransfersByReceiverID(receiverID string, p utils.QueryParams) ([]models.FundTransferModel, error)
	GetAllFundTransfers(p utils.QueryParams) ([]models.FundTransferModel, error)
}

// Admin To MD Fund Transfer
func (fs *PostgresFundTransferStore) AdminToMD(ft *models.FundTransferModel) error {
	return fs.transfer(ft)
}

// Admin To Distributor Fund Transfer
func (fs *PostgresFundTransferStore) AdminToDistributor(ft *models.FundTransferModel) error {
	return fs.transfer(ft)
}

// Admin To Retailer Fund Transfer
func (fs *PostgresFundTransferStore) AdminToRetailer(ft *models.FundTransferModel) error {
	return fs.transfer(ft)
}

// MD To Distributor Fund Transfer
func (fs *PostgresFundTransferStore) MDToDistributor(ft *models.FundTransferModel) error {
	return fs.transfer(ft)
}

// MD To Retailer Fund Transfer
func (fs *PostgresFundTransferStore) MDToRetailer(ft *models.FundTransferModel) error {
	return fs.transfer(ft)
}

// Distributor To Retailer Fund Transfer
func (fs *PostgresFundTransferStore) DistributorToRetailer(ft *models.FundTransferModel) error {
	return fs.transfer(ft)
}

func (fs *PostgresFundTransferStore) transfer(ft *models.FundTransferModel) error {
	senderInfo, err := getUserTableInfo(ft.FundTransfererID)
	if err != nil {
		return err
	}
	receiverInfo, err := getUserTableInfo(ft.FundReceiverID)
	if err != nil {
		return err
	}

	tx, err := fs.db.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()

	// 1. Insert fund_transfer record first to get the reference ID for wallet entries
	ft.FundTransferStatus = "SUCCESS"
	if err = insertFundTransferTx(tx, ft); err != nil {
		return err
	}

	refID := fmt.Sprintf("%d", ft.FundTransferID)

	// 2. Debit sender — atomically checks balance, also creates wallet transaction entry
	if err = debitTx(tx, transaction{
		UserID: ft.FundTransfererID, ReferenceID: refID,
		Amount: ft.Amount, Reason: "FUND_TRANSFER", Remarks: ft.Remarks,
		userTableInfo: *senderInfo,
	}, fs.walletStore); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return checkExistsTx(tx, senderInfo.TableName, senderInfo.IDColumnName, ft.FundTransfererID, "sender")
		}
		return err
	}

	// 3. Credit receiver — also creates wallet transaction entry
	if err = creditTx(tx, transaction{
		UserID: ft.FundReceiverID, ReferenceID: refID,
		Amount: ft.Amount, Reason: "FUND_TRANSFER", Remarks: ft.Remarks,
		userTableInfo: *receiverInfo,
	}, fs.walletStore); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return errors.New("receiver not found")
		}
		return err
	}

	return tx.Commit()
}

func insertFundTransferTx(tx *sql.Tx, ft *models.FundTransferModel) error {
	const q = `
	INSERT INTO fund_transfers (fund_transferer_id, fund_receiver_id, amount, fund_transfer_status, remarks)
	VALUES ($1, $2, $3, $4, $5)
	RETURNING fund_transfer_id, created_at;
	`
	return tx.QueryRow(q,
		ft.FundTransfererID, ft.FundReceiverID, ft.Amount, ft.FundTransferStatus, ft.Remarks,
	).Scan(&ft.FundTransferID, &ft.CreatedAT)
}

const fundTransferSelectBase = `
SELECT
	ft.fund_transfer_id,
	ft.fund_transferer_id,
	ft.fund_receiver_id,
	ft.amount,
	ft.fund_transfer_status,
	ft.remarks,
	ft.created_at,
	COALESCE(t.name, '')   AS transferer_name,
	t.business_name        AS transferer_business_name,
	COALESCE(rec.name, '') AS receiver_name,
	rec.business_name      AS receiver_business_name
FROM fund_transfers ft
LEFT JOIN LATERAL (
	SELECT name, business_name FROM (
		SELECT admin_name AS name,             NULL::TEXT AS business_name               FROM admins            WHERE admin_id            = ft.fund_transferer_id
		UNION ALL
		SELECT master_distributor_name,        master_distributor_business_name          FROM master_distributors WHERE master_distributor_id = ft.fund_transferer_id
		UNION ALL
		SELECT distributor_name,               distributor_business_name                 FROM distributors        WHERE distributor_id       = ft.fund_transferer_id
		UNION ALL
		SELECT retailer_name,                  retailer_business_name                    FROM retailers           WHERE retailer_id          = ft.fund_transferer_id
	) u LIMIT 1
) t ON TRUE
LEFT JOIN LATERAL (
	SELECT name, business_name FROM (
		SELECT admin_name AS name,             NULL::TEXT AS business_name               FROM admins            WHERE admin_id            = ft.fund_receiver_id
		UNION ALL
		SELECT master_distributor_name,        master_distributor_business_name          FROM master_distributors WHERE master_distributor_id = ft.fund_receiver_id
		UNION ALL
		SELECT distributor_name,               distributor_business_name                 FROM distributors        WHERE distributor_id       = ft.fund_receiver_id
		UNION ALL
		SELECT retailer_name,                  retailer_business_name                    FROM retailers           WHERE retailer_id          = ft.fund_receiver_id
	) u LIMIT 1
) rec ON TRUE
`

// Get Fund Transfer By Transferer ID
func (fs *PostgresFundTransferStore) GetFundTransfersByTransfererID(transfererID string, p utils.QueryParams) ([]models.FundTransferModel, error) {
	q := fundTransferSelectBase + `
	WHERE ft.fund_transferer_id = $1
	AND ft.created_at >= COALESCE($4, '-infinity'::TIMESTAMPTZ)
	AND ft.created_at <= COALESCE($5, 'infinity'::TIMESTAMPTZ)
	ORDER BY ft.created_at DESC
	LIMIT $2 OFFSET $3;
	`
	return scanFundTransfers(fs.db, q, transfererID, p.Limit, p.Offset, p.StartDate, p.EndDate)
}

// Get Fund Transfer By Receiver ID
func (fs *PostgresFundTransferStore) GetFundTransfersByReceiverID(receiverID string, p utils.QueryParams) ([]models.FundTransferModel, error) {
	q := fundTransferSelectBase + `
	WHERE ft.fund_receiver_id = $1
	AND ft.created_at >= COALESCE($4, '-infinity'::TIMESTAMPTZ)
	AND ft.created_at <= COALESCE($5, 'infinity'::TIMESTAMPTZ)
	ORDER BY ft.created_at DESC
	LIMIT $2 OFFSET $3;
	`
	return scanFundTransfers(fs.db, q, receiverID, p.Limit, p.Offset, p.StartDate, p.EndDate)
}

// Get All Fund Transfers
func (fs *PostgresFundTransferStore) GetAllFundTransfers(p utils.QueryParams) ([]models.FundTransferModel, error) {
	q := fundTransferSelectBase + `
	WHERE ft.created_at >= COALESCE($3, '-infinity'::TIMESTAMPTZ)
	AND ft.created_at <= COALESCE($4, 'infinity'::TIMESTAMPTZ)
	ORDER BY ft.created_at DESC
	LIMIT $1 OFFSET $2;
	`
	return scanFundTransfers(fs.db, q, p.Limit, p.Offset, p.StartDate, p.EndDate)
}

func scanFundTransfers(db *sql.DB, query string, args ...any) ([]models.FundTransferModel, error) {
	rows, err := db.Query(query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	transfers := []models.FundTransferModel{}
	for rows.Next() {
		var ft models.FundTransferModel
		if err = rows.Scan(
			&ft.FundTransferID,
			&ft.FundTransfererID,
			&ft.FundReceiverID,
			&ft.Amount,
			&ft.FundTransferStatus,
			&ft.Remarks,
			&ft.CreatedAT,
			&ft.TransfererName,
			&ft.TransfererBusinessName,
			&ft.ReceiverName,
			&ft.ReceiverBusinessName,
		); err != nil {
			return nil, err
		}
		transfers = append(transfers, ft)
	}
	return transfers, rows.Err()
}
