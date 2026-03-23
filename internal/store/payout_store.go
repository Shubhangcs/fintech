package store

import (
	"database/sql"
	"errors"
	"fmt"

	"github.com/google/uuid"
	"github.com/levionstudio/fintech/internal/models"
	"github.com/levionstudio/fintech/internal/utils"
)

type PostgresPayoutTransactionStore struct {
	db                    *sql.DB
	commisionStore        CommisionStore
	walletStore           WalletTransactionStore
	transactionLimitStore TransactionLimitStore
}

func NewPostgresPayoutTransactionStore(db *sql.DB, commisionStore CommisionStore, walletStore WalletTransactionStore, transactionLimitStore TransactionLimitStore) *PostgresPayoutTransactionStore {
	return &PostgresPayoutTransactionStore{
		db:                    db,
		commisionStore:        commisionStore,
		walletStore:           walletStore,
		transactionLimitStore: transactionLimitStore,
	}
}

type PayoutTransactionStore interface {
	InitializePayoutTransaction(pt *models.PayoutTransactionModel) error
	FinalizePayout(payoutTransactionID, orderID, operatorTransactionID, status string) error
	RefundPayout(payoutTransactionID string) error
	GetPayoutTransactionByID(payoutTransactionID string) (*models.PayoutTransactionModel, error)
	GetAllPayoutTransactions(p utils.QueryParams) ([]models.PayoutTransactionModel, error)
	GetPayoutTransactionsByRetailerID(retailerID string, p utils.QueryParams) ([]models.PayoutTransactionModel, error)
	GetPayoutTransactionsByDistributorID(distributorID string, p utils.QueryParams) ([]models.PayoutTransactionModel, error)
	GetPayoutTransactionsByMasterDistributorID(mdID string, p utils.QueryParams) ([]models.PayoutTransactionModel, error)
}

type retailerChain struct {
	distributorID string
	mdID          string
	adminID       string
	kyc           bool
	blocked       bool
	balance       float64
}



func (ps *PostgresPayoutTransactionStore) InitializePayoutTransaction(pt *models.PayoutTransactionModel) error {
	// Get Retailer Details
	rc, err := getRetailerDetails(ps.db, pt.RetailerID)
	if err != nil {
		return err
	}

	if !rc.kyc {
		return errors.New("retailer KYC is not verified")
	}
	if rc.blocked {
		return errors.New("retailer is blocked")
	}

	// Resolve Commision
	commision := ps.resolveCommision(pt.RetailerID, rc.distributorID, rc.mdID, rc.adminID, "PAYOUT", pt.Amount)
	totalDeduction := pt.Amount + commision.TotalCommision

	if rc.balance < totalDeduction {
		return errors.New("insufficient wallet balance")
	}

	retailerTransactionLimit, err := ps.getRetailerTransactionLimit(pt.RetailerID, "PAYOUT")
	if err != nil {
		return err
	}

	if pt.Amount > retailerTransactionLimit {
		return errors.New("transaction limit exceded")
	}

	pt.PartnerRequestID = uuid.New().String()
	pt.AdminCommision = commision.AdminCommision
	pt.MasterDistributorCommision = commision.MasterDistributorCommision
	pt.DistributorCommision = commision.DistributorCommision
	pt.RetailerCommision = commision.RetailerCommision

	tx, err := ps.db.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()

	// Insert payout record first to get the reference ID for wallet entries
	const insertQ = `
	INSERT INTO payout_transactions (
		partner_request_id, operator_transaction_id, retailer_id,
		order_id, mobile_number, bank_name, beneficiary_name,
		account_number, ifsc_code, amount, transfer_type,
		admin_commision, master_distributor_commision,
		distributor_commision, retailer_commision,
		payout_transaction_status
	) VALUES (
		$1, '', $2,
		'', $3, $4, $5,
		$6, $7, $8, $9,
		$10, $11,
		$12, $13,
		'PENDING'
	)
	RETURNING payout_transaction_id, payout_transaction_status, created_at, updated_at;
	`
	if err = tx.QueryRow(insertQ,
		pt.PartnerRequestID, pt.RetailerID,
		pt.MobileNumber, pt.BankName, pt.BeneficiaryName,
		pt.AccountNumber, pt.IFSCCode, pt.Amount, pt.TransferType,
		pt.AdminCommision, pt.MasterDistributorCommision,
		pt.DistributorCommision, pt.RetailerCommision,
	).Scan(&pt.PayoutTransactionID, &pt.PayoutTransactionStatus, &pt.CreatedAT, &pt.UpdatedAT); err != nil {
		return err
	}

	refID := pt.PayoutTransactionID
	payoutRemarks := fmt.Sprintf("Payout to %s | Account: %s | Amount: %.2f", pt.BeneficiaryName, pt.AccountNumber, pt.Amount)
	commisionRemarks := fmt.Sprintf("Payout commission | Ref: %s", refID)

	retailerInfo, err := getUserTableInfo(pt.RetailerID)
	if err != nil {
		return err
	}

	// Debit retailer — atomically checks balance, also creates wallet transaction entry
	if err = debitTx(tx, transaction{
		UserID: pt.RetailerID, ReferenceID: refID,
		Amount: totalDeduction, Reason: "PAYOUT", Remarks: payoutRemarks,
		userTableInfo: *retailerInfo,
	}, ps.walletStore); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return checkExistsTx(tx, retailerInfo.TableName, retailerInfo.IDColumnName, pt.RetailerID, "retailer")
		}
		return err
	}

	// Credit commission wallets — skip zero amounts to avoid empty wallet entries
	creditCommision := func(userID string, amount float64) error {
		if amount <= 0 {
			return nil
		}
		info, err := getUserTableInfo(userID)
		if err != nil {
			return err
		}
		return creditTx(tx, transaction{
			UserID: userID, ReferenceID: refID,
			Amount: amount, Reason: "PAYOUT", Remarks: commisionRemarks,
			userTableInfo: *info,
		}, ps.walletStore)
	}

	if err = creditCommision(rc.adminID, pt.AdminCommision); err != nil {
		return err
	}
	if err = creditCommision(rc.mdID, pt.MasterDistributorCommision); err != nil {
		return err
	}
	if err = creditCommision(rc.distributorID, pt.DistributorCommision); err != nil {
		return err
	}
	if err = creditCommision(pt.RetailerID, pt.RetailerCommision); err != nil {
		return err
	}

	return tx.Commit()
}

func (ps *PostgresPayoutTransactionStore) GetPayoutTransactionByID(payoutTransactionID string) (*models.PayoutTransactionModel, error) {
	q := payoutSelectBase + `WHERE pt.payout_transaction_id = $1::UUID;`
	results, err := scanPayoutTransactions(ps.db, q, payoutTransactionID)
	if err != nil {
		return nil, err
	}
	if len(results) == 0 {
		return nil, errors.New("payout transaction not found")
	}
	return &results[0], nil
}

func (ps *PostgresPayoutTransactionStore) FinalizePayout(payoutTransactionID, orderID, operatorTransactionID, status string) error {
	if !models.IsValidPayoutStatus(status) {
		return errors.New("invalid payout_transaction_status")
	}

	res, err := ps.db.Exec(`
		UPDATE payout_transactions
		SET order_id                  = $2,
		    operator_transaction_id   = $3,
		    payout_transaction_status = $4,
		    updated_at                = CURRENT_TIMESTAMP
		WHERE payout_transaction_id = $1 AND payout_transaction_status = 'PENDING'
	`, payoutTransactionID, orderID, operatorTransactionID, status)
	if err != nil {
		return err
	}
	n, err := res.RowsAffected()
	if err != nil {
		return err
	}
	if n == 0 {
		return errors.New("payout transaction not found or already finalized")
	}
	return nil
}

// RefundPayout reverses a FAILED payout: deducts commissions from each recipient
// and credits the full amount (payout + all commissions) back to the retailer.
// Uses AND payout_transaction_status = 'FAILED' guard to prevent double-refund.
func (ps *PostgresPayoutTransactionStore) RefundPayout(payoutTransactionID string) error {
	pt, err := ps.GetPayoutTransactionByID(payoutTransactionID)
	if err != nil {
		return err
	}

	if pt.PayoutTransactionStatus != "FAILED" {
		return errors.New("only FAILED payout transactions can be refunded")
	}

	rc, err := getRetailerDetails(ps.db, pt.RetailerID)
	if err != nil {
		return err
	}

	tx, err := ps.db.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()

	// Atomically mark as REFUND — prevents concurrent double-refund
	res, err := tx.Exec(`
		UPDATE payout_transactions
		SET payout_transaction_status = 'REFUND', updated_at = CURRENT_TIMESTAMP
		WHERE payout_transaction_id = $1 AND payout_transaction_status = 'FAILED'
	`, payoutTransactionID)
	if err != nil {
		return err
	}
	n, err := res.RowsAffected()
	if err != nil {
		return err
	}
	if n == 0 {
		return errors.New("payout transaction not found or already refunded")
	}

	refID := payoutTransactionID
	commisionRemarks := fmt.Sprintf("Payout refund commission recovery | Ref: %s", refID)
	retailerRemarks := fmt.Sprintf("Payout refund | Ref: %s", refID)

	// Debit commissions back from each recipient
	debitCommision := func(userID string, amount float64) error {
		if amount <= 0 {
			return nil
		}
		info, err := getUserTableInfo(userID)
		if err != nil {
			return err
		}
		return debitTx(tx, transaction{
			UserID: userID, ReferenceID: refID,
			Amount: amount, Reason: "PAYOUT_REFUND", Remarks: commisionRemarks,
			userTableInfo: *info,
		}, ps.walletStore)
	}

	if err = debitCommision(rc.adminID, pt.AdminCommision); err != nil {
		return err
	}
	if err = debitCommision(rc.mdID, pt.MasterDistributorCommision); err != nil {
		return err
	}
	if err = debitCommision(rc.distributorID, pt.DistributorCommision); err != nil {
		return err
	}
	if err = debitCommision(pt.RetailerID, pt.RetailerCommision); err != nil {
		return err
	}

	// Credit full amount + all commissions back to retailer
	totalRefund := pt.Amount + pt.AdminCommision + pt.MasterDistributorCommision + pt.DistributorCommision + pt.RetailerCommision
	retailerInfo, err := getUserTableInfo(pt.RetailerID)
	if err != nil {
		return err
	}
	if err = creditTx(tx, transaction{
		UserID: pt.RetailerID, ReferenceID: refID,
		Amount: totalRefund, Reason: "PAYOUT_REFUND", Remarks: retailerRemarks,
		userTableInfo: *retailerInfo,
	}, ps.walletStore); err != nil {
		return err
	}

	return tx.Commit()
}

const payoutSelectBase = `
SELECT
	pt.payout_transaction_id, pt.partner_request_id, pt.operator_transaction_id,
	pt.retailer_id, pt.order_id, pt.mobile_number, pt.bank_name, pt.beneficiary_name,
	pt.account_number, pt.ifsc_code, pt.amount, pt.transfer_type,
	pt.admin_commision, pt.master_distributor_commision,
	pt.distributor_commision, pt.retailer_commision,
	pt.payout_transaction_status, pt.created_at, pt.updated_at,
	COALESCE(r.retailer_name, '')    AS retailer_name,
	r.retailer_business_name,
	COALESCE(wt.before_balance, 0)  AS before_balance,
	COALESCE(wt.after_balance, 0)   AS after_balance
FROM payout_transactions pt
JOIN retailers r ON pt.retailer_id = r.retailer_id
LEFT JOIN wallet_transactions wt ON wt.reference_id = pt.payout_transaction_id::TEXT
	AND wt.user_id = pt.retailer_id
	AND wt.debit_amount IS NOT NULL
`

func (ps *PostgresPayoutTransactionStore) GetAllPayoutTransactions(p utils.QueryParams) ([]models.PayoutTransactionModel, error) {
	q := payoutSelectBase + `
	WHERE pt.created_at >= COALESCE($3, '-infinity'::TIMESTAMPTZ)
	AND pt.created_at <= COALESCE($4, 'infinity'::TIMESTAMPTZ)
	ORDER BY pt.created_at DESC
	LIMIT $1 OFFSET $2;
	`
	return scanPayoutTransactions(ps.db, q, p.Limit, p.Offset, p.StartDate, p.EndDate)
}

func (ps *PostgresPayoutTransactionStore) GetPayoutTransactionsByRetailerID(retailerID string, p utils.QueryParams) ([]models.PayoutTransactionModel, error) {
	q := payoutSelectBase + `
	WHERE pt.retailer_id = $1
	AND pt.created_at >= COALESCE($4, '-infinity'::TIMESTAMPTZ)
	AND pt.created_at <= COALESCE($5, 'infinity'::TIMESTAMPTZ)
	ORDER BY pt.created_at DESC
	LIMIT $2 OFFSET $3;
	`
	return scanPayoutTransactions(ps.db, q, retailerID, p.Limit, p.Offset, p.StartDate, p.EndDate)
}

func (ps *PostgresPayoutTransactionStore) GetPayoutTransactionsByDistributorID(distributorID string, p utils.QueryParams) ([]models.PayoutTransactionModel, error) {
	q := payoutSelectBase + `
	WHERE r.distributor_id = $1
	AND pt.created_at >= COALESCE($4, '-infinity'::TIMESTAMPTZ)
	AND pt.created_at <= COALESCE($5, 'infinity'::TIMESTAMPTZ)
	ORDER BY pt.created_at DESC
	LIMIT $2 OFFSET $3;
	`
	return scanPayoutTransactions(ps.db, q, distributorID, p.Limit, p.Offset, p.StartDate, p.EndDate)
}

func (ps *PostgresPayoutTransactionStore) GetPayoutTransactionsByMasterDistributorID(mdID string, p utils.QueryParams) ([]models.PayoutTransactionModel, error) {
	q := payoutSelectBase + `
	JOIN distributors d ON r.distributor_id = d.distributor_id
	WHERE d.master_distributor_id = $1
	AND pt.created_at >= COALESCE($4, '-infinity'::TIMESTAMPTZ)
	AND pt.created_at <= COALESCE($5, 'infinity'::TIMESTAMPTZ)
	ORDER BY pt.created_at DESC
	LIMIT $2 OFFSET $3;
	`
	return scanPayoutTransactions(ps.db, q, mdID, p.Limit, p.Offset, p.StartDate, p.EndDate)
}

func scanPayoutTransactions(db *sql.DB, query string, args ...any) ([]models.PayoutTransactionModel, error) {
	rows, err := db.Query(query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	results := []models.PayoutTransactionModel{}
	for rows.Next() {
		var pt models.PayoutTransactionModel
		if err = rows.Scan(
			&pt.PayoutTransactionID, &pt.PartnerRequestID, &pt.OperatorTransactionID,
			&pt.RetailerID, &pt.OrderID, &pt.MobileNumber, &pt.BankName, &pt.BeneficiaryName,
			&pt.AccountNumber, &pt.IFSCCode, &pt.Amount, &pt.TransferType,
			&pt.AdminCommision, &pt.MasterDistributorCommision,
			&pt.DistributorCommision, &pt.RetailerCommision,
			&pt.PayoutTransactionStatus, &pt.CreatedAT, &pt.UpdatedAT,
			&pt.RetailerName, &pt.RetailerBusinessName,
			&pt.BeforeBalance, &pt.AfterBalance,
		); err != nil {
			return nil, err
		}
		results = append(results, pt)
	}
	return results, rows.Err()
}
