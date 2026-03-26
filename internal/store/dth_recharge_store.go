package store

import (
	"database/sql"
	"errors"
	"fmt"

	"github.com/google/uuid"
	"github.com/levionstudio/fintech/internal/models"
	"github.com/levionstudio/fintech/internal/utils"
)

const dthRechargeCommision = 1.0

type PostgresDTHRechargeStore struct {
	db          *sql.DB
	walletStore WalletTransactionStore
}

func NewPostgresDTHRechargeStore(db *sql.DB, walletStore WalletTransactionStore) *PostgresDTHRechargeStore {
	return &PostgresDTHRechargeStore{db: db, walletStore: walletStore}
}

type DTHRechargeStore interface {
	InitializeDTHRecharge(dr *models.DTHRechargeModel) error
	FinalizeDTHRecharge(id int64, operatorTxnID, orderID, status string) error
	RefundDTHRecharge(id int64) error
	GetDTHRechargeByID(id int64) (*models.DTHRechargeModel, error)
	GetAllDTHRecharge(p utils.QueryParams) ([]models.DTHRechargeModel, error)
	GetDTHRechargeByRetailerID(retailerID string, p utils.QueryParams) ([]models.DTHRechargeModel, error)
	GetDTHRechargeByDistributorID(distributorID string, p utils.QueryParams) ([]models.DTHRechargeModel, error)
	GetDTHRechargeByMasterDistributorID(mdID string, p utils.QueryParams) ([]models.DTHRechargeModel, error)
	CreateDTHRechargeOperator(op models.DTHRechargeOperatorModel) error
	UpdateDTHRechargeOperator(op models.DTHRechargeOperatorModel) error
	DeleteDTHRechargeOperator(operatorCode int) error
	GetDTHRechargeOperators() ([]models.DTHRechargeOperatorModel, error)
}

func (ds *PostgresDTHRechargeStore) InitializeDTHRecharge(dr *models.DTHRechargeModel) error {
	rc, err := getRetailerDetails(ds.db, dr.RetailerID)
	if err != nil {
		return err
	}
	if !rc.kyc {
		return errors.New("retailer KYC is not verified")
	}
	if rc.blocked {
		return errors.New("retailer is blocked")
	}
	if rc.balance < dr.Amount {
		return errors.New("insufficient wallet balance")
	}

	commision := 0.0
	if dr.Amount > 100 {
		commision = dthRechargeCommision
	}

	dr.PartnerRequestID = uuid.New().String()
	dr.Commision = commision
	dr.Status = "PENDING"

	tx, err := ds.db.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()

	// INSERT first to get ID as wallet entry reference.
	if err = tx.QueryRow(`
		INSERT INTO dth_recharge (
			retailer_id, partner_request_id, customer_id,
			operator_name, operator_code, amount, commision,
			operator_transaction_id, order_id, status
		) VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10)
		RETURNING dth_transaction_id, created_at
	`,
		dr.RetailerID, dr.PartnerRequestID, dr.CustomerID,
		dr.OperatorName, dr.OperatorCode, dr.Amount, dr.Commision,
		"", "", dr.Status,
	).Scan(&dr.DTHTransactionID, &dr.CreatedAt); err != nil {
		return err
	}

	refID := fmt.Sprintf("%d", dr.DTHTransactionID)
	rechargeRemarks := fmt.Sprintf("DTH recharge | %s", dr.CustomerID)

	retailerInfo, err := getUserTableInfo(dr.RetailerID)
	if err != nil {
		return err
	}

	// Debit recharge amount from retailer.
	if err = debitTx(tx, transaction{
		UserID: dr.RetailerID, ReferenceID: refID,
		Amount: dr.Amount, Reason: "DTH_RECHARGE", Remarks: rechargeRemarks,
		userTableInfo: *retailerInfo,
	}, ds.walletStore); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return checkExistsTx(tx, retailerInfo.TableName, retailerInfo.IDColumnName, dr.RetailerID, "retailer")
		}
		return err
	}

	// If amount > ₹100: admin pays ₹1 commission to retailer.
	if commision > 0 {
		adminInfo, err := getUserTableInfo(rc.adminID)
		if err != nil {
			return err
		}
		commRemarks := fmt.Sprintf("DTH recharge commission | Ref: %s", refID)
		if err = debitTx(tx, transaction{
			UserID: rc.adminID, ReferenceID: refID,
			Amount: commision, Reason: "DTH_RECHARGE", Remarks: commRemarks,
			userTableInfo: *adminInfo,
		}, ds.walletStore); err != nil {
			return err
		}
		if err = creditTx(tx, transaction{
			UserID: dr.RetailerID, ReferenceID: refID,
			Amount: commision, Reason: "DTH_RECHARGE", Remarks: commRemarks,
			userTableInfo: *retailerInfo,
		}, ds.walletStore); err != nil {
			return err
		}
	}

	return tx.Commit()
}

func (ds *PostgresDTHRechargeStore) FinalizeDTHRecharge(id int64, operatorTxnID, orderID, status string) error {
	if !models.IsValidRechargeStatus(status) {
		return errors.New("invalid status")
	}
	res, err := ds.db.Exec(`
		UPDATE dth_recharge
		SET status = $2, operator_transaction_id = $3, order_id = $4
		WHERE dth_transaction_id = $1 AND status = 'PENDING'
	`, id, status, operatorTxnID, orderID)
	if err != nil {
		return err
	}
	n, err := res.RowsAffected()
	if err != nil {
		return err
	}
	if n == 0 {
		return errors.New("dth recharge not found or already finalized")
	}
	return nil
}

func (ds *PostgresDTHRechargeStore) RefundDTHRecharge(id int64) error {
	dr, err := ds.GetDTHRechargeByID(id)
	if err != nil {
		return err
	}
	if dr.Status != "FAILED" {
		return errors.New("only FAILED recharges can be refunded")
	}

	rc, err := getRetailerDetails(ds.db, dr.RetailerID)
	if err != nil {
		return err
	}

	tx, err := ds.db.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()

	// Atomic guard: prevents double-refund.
	res, err := tx.Exec(`
		UPDATE dth_recharge
		SET status = 'REFUND'
		WHERE dth_transaction_id = $1 AND status = 'FAILED'
	`, id)
	if err != nil {
		return err
	}
	n, err := res.RowsAffected()
	if err != nil {
		return err
	}
	if n == 0 {
		return errors.New("dth recharge not found or already refunded")
	}

	refID := fmt.Sprintf("%d", id)
	remarks := fmt.Sprintf("DTH recharge refund | Ref: %s", refID)

	retailerInfo, err := getUserTableInfo(dr.RetailerID)
	if err != nil {
		return err
	}

	// Reverse commission: debit from retailer, credit back to admin.
	if dr.Commision > 0 {
		if err = debitTx(tx, transaction{
			UserID: dr.RetailerID, ReferenceID: refID,
			Amount: dr.Commision, Reason: "DTH_RECHARGE_REFUND", Remarks: remarks,
			userTableInfo: *retailerInfo,
		}, ds.walletStore); err != nil {
			return err
		}
		adminInfo, err := getUserTableInfo(rc.adminID)
		if err != nil {
			return err
		}
		if err = creditTx(tx, transaction{
			UserID: rc.adminID, ReferenceID: refID,
			Amount: dr.Commision, Reason: "DTH_RECHARGE_REFUND", Remarks: remarks,
			userTableInfo: *adminInfo,
		}, ds.walletStore); err != nil {
			return err
		}
	}

	// Credit full recharge amount back to retailer.
	if err = creditTx(tx, transaction{
		UserID: dr.RetailerID, ReferenceID: refID,
		Amount: dr.Amount, Reason: "DTH_RECHARGE_REFUND", Remarks: remarks,
		userTableInfo: *retailerInfo,
	}, ds.walletStore); err != nil {
		return err
	}

	return tx.Commit()
}

const dthRechargeSelectBase = `
SELECT
	dr.dth_transaction_id, dr.partner_request_id,
	dr.retailer_id, dr.customer_id, dr.operator_name, dr.operator_code,
	dr.amount, dr.commision,
	COALESCE(dr.operator_transaction_id, '') AS operator_transaction_id,
	COALESCE(dr.order_id, '') AS order_id,
	dr.status, dr.created_at,
	COALESCE(r.retailer_name, '') AS retailer_name,
	r.retailer_business_name,
	COALESCE(wt.before_balance, 0) AS before_balance,
	COALESCE(wt.after_balance, 0) AS after_balance
FROM dth_recharge dr
JOIN retailers r ON dr.retailer_id = r.retailer_id
LEFT JOIN wallet_transactions wt ON wt.reference_id = dr.dth_transaction_id::TEXT
	AND wt.user_id = dr.retailer_id AND wt.debit_amount IS NOT NULL
`

func (ds *PostgresDTHRechargeStore) GetDTHRechargeByID(id int64) (*models.DTHRechargeModel, error) {
	q := dthRechargeSelectBase + `WHERE dr.dth_transaction_id = $1;`
	results, err := scanDTHRecharges(ds.db, q, id)
	if err != nil {
		return nil, err
	}
	if len(results) == 0 {
		return nil, errors.New("dth recharge not found")
	}
	return &results[0], nil
}

func (ds *PostgresDTHRechargeStore) GetAllDTHRecharge(p utils.QueryParams) ([]models.DTHRechargeModel, error) {
	q := dthRechargeSelectBase + `
	WHERE dr.created_at >= COALESCE($3, '-infinity'::TIMESTAMPTZ)
	AND dr.created_at <= COALESCE($4, 'infinity'::TIMESTAMPTZ)
	ORDER BY dr.created_at DESC
	LIMIT $1 OFFSET $2;`
	return scanDTHRecharges(ds.db, q, p.Limit, p.Offset, p.StartDate, p.EndDate)
}

func (ds *PostgresDTHRechargeStore) GetDTHRechargeByRetailerID(retailerID string, p utils.QueryParams) ([]models.DTHRechargeModel, error) {
	q := dthRechargeSelectBase + `
	WHERE dr.retailer_id = $1
	AND dr.created_at >= COALESCE($4, '-infinity'::TIMESTAMPTZ)
	AND dr.created_at <= COALESCE($5, 'infinity'::TIMESTAMPTZ)
	ORDER BY dr.created_at DESC
	LIMIT $2 OFFSET $3;`
	return scanDTHRecharges(ds.db, q, retailerID, p.Limit, p.Offset, p.StartDate, p.EndDate)
}

func (ds *PostgresDTHRechargeStore) GetDTHRechargeByDistributorID(distributorID string, p utils.QueryParams) ([]models.DTHRechargeModel, error) {
	q := dthRechargeSelectBase + `
	JOIN distributors d ON r.distributor_id = d.distributor_id
	WHERE d.distributor_id = $1
	AND dr.created_at >= COALESCE($4, '-infinity'::TIMESTAMPTZ)
	AND dr.created_at <= COALESCE($5, 'infinity'::TIMESTAMPTZ)
	ORDER BY dr.created_at DESC
	LIMIT $2 OFFSET $3;`
	return scanDTHRecharges(ds.db, q, distributorID, p.Limit, p.Offset, p.StartDate, p.EndDate)
}

func (ds *PostgresDTHRechargeStore) GetDTHRechargeByMasterDistributorID(mdID string, p utils.QueryParams) ([]models.DTHRechargeModel, error) {
	q := dthRechargeSelectBase + `
	JOIN distributors d ON r.distributor_id = d.distributor_id
	JOIN master_distributors md ON d.master_distributor_id = md.master_distributor_id
	WHERE md.master_distributor_id = $1
	AND dr.created_at >= COALESCE($4, '-infinity'::TIMESTAMPTZ)
	AND dr.created_at <= COALESCE($5, 'infinity'::TIMESTAMPTZ)
	ORDER BY dr.created_at DESC
	LIMIT $2 OFFSET $3;`
	return scanDTHRecharges(ds.db, q, mdID, p.Limit, p.Offset, p.StartDate, p.EndDate)
}

func scanDTHRecharges(db *sql.DB, q string, args ...any) ([]models.DTHRechargeModel, error) {
	rows, err := db.Query(q, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var results []models.DTHRechargeModel
	for rows.Next() {
		var dr models.DTHRechargeModel
		if err := rows.Scan(
			&dr.DTHTransactionID, &dr.PartnerRequestID,
			&dr.RetailerID, &dr.CustomerID, &dr.OperatorName, &dr.OperatorCode,
			&dr.Amount, &dr.Commision,
			&dr.OperatorTransactionID, &dr.OrderID,
			&dr.Status, &dr.CreatedAt,
			&dr.RetailerName, &dr.RetailerBusinessName,
			&dr.BeforeBalance, &dr.AfterBalance,
		); err != nil {
			return nil, err
		}
		results = append(results, dr)
	}
	return results, rows.Err()
}

func (ds *PostgresDTHRechargeStore) CreateDTHRechargeOperator(op models.DTHRechargeOperatorModel) error {
	_, err := ds.db.Exec(
		`INSERT INTO dth_recharge_operators (operator_code, operator_name) VALUES ($1, $2)`,
		op.OperatorCode, op.OperatorName,
	)
	return err
}

func (ds *PostgresDTHRechargeStore) UpdateDTHRechargeOperator(op models.DTHRechargeOperatorModel) error {
	res, err := ds.db.Exec(
		`UPDATE dth_recharge_operators SET operator_name = $2 WHERE operator_code = $1`,
		op.OperatorCode, op.OperatorName,
	)
	if err != nil {
		return err
	}
	return checkRowsAffected(res)
}

func (ds *PostgresDTHRechargeStore) DeleteDTHRechargeOperator(operatorCode int) error {
	res, err := ds.db.Exec(
		`DELETE FROM dth_recharge_operators WHERE operator_code = $1`, operatorCode,
	)
	if err != nil {
		return err
	}
	return checkRowsAffected(res)
}

func (ds *PostgresDTHRechargeStore) GetDTHRechargeOperators() ([]models.DTHRechargeOperatorModel, error) {
	rows, err := ds.db.Query(`SELECT operator_code, operator_name FROM dth_recharge_operators ORDER BY operator_name`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var operators []models.DTHRechargeOperatorModel
	for rows.Next() {
		var o models.DTHRechargeOperatorModel
		if err := rows.Scan(&o.OperatorCode, &o.OperatorName); err != nil {
			return nil, err
		}
		operators = append(operators, o)
	}
	return operators, rows.Err()
}
