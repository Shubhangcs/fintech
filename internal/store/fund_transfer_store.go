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
	return &PostgresFundTransferStore{db: db, walletStore: walletStore}
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

	// 1. Atomically debit sender — single UPDATE, checks balance in the WHERE clause
	senderBefore, senderAfter, err := debitTx(tx, senderTable, senderIDCol, senderBalanceCol, ft.FundTransfererID, ft.Amount)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return checkExistsTx(tx, senderTable, senderIDCol, ft.FundTransfererID, "sender")
		}
		return err
	}

	// 2. Atomically credit receiver
	receiverBefore, receiverAfter, err := creditTx(tx, receiverTable, receiverIDCol, receiverBalanceCol, ft.FundReceiverID, ft.Amount)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return errors.New("receiver not found")
		}
		return err
	}

	// 3. Insert fund_transfer record
	ft.FundTransferStatus = "SUCCESS"
	ft.BeforeBalance = senderBefore
	ft.AfterBalance = senderAfter
	if err = insertFundTransferTx(tx, ft); err != nil {
		return err
	}

	refID := fmt.Sprintf("%d", ft.FundTransferID)
	debitAmt := ft.Amount
	creditAmt := ft.Amount

	// 4. Wallet transaction for sender (debit)
	if err = fs.walletStore.CreateWalletTransactionTx(tx, &models.WalletTransactionModel{
		UserID: ft.FundTransfererID, ReferenceID: refID,
		DebitAmount: &debitAmt, BeforeBalance: senderBefore, AfterBalance: senderAfter,
		TransactionReason: "FUND_TRANSFER", Remarks: ft.Remarks,
	}); err != nil {
		return err
	}

	// 5. Wallet transaction for receiver (credit)
	if err = fs.walletStore.CreateWalletTransactionTx(tx, &models.WalletTransactionModel{
		UserID: ft.FundReceiverID, ReferenceID: refID,
		CreditAmount: &creditAmt, BeforeBalance: receiverBefore, AfterBalance: receiverAfter,
		TransactionReason: "FUND_TRANSFER", Remarks: ft.Remarks,
	}); err != nil {
		return err
	}

	return tx.Commit()
}

// debitTx atomically deducts amount from the sender.
// Returns (beforeBalance, afterBalance). Returns sql.ErrNoRows if sender
// doesn't exist OR has insufficient balance (use checkExistsTx to distinguish).
func debitTx(tx *sql.Tx, table, idCol, balanceCol, id string, amount float64) (before, after float64, err error) {
	q := fmt.Sprintf(
		`UPDATE %s SET %s = %s - $1, updated_at = CURRENT_TIMESTAMP
		 WHERE %s = $2 AND %s >= $1
		 RETURNING %s + $1, %s;`,
		table, balanceCol, balanceCol, idCol, balanceCol, balanceCol, balanceCol,
	)
	err = tx.QueryRow(q, amount, id).Scan(&before, &after)
	return
}

// creditTx atomically adds amount to the receiver.
// Returns (beforeBalance, afterBalance). Returns sql.ErrNoRows if receiver doesn't exist.
func creditTx(tx *sql.Tx, table, idCol, balanceCol, id string, amount float64) (before, after float64, err error) {
	q := fmt.Sprintf(
		`UPDATE %s SET %s = %s + $1, updated_at = CURRENT_TIMESTAMP
		 WHERE %s = $2
		 RETURNING %s - $1, %s;`,
		table, balanceCol, balanceCol, idCol, balanceCol, balanceCol,
	)
	err = tx.QueryRow(q, amount, id).Scan(&before, &after)
	return
}

// checkExistsTx distinguishes "not found" from "insufficient balance"
// after a failed debit (both produce sql.ErrNoRows from debitTx).
func checkExistsTx(tx *sql.Tx, table, idCol, id, role string) error {
	q := fmt.Sprintf(`SELECT EXISTS(SELECT 1 FROM %s WHERE %s = $1);`, table, idCol)
	var exists bool
	_ = tx.QueryRow(q, id).Scan(&exists)
	if !exists {
		return fmt.Errorf("%s not found", role)
	}
	return errors.New("insufficient balance")
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

// --- get queries ---

// fundTransferSelectBase uses two LATERAL subqueries instead of 8 LEFT JOINs.
// Each LATERAL does a 4-table UNION ALL and stops at the first match (LIMIT 1),
// hitting only PK indexes. The wallet_transactions join is removed — before/after
// balance is set at transfer time and returned in the create response.
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
