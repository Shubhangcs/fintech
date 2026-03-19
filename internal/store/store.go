package store

import (
	"database/sql"
	"fmt"
	"os"
	"time"

	_ "github.com/jackc/pgx/v4/stdlib"
)

func Open() (*sql.DB, error) {
	var (
		hostName = os.Getenv("DB_HOST_NAME")
		userName = os.Getenv("DB_USER_NAME")
		port     = os.Getenv("DB_PORT")
		password = os.Getenv("DB_PASSWORD")
		name     = os.Getenv("DB_NAME")
		sslMode  = os.Getenv("DB_SSL_MODE")
	)

	db, err := sql.Open(
		"pgx",
		fmt.Sprintf(
			"host=%s port=%s user=%s dbname=%s password=%s sslmode=%s",
			hostName,
			port,
			userName,
			name,
			password,
			sslMode,
		),
	)

	if err != nil {
		return nil, err
	}

	if err := db.Ping(); err != nil {
		return nil, err
	}

	db.SetMaxOpenConns(25)
	db.SetMaxIdleConns(10)
	db.SetConnMaxLifetime(5 * time.Minute)
	db.SetConnMaxIdleTime(2 * time.Minute)

	return db, nil
}
