package store

import (
	"database/sql"
	"errors"
	"fmt"

	"github.com/levionstudio/fintech/internal/models"
)

type walletInfo struct {
	table      string
	idCol      string
	balanceCol string
}

func walletInfoFromID(userID string) (walletInfo, error) {
	if len(userID) == 0 {
		return walletInfo{}, errors.New("empty user id")
	}
	switch string(userID[0]) {
	case "A":
		return walletInfo{"admins", "admin_id", "admin_wallet_balance"}, nil
	case "M":
		return walletInfo{"master_distributors", "master_distributor_id", "master_distributor_wallet_balance"}, nil
	case "D":
		return walletInfo{"distributors", "distributor_id", "distributor_wallet_balance"}, nil
	case "R":
		return walletInfo{"retailers", "retailer_id", "retailer_wallet_balance"}, nil
	default:
		return walletInfo{}, fmt.Errorf("unknown user type for id: %s", userID)
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

func checkExistsTx(tx *sql.Tx, table, idCol, id, role string) error {
	q := fmt.Sprintf(`SELECT EXISTS(SELECT 1 FROM %s WHERE %s = $1);`, table, idCol)
	var exists bool
	_ = tx.QueryRow(q, id).Scan(&exists)
	if !exists {
		return fmt.Errorf("%s not found", role)
	}
	return errors.New("insufficient balance")
}
