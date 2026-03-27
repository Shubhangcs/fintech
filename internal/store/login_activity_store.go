package store

import (
	"database/sql"

	"github.com/levionstudio/fintech/internal/models"
	"github.com/levionstudio/fintech/internal/utils"
)

type LoginActivityStore interface {
	CreateLoginActivity(activity models.LoginActivity) error
	GetAllLoginActivities(p utils.QueryParams) ([]models.LoginActivity, error)
	GetLoginActivitiesByUserID(userID string, p utils.QueryParams) ([]models.LoginActivity, error)
}

type PostgresLoginActivityStore struct {
	db *sql.DB
}

func NewPostgresLoginActivityStore(db *sql.DB) *PostgresLoginActivityStore {
	return &PostgresLoginActivityStore{db: db}
}

func (ls *PostgresLoginActivityStore) CreateLoginActivity(activity models.LoginActivity) error {
	_, err := ls.db.Exec(`
		INSERT INTO login_activities (user_id, user_agent, platform, latitude, longitude, accuracy, login_timestamp)
		VALUES ($1, $2, $3, $4, $5, $6, $7)
	`, activity.UserID, activity.UserAgent, activity.Platform,
		activity.Latitude, activity.Longitude, activity.Accuracy, activity.LoginTimestamp)
	return err
}

func (ls *PostgresLoginActivityStore) GetAllLoginActivities(p utils.QueryParams) ([]models.LoginActivity, error) {
	rows, err := ls.db.Query(`
		SELECT login_id, user_id, user_agent, platform, latitude, longitude, accuracy, login_timestamp, created_at
		FROM login_activities
		WHERE created_at >= COALESCE($3, '-infinity'::TIMESTAMPTZ)
		AND created_at <= COALESCE($4, 'infinity'::TIMESTAMPTZ)
		ORDER BY created_at DESC
		LIMIT $1 OFFSET $2
	`, p.Limit, p.Offset, p.StartDate, p.EndDate)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	return scanLoginActivities(rows)
}

func (ls *PostgresLoginActivityStore) GetLoginActivitiesByUserID(userID string, p utils.QueryParams) ([]models.LoginActivity, error) {
	rows, err := ls.db.Query(`
		SELECT login_id, user_id, user_agent, platform, latitude, longitude, accuracy, login_timestamp, created_at
		FROM login_activities
		WHERE user_id = $1
		AND created_at >= COALESCE($4, '-infinity'::TIMESTAMPTZ)
		AND created_at <= COALESCE($5, 'infinity'::TIMESTAMPTZ)
		ORDER BY created_at DESC
		LIMIT $2 OFFSET $3
	`, userID, p.Limit, p.Offset, p.StartDate, p.EndDate)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	return scanLoginActivities(rows)
}

func scanLoginActivities(rows *sql.Rows) ([]models.LoginActivity, error) {
	var results []models.LoginActivity
	for rows.Next() {
		var a models.LoginActivity
		if err := rows.Scan(
			&a.LoginID, &a.UserID, &a.UserAgent, &a.Platform,
			&a.Latitude, &a.Longitude, &a.Accuracy,
			&a.LoginTimestamp, &a.CreatedAt,
		); err != nil {
			return nil, err
		}
		results = append(results, a)
	}
	return results, rows.Err()
}
