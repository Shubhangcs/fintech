package store

import (
	"database/sql"
	"errors"
	"fmt"

	"github.com/levionstudio/fintech/internal/models"
)

type PostgresPayoutStore struct {
	db *sql.DB
}

func NewPostgresPayoutStore(db *sql.DB) *PostgresPayoutStore {
	return &PostgresPayoutStore{
		db: db,
	}
}

type PayoutStore interface {
	GetPayoutCommision(retailerID string, amount float64) (*models.PayoutCommision, error)
	InitiatePayoutTransaction(req *models.CreatePayoutRequest, partnerRequestID string, commission *models.PayoutCommision) (string, error)
	FinalizePayoutTransaction(transactionID, orderID, operatorTxID, status string, commission *models.PayoutCommision, retailerID string) error
	FailPayoutTransaction(transactionID, orderID, operatorTxID string) error
	GetAllPayoutTransactions(limit, offset int) ([]models.PayoutTransactionModel, error)
	GetPayoutTransactionsByRetailerID(retailerID string, limit, offset int) ([]models.PayoutTransactionModel, error)
	PayoutRefund(transactionID string) error
	UpdatePayoutTransaction(transactionID string, req *models.UpdatePayoutTransactionRequest) error
}

func (ps *PostgresPayoutStore) GetPayoutCommision(retailerID string, amount float64) (*models.PayoutCommision, error) {

	var (
		adminId             string
		masterDistributorId string
		distributorId       string
	)

	hierarchyQuery := `
		SELECT a.admin_id, md.master_distributor_id, d.distributor_id
		FROM retailers r
		JOIN distributors d ON d.distributor_id = r.distributor_id
		JOIN master_distributors md ON md.master_distributor_id = d.master_distributor_id
		JOIN admins a ON a.admin_id = md.admin_id
		WHERE r.retailer_id = $1;
	`

	if err := ps.db.QueryRow(hierarchyQuery, retailerID).Scan(&adminId, &masterDistributorId, &distributorId); err != nil {
		return nil, err
	}

	getCommision := func(userId string) (*models.PayoutCommision, error) {
		query := `
			SELECT
				total_commision,
				admin_commision,
				master_distributor_commision,
				distributor_commision,
				retailer_commision
			FROM commisions
			WHERE user_id=$1 AND service='PAYOUT'
			LIMIT 1;
		`

		var c models.PayoutCommision
		err := ps.db.QueryRow(query, userId).Scan(
			&c.Total,
			&c.Admin,
			&c.MasterDistributor,
			&c.Distributor,
			&c.Retailer,
		)

		if err != nil && errors.Is(err, sql.ErrNoRows) {
			return nil, nil
		}
		if err != nil {
			return nil, err
		}
		return &c, nil
	}

	var commision *models.PayoutCommision

	ids := []string{
		retailerID,
		distributorId,
		masterDistributorId,
	}

	for _, id := range ids {
		c, err := getCommision(id)
		if err != nil {
			return nil, err
		}
		if c != nil {
			commision = c
			break
		}
	}

	// Default commision if nothing found
	if commision == nil {
		commision = &models.PayoutCommision{
			Total:             1.2,
			Retailer:          0.5,
			Distributor:       0.2,
			MasterDistributor: 0.05,
			Admin:             0.25,
		}
	}

	// Final calculation (percentage → amount)
	totalAmount := (commision.Total / 100) * amount

	return &models.PayoutCommision{
		Total:             totalAmount,
		Retailer:          totalAmount * commision.Retailer,
		Distributor:       totalAmount * commision.Distributor,
		MasterDistributor: totalAmount * commision.MasterDistributor,
		Admin:             totalAmount * commision.Admin,
	}, nil
}

// InitiatePayoutTransaction is Phase 1: validates the retailer, debits the wallet,
// and writes a PENDING payout record — all in one atomic transaction.
// The external API must be called AFTER this returns successfully.
func (ps *PostgresPayoutStore) InitiatePayoutTransaction(req *models.CreatePayoutRequest, partnerRequestID string, commission *models.PayoutCommision) (string, error) {
	tx, err := ps.db.Begin()
	if err != nil {
		return "", err
	}
	defer tx.Rollback()

	// 1. Check retailer status and get current balance
	var balance float64
	var kycStatus string
	var isBlocked bool
	err = tx.QueryRow(`
		SELECT retailer_wallet_balance, retailer_kyc_status, is_retailer_blocked
		FROM retailers WHERE retailer_id = $1
	`, req.RetailerID).Scan(&balance, &kycStatus, &isBlocked)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return "", errors.New("retailer not found")
		}
		return "", err
	}
	if isBlocked {
		return "", errors.New("retailer is blocked")
	}
	if kycStatus != "APPROVED" {
		return "", errors.New("retailer KYC is not approved")
	}

	// 2. Check per-transaction limit (optional — if no record, no limit applies)
	var limitAmount float64
	err = tx.QueryRow(`
		SELECT limit_amount FROM transaction_limit
		WHERE retailer_id = $1 AND service = 'PAYOUT'
	`, req.RetailerID).Scan(&limitAmount)
	if err != nil && !errors.Is(err, sql.ErrNoRows) {
		return "", err
	}
	if err == nil && req.Amount > limitAmount {
		return "", fmt.Errorf("amount exceeds transaction limit of %.2f", limitAmount)
	}

	// 3. Debit retailer: amount + total commission
	totalDebit := req.Amount + commission.Total
	if balance < totalDebit {
		return "", errors.New("insufficient balance")
	}
	newBalance := balance - totalDebit
	_, err = tx.Exec(`
		UPDATE retailers SET retailer_wallet_balance = $1, updated_at = CURRENT_TIMESTAMP
		WHERE retailer_id = $2
	`, newBalance, req.RetailerID)
	if err != nil {
		return "", err
	}

	// 4. Insert PENDING payout record
	transferTypeStr := "IMPS"
	if req.TransferType == 6 {
		transferTypeStr = "NEFT"
	}
	var transactionID string
	err = tx.QueryRow(`
		INSERT INTO payout_transactions (
			payout_transaction_id, partner_request_id, operator_transaction_id, order_id,
			retailer_id, mobile_number, bank_name, beneficiary_name, account_number, ifsc_code,
			amount, transfer_type,
			admin_commision, master_distributor_commision, distributor_commision, retailer_commision,
			before_balance, after_balance, payout_transaction_status
		) VALUES (
			gen_random_uuid(), $1, '', '',
			$2, $3, $4, $5, $6, $7,
			$8, $9,
			$10, $11, $12, $13,
			$14, $15, 'PENDING'
		) RETURNING payout_transaction_id
	`,
		partnerRequestID,
		req.RetailerID, req.MobileNumber, req.BankName, req.BeneficiaryName, req.AccountNumber, req.IFSCCode,
		req.Amount, transferTypeStr,
		commission.Admin, commission.MasterDistributor, commission.Distributor, commission.Retailer,
		balance, newBalance,
	).Scan(&transactionID)
	if err != nil {
		return "", err
	}

	// 5. Wallet transaction for retailer debit
	if err = insertWalletTransactionTx(tx, req.RetailerID, transactionID, nil, &totalDebit, balance, newBalance, "PAYOUT", ""); err != nil {
		return "", err
	}

	return transactionID, tx.Commit()
}

// FinalizePayoutTransaction is Phase 2 on success/pending:
// credits commissions to the hierarchy and updates the payout record status.
func (ps *PostgresPayoutStore) FinalizePayoutTransaction(transactionID, orderID, operatorTxID, status string, commission *models.PayoutCommision, retailerID string) error {
	tx, err := ps.db.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()

	// Resolve hierarchy
	var distributorID, masterDistributorID, adminID string
	err = tx.QueryRow(`
		SELECT r.distributor_id, d.master_distributor_id, md.admin_id
		FROM retailers r
		JOIN distributors d ON r.distributor_id = d.distributor_id
		JOIN master_distributors md ON d.master_distributor_id = md.master_distributor_id
		WHERE r.retailer_id = $1
	`, retailerID).Scan(&distributorID, &masterDistributorID, &adminID)
	if err != nil {
		return err
	}

	type commisionEntry struct {
		userID string
		amount float64
	}
	entries := []commisionEntry{
		{adminID, commission.Admin},
		{masterDistributorID, commission.MasterDistributor},
		{distributorID, commission.Distributor},
		{retailerID, commission.Retailer},
	}

	for _, entry := range entries {
		if entry.amount <= 0 {
			continue
		}
		wi, err := walletInfoFromID(entry.userID)
		if err != nil {
			return err
		}
		var before float64
		err = tx.QueryRow(
			fmt.Sprintf(`SELECT %s FROM %s WHERE %s = $1`, wi.balanceCol, wi.table, wi.idCol),
			entry.userID,
		).Scan(&before)
		if err != nil {
			return err
		}
		after := before + entry.amount
		_, err = tx.Exec(
			fmt.Sprintf(`UPDATE %s SET %s = $1, updated_at = CURRENT_TIMESTAMP WHERE %s = $2`, wi.table, wi.balanceCol, wi.idCol),
			after, entry.userID,
		)
		if err != nil {
			return err
		}
		credit := entry.amount
		if err = insertWalletTransactionTx(tx, entry.userID, transactionID, &credit, nil, before, after, "PAYOUT_COMMISSION", ""); err != nil {
			return err
		}
	}

	_, err = tx.Exec(`
		UPDATE payout_transactions
		SET operator_transaction_id = $1, order_id = $2, payout_transaction_status = $3,
		    updated_at = CURRENT_TIMESTAMP
		WHERE payout_transaction_id = $4
	`, operatorTxID, orderID, status, transactionID)
	if err != nil {
		return err
	}

	return tx.Commit()
}

// FailPayoutTransaction is Phase 2 on failure:
// refunds the full debit (amount + commission) back to the retailer and marks the record FAILED.
func (ps *PostgresPayoutStore) FailPayoutTransaction(transactionID, orderID, operatorTxID string) error {
	tx, err := ps.db.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()

	var retailerID string
	var amount, adminComm, mdComm, distComm, retailerComm float64
	err = tx.QueryRow(`
		SELECT retailer_id, amount, admin_commision, master_distributor_commision,
		       distributor_commision, retailer_commision
		FROM payout_transactions
		WHERE payout_transaction_id = $1
	`, transactionID).Scan(&retailerID, &amount, &adminComm, &mdComm, &distComm, &retailerComm)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return errors.New("transaction not found")
		}
		return err
	}

	totalRefund := amount + adminComm + mdComm + distComm + retailerComm

	var retailerBefore float64
	err = tx.QueryRow(`SELECT retailer_wallet_balance FROM retailers WHERE retailer_id = $1`, retailerID).Scan(&retailerBefore)
	if err != nil {
		return err
	}
	retailerAfter := retailerBefore + totalRefund
	_, err = tx.Exec(`
		UPDATE retailers SET retailer_wallet_balance = $1, updated_at = CURRENT_TIMESTAMP
		WHERE retailer_id = $2
	`, retailerAfter, retailerID)
	if err != nil {
		return err
	}
	if err = insertWalletTransactionTx(tx, retailerID, transactionID, &totalRefund, nil, retailerBefore, retailerAfter, "PAYOUT_REFUND", ""); err != nil {
		return err
	}

	_, err = tx.Exec(`
		UPDATE payout_transactions
		SET operator_transaction_id = $1, order_id = $2, payout_transaction_status = 'FAILED',
		    updated_at = CURRENT_TIMESTAMP
		WHERE payout_transaction_id = $3
	`, operatorTxID, orderID, transactionID)
	if err != nil {
		return err
	}

	return tx.Commit()
}

// PayoutRefund manually reverses a SUCCESS transaction:
// claws back commissions from the hierarchy and refunds the retailer.
func (ps *PostgresPayoutStore) PayoutRefund(transactionID string) error {
	tx, err := ps.db.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()

	var retailerID string
	var amount, adminComm, mdComm, distComm, retailerComm float64
	var status string
	err = tx.QueryRow(`
		SELECT retailer_id, amount, admin_commision, master_distributor_commision,
		       distributor_commision, retailer_commision, payout_transaction_status
		FROM payout_transactions
		WHERE payout_transaction_id = $1
	`, transactionID).Scan(&retailerID, &amount, &adminComm, &mdComm, &distComm, &retailerComm, &status)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return errors.New("transaction not found")
		}
		return err
	}
	if status != "SUCCESS" {
		return errors.New("only successful transactions can be refunded")
	}

	// Resolve hierarchy
	var distributorID, masterDistributorID, adminID string
	err = tx.QueryRow(`
		SELECT r.distributor_id, d.master_distributor_id, md.admin_id
		FROM retailers r
		JOIN distributors d ON r.distributor_id = d.distributor_id
		JOIN master_distributors md ON d.master_distributor_id = md.master_distributor_id
		WHERE r.retailer_id = $1
	`, retailerID).Scan(&distributorID, &masterDistributorID, &adminID)
	if err != nil {
		return err
	}

	// Claw back commissions from hierarchy
	type commisionEntry struct {
		userID string
		amount float64
	}
	for _, entry := range []commisionEntry{
		{adminID, adminComm},
		{masterDistributorID, mdComm},
		{distributorID, distComm},
		{retailerID, retailerComm},
	} {
		if entry.amount <= 0 {
			continue
		}
		wi, err := walletInfoFromID(entry.userID)
		if err != nil {
			return err
		}
		var before float64
		err = tx.QueryRow(
			fmt.Sprintf(`SELECT %s FROM %s WHERE %s = $1`, wi.balanceCol, wi.table, wi.idCol),
			entry.userID,
		).Scan(&before)
		if err != nil {
			return err
		}
		after := before - entry.amount
		_, err = tx.Exec(
			fmt.Sprintf(`UPDATE %s SET %s = $1, updated_at = CURRENT_TIMESTAMP WHERE %s = $2`, wi.table, wi.balanceCol, wi.idCol),
			after, entry.userID,
		)
		if err != nil {
			return err
		}
		debit := entry.amount
		if err = insertWalletTransactionTx(tx, entry.userID, transactionID, nil, &debit, before, after, "PAYOUT_COMMISSION_REVERSAL", ""); err != nil {
			return err
		}
	}

	// Refund full amount + all commissions to retailer
	var retailerBefore float64
	err = tx.QueryRow(`SELECT retailer_wallet_balance FROM retailers WHERE retailer_id = $1`, retailerID).Scan(&retailerBefore)
	if err != nil {
		return err
	}
	totalRefund := amount + adminComm + mdComm + distComm + retailerComm
	retailerAfter := retailerBefore + totalRefund
	_, err = tx.Exec(`
		UPDATE retailers SET retailer_wallet_balance = $1, updated_at = CURRENT_TIMESTAMP
		WHERE retailer_id = $2
	`, retailerAfter, retailerID)
	if err != nil {
		return err
	}
	if err = insertWalletTransactionTx(tx, retailerID, transactionID, &totalRefund, nil, retailerBefore, retailerAfter, "PAYOUT_REFUND", ""); err != nil {
		return err
	}

	_, err = tx.Exec(`
		UPDATE payout_transactions SET payout_transaction_status = 'REFUNDED', updated_at = CURRENT_TIMESTAMP
		WHERE payout_transaction_id = $1
	`, transactionID)
	if err != nil {
		return err
	}

	return tx.Commit()
}

const payoutSelectBase = `
SELECT
	pt.payout_transaction_id,
	pt.partner_request_id,
	pt.operator_transaction_id,
	pt.order_id,
	pt.retailer_id,
	COALESCE(r.retailer_name, '') AS retailer_name,
	r.retailer_business_name,
	pt.mobile_number,
	pt.bank_name,
	pt.beneficiary_name,
	pt.account_number,
	pt.ifsc_code,
	pt.amount,
	pt.transfer_type,
	pt.admin_commision,
	pt.master_distributor_commision,
	pt.distributor_commision,
	pt.retailer_commision,
	pt.before_balance,
	pt.after_balance,
	pt.payout_transaction_status,
	pt.created_at,
	pt.updated_at
FROM payout_transactions pt
LEFT JOIN retailers r ON pt.retailer_id = r.retailer_id
`

func (ps *PostgresPayoutStore) GetAllPayoutTransactions(limit, offset int) ([]models.PayoutTransactionModel, error) {
	query := payoutSelectBase + `ORDER BY pt.created_at DESC LIMIT $1 OFFSET $2;`
	return scanPayoutTransactions(ps.db, query, limit, offset)
}

func (ps *PostgresPayoutStore) GetPayoutTransactionsByRetailerID(retailerID string, limit, offset int) ([]models.PayoutTransactionModel, error) {
	query := payoutSelectBase + `WHERE pt.retailer_id = $1 ORDER BY pt.created_at DESC LIMIT $2 OFFSET $3;`
	return scanPayoutTransactions(ps.db, query, retailerID, limit, offset)
}

func (ps *PostgresPayoutStore) UpdatePayoutTransaction(transactionID string, req *models.UpdatePayoutTransactionRequest) error {
	res, err := ps.db.Exec(`
		UPDATE payout_transactions
		SET operator_transaction_id    = COALESCE(NULLIF($1, ''), operator_transaction_id),
		    order_id                   = COALESCE(NULLIF($2, ''), order_id),
		    payout_transaction_status  = COALESCE(NULLIF($3, ''), payout_transaction_status),
		    updated_at                 = CURRENT_TIMESTAMP
		WHERE payout_transaction_id = $4
	`, req.OperatorTransactionID, req.OrderID, req.Status, transactionID)
	if err != nil {
		return err
	}
	return checkRowsAffected(res)
}

func scanPayoutTransactions(db *sql.DB, query string, args ...any) ([]models.PayoutTransactionModel, error) {
	rows, err := db.Query(query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var transactions []models.PayoutTransactionModel
	for rows.Next() {
		var pt models.PayoutTransactionModel
		if err = rows.Scan(
			&pt.PayoutTransactionID,
			&pt.PartnerRequestID,
			&pt.OperatorTransactionID,
			&pt.OrderID,
			&pt.RetailerID,
			&pt.RetailerName,
			&pt.RetailerBusinessName,
			&pt.MobileNumber,
			&pt.BankName,
			&pt.BeneficiaryName,
			&pt.AccountNumber,
			&pt.IFSCCode,
			&pt.Amount,
			&pt.TransferType,
			&pt.AdminCommision,
			&pt.MasterDistributorCommision,
			&pt.DistributorCommision,
			&pt.RetailerCommision,
			&pt.BeforeBalance,
			&pt.AfterBalance,
			&pt.TransactionStatus,
			&pt.CreatedAT,
			&pt.UpdatedAT,
		); err != nil {
			return nil, err
		}
		transactions = append(transactions, pt)
	}

	return transactions, rows.Err()
}
