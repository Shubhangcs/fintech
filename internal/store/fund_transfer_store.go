package store

import (
	"database/sql"
	"errors"
	"fmt"
	"time"

	"github.com/levionstudio/fintech/internal/models"
)

type PostgresFundTransferStore struct {
	db *sql.DB
}

func NewPostgresFundTransferStore(db *sql.DB) *PostgresFundTransferStore {
	return &PostgresFundTransferStore{db: db}
}

type FundTransferStore interface {
	AdminToMD(ft *models.FundTransferModel) error
	AdminToDistributor(ft *models.FundTransferModel) error
	AdminToRetailer(ft *models.FundTransferModel) error
	MDToDistributor(ft *models.FundTransferModel) error
	MDToRetailer(ft *models.FundTransferModel) error
	DistributorToRetailer(ft *models.FundTransferModel) error
	GetFundTransfersByTransfererID(transfererID string, limit, offset int, startDate, endDate *time.Time) ([]models.FundTransferModel, error)
	GetFundTransfersByReceiverID(receiverID string, limit, offset int, startDate, endDate *time.Time) ([]models.FundTransferModel, error)
	GetAllFundTransfers(limit, offset int, startDate, endDate *time.Time) ([]models.FundTransferModel, error)
}

// --- transfer implementations ---

func (fs *PostgresFundTransferStore) AdminToMD(ft *models.FundTransferModel) error {
	return fs.transfer(ft,
		"admins", "admin_id", "admin_wallet_balance",
		"master_distributors", "master_distributor_id", "master_distributor_wallet_balance",
	)
}

func (fs *PostgresFundTransferStore) AdminToDistributor(ft *models.FundTransferModel) error {
	return fs.transfer(ft,
		"admins", "admin_id", "admin_wallet_balance",
		"distributors", "distributor_id", "distributor_wallet_balance",
	)
}

func (fs *PostgresFundTransferStore) AdminToRetailer(ft *models.FundTransferModel) error {
	return fs.transfer(ft,
		"admins", "admin_id", "admin_wallet_balance",
		"retailers", "retailer_id", "retailer_wallet_balance",
	)
}

func (fs *PostgresFundTransferStore) MDToDistributor(ft *models.FundTransferModel) error {
	return fs.transfer(ft,
		"master_distributors", "master_distributor_id", "master_distributor_wallet_balance",
		"distributors", "distributor_id", "distributor_wallet_balance",
	)
}

func (fs *PostgresFundTransferStore) MDToRetailer(ft *models.FundTransferModel) error {
	return fs.transfer(ft,
		"master_distributors", "master_distributor_id", "master_distributor_wallet_balance",
		"retailers", "retailer_id", "retailer_wallet_balance",
	)
}

func (fs *PostgresFundTransferStore) DistributorToRetailer(ft *models.FundTransferModel) error {
	return fs.transfer(ft,
		"distributors", "distributor_id", "distributor_wallet_balance",
		"retailers", "retailer_id", "retailer_wallet_balance",
	)
}

// --- core transfer logic ---

func (fs *PostgresFundTransferStore) transfer(
	ft *models.FundTransferModel,
	senderTable, senderIDCol, senderBalanceCol string,
	receiverTable, receiverIDCol, receiverBalanceCol string,
) error {
	tx, err := fs.db.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()

	// 1. Get sender balance and check sufficient funds
	senderBefore, err := getBalanceTx(tx, senderTable, senderIDCol, senderBalanceCol, ft.FundTransfererID)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return errors.New("sender not found")
		}
		return err
	}
	if senderBefore < ft.Amount {
		return errors.New("insufficient balance")
	}

	// 2. Get receiver balance
	receiverBefore, err := getBalanceTx(tx, receiverTable, receiverIDCol, receiverBalanceCol, ft.FundReceiverID)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return errors.New("receiver not found")
		}
		return err
	}

	senderAfter := senderBefore - ft.Amount
	receiverAfter := receiverBefore + ft.Amount

	// 3. Deduct from sender
	if err = updateBalanceTx(tx, senderTable, senderIDCol, senderBalanceCol, ft.FundTransfererID, senderAfter); err != nil {
		return err
	}

	// 4. Add to receiver
	if err = updateBalanceTx(tx, receiverTable, receiverIDCol, receiverBalanceCol, ft.FundReceiverID, receiverAfter); err != nil {
		return err
	}

	// 5. Insert fund_transfer record
	ft.FundTransferStatus = "SUCCESS"
	if err = insertFundTransferTx(tx, ft); err != nil {
		return err
	}

	refID := fmt.Sprintf("%d", ft.FundTransferID)

	// 6. Wallet transaction for sender (debit)
	debitAmt := ft.Amount
	if err = insertWalletTransactionTx(tx, ft.FundTransfererID, refID, nil, &debitAmt, senderBefore, senderAfter, "FUND_TRANSFER", ft.Remarks); err != nil {
		return err
	}

	// 7. Wallet transaction for receiver (credit)
	creditAmt := ft.Amount
	if err = insertWalletTransactionTx(tx, ft.FundReceiverID, refID, &creditAmt, nil, receiverBefore, receiverAfter, "FUND_TRANSFER", ft.Remarks); err != nil {
		return err
	}

	return tx.Commit()
}

// --- tx helpers ---

func getBalanceTx(tx *sql.Tx, table, idCol, balanceCol, id string) (float64, error) {
	query := fmt.Sprintf(`SELECT %s FROM %s WHERE %s = $1;`, balanceCol, table, idCol)
	var balance float64
	err := tx.QueryRow(query, id).Scan(&balance)
	return balance, err
}

func updateBalanceTx(tx *sql.Tx, table, idCol, balanceCol, id string, newBalance float64) error {
	query := fmt.Sprintf(`UPDATE %s SET %s = $1, updated_at = CURRENT_TIMESTAMP WHERE %s = $2;`, table, balanceCol, idCol)
	_, err := tx.Exec(query, newBalance, id)
	return err
}

func insertFundTransferTx(tx *sql.Tx, ft *models.FundTransferModel) error {
	query := `
	INSERT INTO fund_transfers (
		fund_transferer_id,
		fund_receiver_id,
		amount,
		fund_transfer_status,
		remarks
	) VALUES ($1, $2, $3, $4, $5)
	RETURNING fund_transfer_id, created_at;
	`
	return tx.QueryRow(query,
		ft.FundTransfererID,
		ft.FundReceiverID,
		ft.Amount,
		ft.FundTransferStatus,
		ft.Remarks,
	).Scan(&ft.FundTransferID, &ft.CreatedAT)
}

// fundTransferSelectBase selects all columns including user names for both parties.
const fundTransferSelectBase = `
SELECT
	ft.fund_transfer_id,
	ft.fund_transferer_id,
	ft.fund_receiver_id,
	ft.amount,
	ft.fund_transfer_status,
	ft.remarks,
	ft.created_at,
	wt.before_balance,
	wt.after_balance,
	COALESCE(ta.admin_name, tmd.master_distributor_name, td.distributor_name, tr.retailer_name, '') AS transferer_name,
	COALESCE(tmd.master_distributor_business_name, td.distributor_business_name, tr.retailer_business_name) AS transferer_business_name,
	COALESCE(ra.admin_name, rmd.master_distributor_name, rd.distributor_name, rr.retailer_name, '') AS receiver_name,
	COALESCE(rmd.master_distributor_business_name, rd.distributor_business_name, rr.retailer_business_name) AS receiver_business_name
FROM fund_transfers ft
JOIN wallet_transactions wt
	ON CAST(ft.fund_transfer_id AS TEXT) = wt.reference_id
	AND wt.user_id = ft.fund_transferer_id
LEFT JOIN admins ta               ON ft.fund_transferer_id = ta.admin_id
LEFT JOIN master_distributors tmd ON ft.fund_transferer_id = tmd.master_distributor_id
LEFT JOIN distributors td         ON ft.fund_transferer_id = td.distributor_id
LEFT JOIN retailers tr            ON ft.fund_transferer_id = tr.retailer_id
LEFT JOIN admins ra               ON ft.fund_receiver_id = ra.admin_id
LEFT JOIN master_distributors rmd ON ft.fund_receiver_id = rmd.master_distributor_id
LEFT JOIN distributors rd         ON ft.fund_receiver_id = rd.distributor_id
LEFT JOIN retailers rr            ON ft.fund_receiver_id = rr.retailer_id
`

func (fs *PostgresFundTransferStore) GetFundTransfersByTransfererID(transfererID string, limit, offset int, startDate, endDate *time.Time) ([]models.FundTransferModel, error) {
	query := fundTransferSelectBase + `
	WHERE ft.fund_transferer_id = $1
	AND ft.created_at >= COALESCE($4, '-infinity'::TIMESTAMPTZ)
	AND ft.created_at <= COALESCE($5, 'infinity'::TIMESTAMPTZ)
	ORDER BY ft.created_at DESC
	LIMIT $2 OFFSET $3;
	`
	return scanFundTransfers(fs.db, query, transfererID, limit, offset, startDate, endDate)
}

func (fs *PostgresFundTransferStore) GetFundTransfersByReceiverID(receiverID string, limit, offset int, startDate, endDate *time.Time) ([]models.FundTransferModel, error) {
	query := fundTransferSelectBase + `
	WHERE ft.fund_receiver_id = $1
	AND ft.created_at >= COALESCE($4, '-infinity'::TIMESTAMPTZ)
	AND ft.created_at <= COALESCE($5, 'infinity'::TIMESTAMPTZ)
	ORDER BY ft.created_at DESC
	LIMIT $2 OFFSET $3;
	`
	return scanFundTransfers(fs.db, query, receiverID, limit, offset, startDate, endDate)
}

func (fs *PostgresFundTransferStore) GetAllFundTransfers(limit, offset int, startDate, endDate *time.Time) ([]models.FundTransferModel, error) {
	query := fundTransferSelectBase + `
	WHERE ft.created_at >= COALESCE($3, '-infinity'::TIMESTAMPTZ)
	AND ft.created_at <= COALESCE($4, 'infinity'::TIMESTAMPTZ)
	ORDER BY ft.created_at DESC
	LIMIT $1 OFFSET $2;
	`
	return scanFundTransfers(fs.db, query, limit, offset, startDate, endDate)
}

func scanFundTransfers(db *sql.DB, query string, args ...any) ([]models.FundTransferModel, error) {
	rows, err := db.Query(query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var transfers []models.FundTransferModel
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
			&ft.BeforeBalance,
			&ft.AfterBalance,
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

func insertWalletTransactionTx(tx *sql.Tx, userID, referenceID string, creditAmount, debitAmount *float64, beforeBalance, afterBalance float64, reason, remarks string) error {
	query := `
	INSERT INTO wallet_transactions (
		user_id,
		reference_id,
		credit_amount,
		debit_amount,
		before_balance,
		after_balance,
		transaction_reason,
		remarks
	) VALUES ($1, $2, $3, $4, $5, $6, $7, $8);
	`
	_, err := tx.Exec(query, userID, referenceID, creditAmount, debitAmount, beforeBalance, afterBalance, reason, remarks)
	return err
}
