package store

import (
	"database/sql"
	"errors"
	"fmt"

	"github.com/google/uuid"
	"github.com/levionstudio/fintech/internal/models"
	"github.com/levionstudio/fintech/internal/utils"
)

const mobileRechargeCommision = 1.0

type PostgresMobileRechargeStore struct {
	db          *sql.DB
	walletStore WalletTransactionStore
}

func NewPostgresMobileRechargeStore(db *sql.DB, walletStore WalletTransactionStore) *PostgresMobileRechargeStore {
	return &PostgresMobileRechargeStore{db: db, walletStore: walletStore}
}

type MobileRechargeStore interface {
	InitializeMobileRecharge(mr *models.MobileRechargeModel) error
	FinalizeMobileRecharge(id int64, operatorTxnID, orderID, status string) error
	RefundMobileRecharge(id int64) error
	GetMobileRechargeByID(id int64) (*models.MobileRechargeModel, error)
	GetAllMobileRecharge(p utils.QueryParams) ([]models.MobileRechargeModel, error)
	GetMobileRechargeByRetailerID(retailerID string, p utils.QueryParams) ([]models.MobileRechargeModel, error)
	GetMobileRechargeByDistributorID(distributorID string, p utils.QueryParams) ([]models.MobileRechargeModel, error)
	GetMobileRechargeByMasterDistributorID(mdID string, p utils.QueryParams) ([]models.MobileRechargeModel, error)
	CreateMobileRechargeCircle(circle models.MobileRechargeCircleModel) error
	UpdateMobileRechargeCircle(circle models.MobileRechargeCircleModel) error
	DeleteMobileRechargeCircle(circleCode int) error
	GetMobileRechargeCircles() ([]models.MobileRechargeCircleModel, error)
	CreateMobileRechargeOperator(op models.MobileRechargeOperatorModel) error
	UpdateMobileRechargeOperator(op models.MobileRechargeOperatorModel) error
	DeleteMobileRechargeOperator(operatorCode int) error
	GetMobileRechargeOperators() ([]models.MobileRechargeOperatorModel, error)
}

func (ms *PostgresMobileRechargeStore) InitializeMobileRecharge(mr *models.MobileRechargeModel) error {
	rc, err := getRetailerDetails(ms.db, mr.RetailerID)
	if err != nil {
		return err
	}
	if !rc.kyc {
		return errors.New("retailer KYC is not verified")
	}
	if rc.blocked {
		return errors.New("retailer is blocked")
	}
	if rc.balance < mr.Amount {
		return errors.New("insufficient wallet balance")
	}

	commision := 0.0
	if mr.Amount > 100 {
		commision = mobileRechargeCommision
	}

	mr.PartnerRequestID = uuid.New().String()
	mr.Commision = commision
	mr.RechargeStatus = "PENDING"

	walletReason := "MOBILE_RECHARGE"
	if mr.RechargeType == "POSTPAID" {
		walletReason = "POSTPAID_MOBILE_RECHARGE"
	}

	tx, err := ms.db.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()

	// INSERT first to get ID as wallet entry reference.
	// operator_transaction_id and order_id start empty and are filled by FinalizeMobileRecharge.
	if err = tx.QueryRow(`
		INSERT INTO mobile_recharge (
			retailer_id, partner_request_id, mobile_number,
			operator_name, circle_name, operator_code, circle_code,
			amount, commision, recharge_type,
			operator_transaction_id, order_id, recharge_status
		) VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,'','',$11)
		RETURNING mobile_recharge_transaction_id, created_at
	`,
		mr.RetailerID, mr.PartnerRequestID, mr.MobileNumber,
		mr.OperatorName, mr.CircleName, mr.OperatorCode, mr.CircleCode,
		mr.Amount, mr.Commision, mr.RechargeType, mr.RechargeStatus,
	).Scan(&mr.MobileRechargeTransactionID, &mr.CreatedAt); err != nil {
		return err
	}

	refID := fmt.Sprintf("%d", mr.MobileRechargeTransactionID)
	rechargeRemarks := fmt.Sprintf("Mobile recharge | %s", mr.MobileNumber)

	retailerInfo, err := getUserTableInfo(mr.RetailerID)
	if err != nil {
		return err
	}

	// Debit recharge amount from retailer.
	if err = debitTx(tx, transaction{
		UserID: mr.RetailerID, ReferenceID: refID,
		Amount: mr.Amount, Reason: walletReason, Remarks: rechargeRemarks,
		userTableInfo: *retailerInfo,
	}, ms.walletStore); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return checkExistsTx(tx, retailerInfo.TableName, retailerInfo.IDColumnName, mr.RetailerID, "retailer")
		}
		return err
	}

	// If amount > ₹100: admin pays ₹1 commission to retailer.
	if commision > 0 {
		adminInfo, err := getUserTableInfo(rc.adminID)
		if err != nil {
			return err
		}
		commRemarks := fmt.Sprintf("Mobile recharge commission | Ref: %s", refID)
		if err = debitTx(tx, transaction{
			UserID: rc.adminID, ReferenceID: refID,
			Amount: commision, Reason: walletReason, Remarks: commRemarks,
			userTableInfo: *adminInfo,
		}, ms.walletStore); err != nil {
			return err
		}
		if err = creditTx(tx, transaction{
			UserID: mr.RetailerID, ReferenceID: refID,
			Amount: commision, Reason: walletReason, Remarks: commRemarks,
			userTableInfo: *retailerInfo,
		}, ms.walletStore); err != nil {
			return err
		}
	}

	return tx.Commit()
}

func (ms *PostgresMobileRechargeStore) FinalizeMobileRecharge(id int64, operatorTxnID, orderID, status string) error {
	if !models.IsValidRechargeStatus(status) {
		return errors.New("invalid recharge_status")
	}
	res, err := ms.db.Exec(`
		UPDATE mobile_recharge
		SET recharge_status          = $2,
		    operator_transaction_id  = $3,
		    order_id                 = $4
		WHERE mobile_recharge_transaction_id = $1 AND recharge_status = 'PENDING'
	`, id, status, operatorTxnID, orderID)
	if err != nil {
		return err
	}
	n, err := res.RowsAffected()
	if err != nil {
		return err
	}
	if n == 0 {
		return errors.New("recharge not found or already finalized")
	}
	return nil
}

func (ms *PostgresMobileRechargeStore) RefundMobileRecharge(id int64) error {
	mr, err := ms.GetMobileRechargeByID(id)
	if err != nil {
		return err
	}
	if mr.RechargeStatus != "FAILED" {
		return errors.New("only FAILED recharges can be refunded")
	}

	rc, err := getRetailerDetails(ms.db, mr.RetailerID)
	if err != nil {
		return err
	}

	tx, err := ms.db.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()

	// Atomic guard: prevents double-refund.
	res, err := tx.Exec(`
		UPDATE mobile_recharge
		SET recharge_status = 'REFUND'
		WHERE mobile_recharge_transaction_id = $1 AND recharge_status = 'FAILED'
	`, id)
	if err != nil {
		return err
	}
	n, err := res.RowsAffected()
	if err != nil {
		return err
	}
	if n == 0 {
		return errors.New("recharge not found or already refunded")
	}

	refID := fmt.Sprintf("%d", id)
	remarks := fmt.Sprintf("Mobile recharge refund | Ref: %s", refID)

	retailerInfo, err := getUserTableInfo(mr.RetailerID)
	if err != nil {
		return err
	}

	// Reverse commission: debit from retailer, credit back to admin.
	if mr.Commision > 0 {
		if err = debitTx(tx, transaction{
			UserID: mr.RetailerID, ReferenceID: refID,
			Amount: mr.Commision, Reason: "MOBILE_RECHARGE_REFUND", Remarks: remarks,
			userTableInfo: *retailerInfo,
		}, ms.walletStore); err != nil {
			return err
		}
		adminInfo, err := getUserTableInfo(rc.adminID)
		if err != nil {
			return err
		}
		if err = creditTx(tx, transaction{
			UserID: rc.adminID, ReferenceID: refID,
			Amount: mr.Commision, Reason: "MOBILE_RECHARGE_REFUND", Remarks: remarks,
			userTableInfo: *adminInfo,
		}, ms.walletStore); err != nil {
			return err
		}
	}

	// Credit full recharge amount back to retailer.
	if err = creditTx(tx, transaction{
		UserID: mr.RetailerID, ReferenceID: refID,
		Amount: mr.Amount, Reason: "MOBILE_RECHARGE_REFUND", Remarks: remarks,
		userTableInfo: *retailerInfo,
	}, ms.walletStore); err != nil {
		return err
	}

	return tx.Commit()
}

const mobileRechargeSelectBase = `
SELECT
	mr.mobile_recharge_transaction_id, mr.partner_request_id,
	mr.retailer_id, mr.mobile_number, mr.operator_name, mr.circle_name,
	mr.operator_code, mr.circle_code, mr.amount, mr.commision,
	mr.recharge_type, mr.operator_transaction_id, mr.order_id,
	mr.recharge_status, mr.created_at,
	COALESCE(r.retailer_name, '') AS retailer_name,
	r.retailer_business_name,
	COALESCE(wt.before_balance, 0) AS before_balance,
	COALESCE(wt.after_balance, 0) AS after_balance
FROM mobile_recharge mr
JOIN retailers r ON mr.retailer_id = r.retailer_id
LEFT JOIN wallet_transactions wt ON wt.reference_id = mr.mobile_recharge_transaction_id::TEXT
	AND wt.user_id = mr.retailer_id AND wt.debit_amount IS NOT NULL
`

func (ms *PostgresMobileRechargeStore) GetMobileRechargeByID(id int64) (*models.MobileRechargeModel, error) {
	q := mobileRechargeSelectBase + `WHERE mr.mobile_recharge_transaction_id = $1;`
	results, err := scanMobileRecharges(ms.db, q, id)
	if err != nil {
		return nil, err
	}
	if len(results) == 0 {
		return nil, errors.New("recharge not found")
	}
	return &results[0], nil
}

func (ms *PostgresMobileRechargeStore) GetAllMobileRecharge(p utils.QueryParams) ([]models.MobileRechargeModel, error) {
	q := mobileRechargeSelectBase + `
	WHERE mr.created_at >= COALESCE($3, '-infinity'::TIMESTAMPTZ)
	AND mr.created_at <= COALESCE($4, 'infinity'::TIMESTAMPTZ)
	ORDER BY mr.created_at DESC
	LIMIT $1 OFFSET $2;`
	return scanMobileRecharges(ms.db, q, p.Limit, p.Offset, p.StartDate, p.EndDate)
}

func (ms *PostgresMobileRechargeStore) GetMobileRechargeByRetailerID(retailerID string, p utils.QueryParams) ([]models.MobileRechargeModel, error) {
	q := mobileRechargeSelectBase + `
	WHERE mr.retailer_id = $1
	AND mr.created_at >= COALESCE($4, '-infinity'::TIMESTAMPTZ)
	AND mr.created_at <= COALESCE($5, 'infinity'::TIMESTAMPTZ)
	ORDER BY mr.created_at DESC
	LIMIT $2 OFFSET $3;`
	return scanMobileRecharges(ms.db, q, retailerID, p.Limit, p.Offset, p.StartDate, p.EndDate)
}

func (ms *PostgresMobileRechargeStore) GetMobileRechargeByDistributorID(distributorID string, p utils.QueryParams) ([]models.MobileRechargeModel, error) {
	q := mobileRechargeSelectBase + `
	JOIN distributors d ON r.distributor_id = d.distributor_id
	WHERE d.distributor_id = $1
	AND mr.created_at >= COALESCE($4, '-infinity'::TIMESTAMPTZ)
	AND mr.created_at <= COALESCE($5, 'infinity'::TIMESTAMPTZ)
	ORDER BY mr.created_at DESC
	LIMIT $2 OFFSET $3;`
	return scanMobileRecharges(ms.db, q, distributorID, p.Limit, p.Offset, p.StartDate, p.EndDate)
}

func (ms *PostgresMobileRechargeStore) GetMobileRechargeByMasterDistributorID(mdID string, p utils.QueryParams) ([]models.MobileRechargeModel, error) {
	q := mobileRechargeSelectBase + `
	JOIN distributors d ON r.distributor_id = d.distributor_id
	JOIN master_distributors md ON d.master_distributor_id = md.master_distributor_id
	WHERE md.master_distributor_id = $1
	AND mr.created_at >= COALESCE($4, '-infinity'::TIMESTAMPTZ)
	AND mr.created_at <= COALESCE($5, 'infinity'::TIMESTAMPTZ)
	ORDER BY mr.created_at DESC
	LIMIT $2 OFFSET $3;`
	return scanMobileRecharges(ms.db, q, mdID, p.Limit, p.Offset, p.StartDate, p.EndDate)
}

func scanMobileRecharges(db *sql.DB, q string, args ...any) ([]models.MobileRechargeModel, error) {
	rows, err := db.Query(q, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var results []models.MobileRechargeModel
	for rows.Next() {
		var mr models.MobileRechargeModel
		if err := rows.Scan(
			&mr.MobileRechargeTransactionID, &mr.PartnerRequestID,
			&mr.RetailerID, &mr.MobileNumber, &mr.OperatorName, &mr.CircleName,
			&mr.OperatorCode, &mr.CircleCode, &mr.Amount, &mr.Commision,
			&mr.RechargeType, &mr.OperatorTransactionID, &mr.OrderID,
			&mr.RechargeStatus, &mr.CreatedAt,
			&mr.RetailerName, &mr.RetailerBusinessName,
			&mr.BeforeBalance, &mr.AfterBalance,
		); err != nil {
			return nil, err
		}
		results = append(results, mr)
	}
	return results, rows.Err()
}

func (ms *PostgresMobileRechargeStore) CreateMobileRechargeCircle(circle models.MobileRechargeCircleModel) error {
	_, err := ms.db.Exec(
		`INSERT INTO mobile_recharge_circles (circle_code, circle_name) VALUES ($1, $2)`,
		circle.CircleCode, circle.CircleName,
	)
	return err
}

func (ms *PostgresMobileRechargeStore) UpdateMobileRechargeCircle(circle models.MobileRechargeCircleModel) error {
	res, err := ms.db.Exec(
		`UPDATE mobile_recharge_circles SET circle_name = $2 WHERE circle_code = $1`,
		circle.CircleCode, circle.CircleName,
	)
	if err != nil {
		return err
	}
	return checkRowsAffected(res)
}

func (ms *PostgresMobileRechargeStore) DeleteMobileRechargeCircle(circleCode int) error {
	res, err := ms.db.Exec(
		`DELETE FROM mobile_recharge_circles WHERE circle_code = $1`, circleCode,
	)
	if err != nil {
		return err
	}
	return checkRowsAffected(res)
}

func (ms *PostgresMobileRechargeStore) GetMobileRechargeCircles() ([]models.MobileRechargeCircleModel, error) {
	rows, err := ms.db.Query(`SELECT circle_code, circle_name FROM mobile_recharge_circles ORDER BY circle_name`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var circles []models.MobileRechargeCircleModel
	for rows.Next() {
		var c models.MobileRechargeCircleModel
		if err := rows.Scan(&c.CircleCode, &c.CircleName); err != nil {
			return nil, err
		}
		circles = append(circles, c)
	}
	return circles, rows.Err()
}

func (ms *PostgresMobileRechargeStore) CreateMobileRechargeOperator(op models.MobileRechargeOperatorModel) error {
	_, err := ms.db.Exec(
		`INSERT INTO mobile_recharge_operators (operator_code, operator_name) VALUES ($1, $2)`,
		op.OperatorCode, op.OperatorName,
	)
	return err
}

func (ms *PostgresMobileRechargeStore) UpdateMobileRechargeOperator(op models.MobileRechargeOperatorModel) error {
	res, err := ms.db.Exec(
		`UPDATE mobile_recharge_operators SET operator_name = $2 WHERE operator_code = $1`,
		op.OperatorCode, op.OperatorName,
	)
	if err != nil {
		return err
	}
	return checkRowsAffected(res)
}

func (ms *PostgresMobileRechargeStore) DeleteMobileRechargeOperator(operatorCode int) error {
	res, err := ms.db.Exec(
		`DELETE FROM mobile_recharge_operators WHERE operator_code = $1`, operatorCode,
	)
	if err != nil {
		return err
	}
	return checkRowsAffected(res)
}

func (ms *PostgresMobileRechargeStore) GetMobileRechargeOperators() ([]models.MobileRechargeOperatorModel, error) {
	rows, err := ms.db.Query(`SELECT operator_code, operator_name FROM mobile_recharge_operators ORDER BY operator_name`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var operators []models.MobileRechargeOperatorModel
	for rows.Next() {
		var o models.MobileRechargeOperatorModel
		if err := rows.Scan(&o.OperatorCode, &o.OperatorName); err != nil {
			return nil, err
		}
		operators = append(operators, o)
	}
	return operators, rows.Err()
}
