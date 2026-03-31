package store

import (
	"database/sql"
	"errors"
	"fmt"

	"github.com/google/uuid"
	"github.com/levionstudio/fintech/internal/models"
	"github.com/levionstudio/fintech/internal/utils"
)

type PostgresElectricityBillStore struct {
	db          *sql.DB
	walletStore WalletTransactionStore
}

func NewPostgresElectricityBillStore(db *sql.DB, walletStore WalletTransactionStore) *PostgresElectricityBillStore {
	return &PostgresElectricityBillStore{db: db, walletStore: walletStore}
}

type ElectricityBillStore interface {
	InitializeElectricityBill(eb *models.ElectricityBillModel) error
	FinalizeElectricityBill(id int64, operatorTxnID, orderID, status string) error
	RefundElectricityBill(id int64) error
	GetElectricityBillByID(id int64) (*models.ElectricityBillModel, error)
	GetAllElectricityBills(p utils.QueryParams) ([]models.ElectricityBillModel, error)
	GetElectricityBillsByRetailerID(retailerID string, p utils.QueryParams) ([]models.ElectricityBillModel, error)
	GetElectricityBillsByDistributorID(distributorID string, p utils.QueryParams) ([]models.ElectricityBillModel, error)
	GetElectricityBillsByMasterDistributorID(mdID string, p utils.QueryParams) ([]models.ElectricityBillModel, error)
	CreateElectricityOperator(op models.ElectricityOperatorModel) error
	UpdateElectricityOperator(op models.ElectricityOperatorModel) error
	DeleteElectricityOperator(operatorCode int) error
	GetElectricityOperators() ([]models.ElectricityOperatorModel, error)
}

func (es *PostgresElectricityBillStore) InitializeElectricityBill(eb *models.ElectricityBillModel) error {
	rc, err := getRetailerDetails(es.db, eb.RetailerID)
	if err != nil {
		return err
	}
	if !rc.kyc {
		return errors.New("retailer KYC is not verified")
	}
	if rc.blocked {
		return errors.New("retailer is blocked")
	}
	if rc.balance < eb.Amount {
		return errors.New("insufficient wallet balance")
	}

	eb.PartnerRequestID = uuid.New().String()
	eb.Commision = 0
	eb.TransactionStatus = "PENDING"

	tx, err := es.db.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()

	// INSERT first to get ID as wallet entry reference.
	if err = tx.QueryRow(`
		INSERT INTO electricity_bill_payments (
			retailer_id, partner_request_id, customer_id,
			operator_name, operator_code, customer_email,
			amount, commision, operator_transaction_id, order_id, transaction_status
		) VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11)
		RETURNING electricity_bill_transaction_id, created_at
	`,
		eb.RetailerID, eb.PartnerRequestID, eb.CustomerID,
		eb.OperatorName, eb.OperatorCode, eb.CustomerEmail,
		eb.Amount, eb.Commision, "", "", eb.TransactionStatus,
	).Scan(&eb.ElectricityBillTransactionID, &eb.CreatedAt); err != nil {
		return err
	}

	refID := fmt.Sprintf("%d", eb.ElectricityBillTransactionID)
	billRemarks := fmt.Sprintf("Electricity bill payment | %s", eb.CustomerID)

	retailerInfo, err := getUserTableInfo(eb.RetailerID)
	if err != nil {
		return err
	}

	// Debit bill amount from retailer.
	if err = debitTx(tx, transaction{
		UserID: eb.RetailerID, ReferenceID: refID,
		Amount: eb.Amount, Reason: "ELECTRICITY_BILL_PAYMENT", Remarks: billRemarks,
		userTableInfo: *retailerInfo,
	}, es.walletStore); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return checkExistsTx(tx, retailerInfo.TableName, retailerInfo.IDColumnName, eb.RetailerID, "retailer")
		}
		return err
	}

	return tx.Commit()
}

func (es *PostgresElectricityBillStore) FinalizeElectricityBill(id int64, operatorTxnID, orderID, status string) error {
	if !models.IsValidRechargeStatus(status) {
		return errors.New("invalid status")
	}
	res, err := es.db.Exec(`
		UPDATE electricity_bill_payments
		SET transaction_status = $2, operator_transaction_id = $3, order_id = $4
		WHERE electricity_bill_transaction_id = $1 AND transaction_status = 'PENDING'
	`, id, status, operatorTxnID, orderID)
	if err != nil {
		return err
	}
	n, err := res.RowsAffected()
	if err != nil {
		return err
	}
	if n == 0 {
		return errors.New("electricity bill not found or already finalized")
	}
	return nil
}

func (es *PostgresElectricityBillStore) RefundElectricityBill(id int64) error {
	eb, err := es.GetElectricityBillByID(id)
	if err != nil {
		return err
	}
	if eb.TransactionStatus != "FAILED" {
		return errors.New("only FAILED electricity bills can be refunded")
	}

	tx, err := es.db.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()

	// Atomic guard: prevents double-refund.
	res, err := tx.Exec(`
		UPDATE electricity_bill_payments
		SET transaction_status = 'REFUND'
		WHERE electricity_bill_transaction_id = $1 AND transaction_status = 'FAILED'
	`, id)
	if err != nil {
		return err
	}
	n, err := res.RowsAffected()
	if err != nil {
		return err
	}
	if n == 0 {
		return errors.New("electricity bill not found or already refunded")
	}

	refID := fmt.Sprintf("%d", id)
	remarks := fmt.Sprintf("Electricity bill refund | Ref: %s", refID)

	retailerInfo, err := getUserTableInfo(eb.RetailerID)
	if err != nil {
		return err
	}

	// Credit full bill amount back to retailer (no commission to reverse).
	if err = creditTx(tx, transaction{
		UserID: eb.RetailerID, ReferenceID: refID,
		Amount: eb.Amount, Reason: "ELECTRICITY_BILL_PAYMENT_REFUND", Remarks: remarks,
		userTableInfo: *retailerInfo,
	}, es.walletStore); err != nil {
		return err
	}

	return tx.Commit()
}

const electricityBillSelectBase = `
SELECT
	eb.electricity_bill_transaction_id, eb.partner_request_id,
	eb.retailer_id, eb.customer_id, eb.operator_name, eb.operator_code,
	eb.customer_email, eb.amount, eb.commision,
	COALESCE(eb.operator_transaction_id, '') AS operator_transaction_id,
	COALESCE(eb.order_id, '') AS order_id,
	eb.transaction_status, eb.created_at,
	COALESCE(r.retailer_name, '') AS retailer_name,
	r.retailer_business_name,
	COALESCE(wt.before_balance, 0) AS before_balance,
	COALESCE(wt.after_balance, 0) AS after_balance
FROM electricity_bill_payments eb
JOIN retailers r ON eb.retailer_id = r.retailer_id
LEFT JOIN wallet_transactions wt ON wt.reference_id = eb.electricity_bill_transaction_id::TEXT
	AND wt.user_id = eb.retailer_id AND wt.debit_amount IS NOT NULL
	AND wt.transaction_reason = 'ELECTRICITY_BILL_PAYMENT'
`

func (es *PostgresElectricityBillStore) GetElectricityBillByID(id int64) (*models.ElectricityBillModel, error) {
	q := electricityBillSelectBase + `WHERE eb.electricity_bill_transaction_id = $1;`
	results, err := scanElectricityBills(es.db, q, id)
	if err != nil {
		return nil, err
	}
	if len(results) == 0 {
		return nil, errors.New("electricity bill not found")
	}
	return &results[0], nil
}

func (es *PostgresElectricityBillStore) GetAllElectricityBills(p utils.QueryParams) ([]models.ElectricityBillModel, error) {
	q := electricityBillSelectBase + `
	WHERE eb.created_at >= COALESCE($3, '-infinity'::TIMESTAMPTZ)
	AND eb.created_at <= COALESCE($4, 'infinity'::TIMESTAMPTZ)
	AND ($5::TEXT IS NULL OR eb.transaction_status = $5)
	AND ($6::TEXT IS NULL OR (
		COALESCE(eb.operator_transaction_id, '') ILIKE '%'||$6||'%' OR
		COALESCE(eb.order_id, '') ILIKE '%'||$6||'%' OR
		eb.partner_request_id ILIKE '%'||$6||'%' OR
		eb.customer_id ILIKE '%'||$6||'%'
	))
	ORDER BY eb.created_at DESC
	LIMIT $1 OFFSET $2;`
	return scanElectricityBills(es.db, q, p.Limit, p.Offset, p.StartDate, p.EndDate, p.Status, p.Search)
}

func (es *PostgresElectricityBillStore) GetElectricityBillsByRetailerID(retailerID string, p utils.QueryParams) ([]models.ElectricityBillModel, error) {
	q := electricityBillSelectBase + `
	WHERE eb.retailer_id = $1
	AND eb.created_at >= COALESCE($4, '-infinity'::TIMESTAMPTZ)
	AND eb.created_at <= COALESCE($5, 'infinity'::TIMESTAMPTZ)
	AND ($6::TEXT IS NULL OR eb.transaction_status = $6)
	AND ($7::TEXT IS NULL OR (
		COALESCE(eb.operator_transaction_id, '') ILIKE '%'||$7||'%' OR
		COALESCE(eb.order_id, '') ILIKE '%'||$7||'%' OR
		eb.partner_request_id ILIKE '%'||$7||'%' OR
		eb.customer_id ILIKE '%'||$7||'%'
	))
	ORDER BY eb.created_at DESC
	LIMIT $2 OFFSET $3;`
	return scanElectricityBills(es.db, q, retailerID, p.Limit, p.Offset, p.StartDate, p.EndDate, p.Status, p.Search)
}

func (es *PostgresElectricityBillStore) GetElectricityBillsByDistributorID(distributorID string, p utils.QueryParams) ([]models.ElectricityBillModel, error) {
	q := electricityBillSelectBase + `
	JOIN distributors d ON r.distributor_id = d.distributor_id
	WHERE d.distributor_id = $1
	AND eb.created_at >= COALESCE($4, '-infinity'::TIMESTAMPTZ)
	AND eb.created_at <= COALESCE($5, 'infinity'::TIMESTAMPTZ)
	AND ($6::TEXT IS NULL OR eb.transaction_status = $6)
	AND ($7::TEXT IS NULL OR (
		COALESCE(eb.operator_transaction_id, '') ILIKE '%'||$7||'%' OR
		COALESCE(eb.order_id, '') ILIKE '%'||$7||'%' OR
		eb.partner_request_id ILIKE '%'||$7||'%' OR
		eb.customer_id ILIKE '%'||$7||'%'
	))
	ORDER BY eb.created_at DESC
	LIMIT $2 OFFSET $3;`
	return scanElectricityBills(es.db, q, distributorID, p.Limit, p.Offset, p.StartDate, p.EndDate, p.Status, p.Search)
}

func (es *PostgresElectricityBillStore) GetElectricityBillsByMasterDistributorID(mdID string, p utils.QueryParams) ([]models.ElectricityBillModel, error) {
	q := electricityBillSelectBase + `
	JOIN distributors d ON r.distributor_id = d.distributor_id
	JOIN master_distributors md ON d.master_distributor_id = md.master_distributor_id
	WHERE md.master_distributor_id = $1
	AND eb.created_at >= COALESCE($4, '-infinity'::TIMESTAMPTZ)
	AND eb.created_at <= COALESCE($5, 'infinity'::TIMESTAMPTZ)
	AND ($6::TEXT IS NULL OR eb.transaction_status = $6)
	AND ($7::TEXT IS NULL OR (
		COALESCE(eb.operator_transaction_id, '') ILIKE '%'||$7||'%' OR
		COALESCE(eb.order_id, '') ILIKE '%'||$7||'%' OR
		eb.partner_request_id ILIKE '%'||$7||'%' OR
		eb.customer_id ILIKE '%'||$7||'%'
	))
	ORDER BY eb.created_at DESC
	LIMIT $2 OFFSET $3;`
	return scanElectricityBills(es.db, q, mdID, p.Limit, p.Offset, p.StartDate, p.EndDate, p.Status, p.Search)
}

func scanElectricityBills(db *sql.DB, q string, args ...any) ([]models.ElectricityBillModel, error) {
	rows, err := db.Query(q, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var results []models.ElectricityBillModel
	for rows.Next() {
		var eb models.ElectricityBillModel
		if err := rows.Scan(
			&eb.ElectricityBillTransactionID, &eb.PartnerRequestID,
			&eb.RetailerID, &eb.CustomerID, &eb.OperatorName, &eb.OperatorCode,
			&eb.CustomerEmail, &eb.Amount, &eb.Commision,
			&eb.OperatorTransactionID, &eb.OrderID,
			&eb.TransactionStatus, &eb.CreatedAt,
			&eb.RetailerName, &eb.RetailerBusinessName,
			&eb.BeforeBalance, &eb.AfterBalance,
		); err != nil {
			return nil, err
		}
		results = append(results, eb)
	}
	return results, rows.Err()
}

func (es *PostgresElectricityBillStore) CreateElectricityOperator(op models.ElectricityOperatorModel) error {
	_, err := es.db.Exec(
		`INSERT INTO electricity_operators (operator_code, operator_name) VALUES ($1, $2)`,
		op.OperatorCode, op.OperatorName,
	)
	return err
}

func (es *PostgresElectricityBillStore) UpdateElectricityOperator(op models.ElectricityOperatorModel) error {
	res, err := es.db.Exec(
		`UPDATE electricity_operators SET operator_name = $2 WHERE operator_code = $1`,
		op.OperatorCode, op.OperatorName,
	)
	if err != nil {
		return err
	}
	return checkRowsAffected(res)
}

func (es *PostgresElectricityBillStore) DeleteElectricityOperator(operatorCode int) error {
	res, err := es.db.Exec(
		`DELETE FROM electricity_operators WHERE operator_code = $1`, operatorCode,
	)
	if err != nil {
		return err
	}
	return checkRowsAffected(res)
}

func (es *PostgresElectricityBillStore) GetElectricityOperators() ([]models.ElectricityOperatorModel, error) {
	rows, err := es.db.Query(`SELECT operator_code, operator_name FROM electricity_operators ORDER BY operator_name`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var operators []models.ElectricityOperatorModel
	for rows.Next() {
		var o models.ElectricityOperatorModel
		if err := rows.Scan(&o.OperatorCode, &o.OperatorName); err != nil {
			return nil, err
		}
		operators = append(operators, o)
	}
	return operators, rows.Err()
}
