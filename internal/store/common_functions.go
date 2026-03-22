package store

import (
	"database/sql"
	"errors"
	"fmt"

	"github.com/levionstudio/fintech/internal/models"
)

type userTableInfo struct {
	TableName               string
	IDColumnName            string
	WalletBalanceColumnName string
}

type transaction struct {
	UserID      string  `json:"user_id"`
	ReferenceID string  `json:"reference_id"`
	Amount      float64 `json:"amount"`
	Reason      string  `json:"reason"`
	Remarks     string  `json:"remarks"`
	userTableInfo
}

func getUserTableInfo(id string) (*userTableInfo, error) {
	if id == "" {
		return nil, errors.New("invalid user id")
	}
	switch string(id[0]) {
	case "A":
		return &userTableInfo{TableName: "admins", IDColumnName: "admin_id", WalletBalanceColumnName: "admin_wallet_balance"}, nil
	case "M":
		return &userTableInfo{TableName: "master_distributors", IDColumnName: "master_distributor_id", WalletBalanceColumnName: "master_distributor_wallet_balance"}, nil
	case "D":
		return &userTableInfo{TableName: "distributors", IDColumnName: "distributor_id", WalletBalanceColumnName: "distributor_wallet_balance"}, nil
	case "R":
		return &userTableInfo{TableName: "retailers", IDColumnName: "retailer_id", WalletBalanceColumnName: "retailer_wallet_balance"}, nil
	default:
		return nil, errors.New("invalid user id")
	}
}

func checkRowsAffected(res sql.Result) error {
	rowsAffected, err := res.RowsAffected()
	if err != nil {
		return err
	}
	if rowsAffected == 0 {
		return sql.ErrNoRows
	}
	return nil
}

func scanDropdown(db *sql.DB, query string, args ...any) ([]models.DropdownItem, error) {
	rows, err := db.Query(query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var items []models.DropdownItem
	for rows.Next() {
		var item models.DropdownItem
		if err := rows.Scan(&item.ID, &item.Name); err != nil {
			return nil, err
		}
		items = append(items, item)
	}

	return items, rows.Err()
}

func debitTx(tx *sql.Tx, txn transaction, wts WalletTransactionStore) error {
	if txn.Amount <= 0 {
		return nil
	}
	q := fmt.Sprintf(
		`UPDATE %s SET %s = %s - $1, updated_at = CURRENT_TIMESTAMP
		 WHERE %s = $2 AND %s >= $1
		 RETURNING %s + $1, %s;`,
		txn.TableName, txn.WalletBalanceColumnName,
		txn.WalletBalanceColumnName, txn.IDColumnName,
		txn.WalletBalanceColumnName, txn.WalletBalanceColumnName,
		txn.WalletBalanceColumnName,
	)
	var before, after float64
	err := tx.QueryRow(q, txn.Amount, txn.UserID).Scan(&before, &after)

	if err != nil {
		return err
	}

	err = wts.CreateWalletTransactionTx(tx, &models.WalletTransactionModel{
		UserID: txn.UserID, ReferenceID: txn.ReferenceID,
		DebitAmount: &txn.Amount, BeforeBalance: before, AfterBalance: after,
		TransactionReason: txn.Reason, Remarks: txn.Remarks,
	})
	return err
}

func creditTx(tx *sql.Tx, txn transaction, wts WalletTransactionStore) error {
	q := fmt.Sprintf(
		`UPDATE %s SET %s = %s + $1, updated_at = CURRENT_TIMESTAMP
		 WHERE %s = $2
		 RETURNING %s - $1, %s;`,
		txn.TableName, txn.WalletBalanceColumnName, txn.WalletBalanceColumnName, idCol, balanceCol, balanceCol,
	)
	err = tx.QueryRow(q, amount, id).Scan(&before, &after)
	return
}

func checkExistsTx(tx *sql.Tx, table, idCol, id, role string) error {
	q := fmt.Sprintf(`SELECT EXISTS(SELECT 1 FROM %s WHERE %s = $1);`, table, idCol)
	var exists bool
	_ = tx.QueryRow(q, id).Scan(&exists)
	if !exists {
		return fmt.Errorf("%s not found", role)
	}
	return errors.New("insufficient balance")
}

func (ps *PostgresPayoutTransactionStore) resolveCommision(
	retailerID, distributorID, mdID, adminID, service string,
	amount float64,
) *models.CommisionModel {
	for _, userID := range []string{retailerID, distributorID, mdID, adminID} {
		c, err := ps.commisionStore.GetCommisionByUserIDServiceAndAmount(userID, service, amount)
		if err == nil && c != nil {
			return c
		}
	}
	return ps.commisionStore.GetDefaultCommision(amount)
}

func (ps *PostgresPayoutTransactionStore) getRetailerTransactionLimit(retailerID, service string) (float64, error) {
	limit, _, err := ps.transactionLimitStore.GetTransactionLimitByRetailerIDAndService(&models.TransactionLimitModel{RetailerID: retailerID, Service: service})
	if err != nil {
		return 0, err
	}
	return limit, nil
}

func getRetailerDetails(db *sql.DB, retailerID string) (retailerChain, error) {
	const q = `
	SELECT
		r.retailer_kyc_status,
		r.is_retailer_blocked,
		r.retailer_wallet_balance,
		r.distributor_id,
		d.master_distributor_id,
		md.admin_id
	FROM retailers r
	JOIN distributors d         ON r.distributor_id         = d.distributor_id
	JOIN master_distributors md ON d.master_distributor_id  = md.master_distributor_id
	WHERE r.retailer_id = $1;
	`
	var rc retailerChain
	err := db.QueryRow(q, retailerID).Scan(
		&rc.kyc, &rc.blocked, &rc.balance,
		&rc.distributorID, &rc.mdID, &rc.adminID,
	)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return retailerChain{}, errors.New("retailer not found")
		}
		return retailerChain{}, err
	}
	return rc, nil
}

func (ps *PostgresPayoutTransactionStore) creditIfNonZero(
	tx *sql.Tx,
	userID, refID, remarks, service string,
	amount float64,
) error {
	if amount <= 0 {
		return nil
	}
	wi, err := walletInfoFromID(userID)
	if err != nil {
		return err
	}
	before, after, err := creditTx(tx, wi.table, wi.idCol, wi.balanceCol, userID, amount)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return fmt.Errorf("%s not found", userID)
		}
		return err
	}
	return ps.walletStore.CreateWalletTransactionTx(tx, &models.WalletTransactionModel{
		UserID: userID, ReferenceID: refID,
		CreditAmount: &amount, BeforeBalance: before, AfterBalance: after,
		TransactionReason: service, Remarks: remarks,
	})
}

func (ps *PostgresPayoutTransactionStore) debitIfNonZero(
	tx *sql.Tx,
	userID, refID, remarks, service string,
	amount float64,
) error {
	if amount <= 0 {
		return nil
	}
	wi, err := walletInfoFromID(userID)
	if err != nil {
		return err
	}
	before, after, err := debitTx(tx, wi.table, wi.idCol, wi.balanceCol, userID, amount)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return fmt.Errorf("%s not found", userID)
		}
		return err
	}
	return ps.walletStore.CreateWalletTransactionTx(tx, &models.WalletTransactionModel{
		UserID: userID, ReferenceID: refID,
		DebitAmount: &amount, BeforeBalance: before, AfterBalance: after,
		TransactionReason: service, Remarks: remarks,
	})
}
