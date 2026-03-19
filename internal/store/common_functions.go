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
