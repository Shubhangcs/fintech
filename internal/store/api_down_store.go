package store

import (
	"database/sql"
	"errors"

	"github.com/levionstudio/fintech/internal/models"
)

type ApiDownStore interface {
	IsServiceDown(serviceName string) (bool, error)
	UpdateServiceStatus(serviceName string, status bool) error
	GetAllServiceStatuses() ([]models.ApiDownModel, error)
}

type PostgresApiDownStore struct {
	db *sql.DB
}

func NewPostgresApiDownStore(db *sql.DB) *PostgresApiDownStore {
	return &PostgresApiDownStore{db: db}
}

func (s *PostgresApiDownStore) IsServiceDown(serviceName string) (bool, error) {
	var status bool
	err := s.db.QueryRow(
		`SELECT status FROM api_down WHERE service_name = $1`,
		serviceName,
	).Scan(&status)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return false, nil
		}
		return false, err
	}
	return status, nil
}

func (s *PostgresApiDownStore) UpdateServiceStatus(serviceName string, status bool) error {
	res, err := s.db.Exec(
		`UPDATE api_down SET status = $1, updated_at = CURRENT_TIMESTAMP WHERE service_name = $2`,
		status, serviceName,
	)
	if err != nil {
		return err
	}
	return checkRowsAffected(res)
}

func (s *PostgresApiDownStore) GetAllServiceStatuses() ([]models.ApiDownModel, error) {
	rows, err := s.db.Query(
		`SELECT api_down_id, service_name, status, created_at, updated_at FROM api_down ORDER BY service_name`,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var results []models.ApiDownModel
	for rows.Next() {
		var a models.ApiDownModel
		if err := rows.Scan(&a.ApiDownID, &a.ServiceName, &a.Status, &a.CreatedAt, &a.UpdatedAt); err != nil {
			return nil, err
		}
		results = append(results, a)
	}
	return results, rows.Err()
}
