package store

import (
	"database/sql"

	"github.com/levionstudio/fintech/internal/models"
)

type LoginActivityStore interface {
	CreateLoginActivity(activity models.LoginActivity) error
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
