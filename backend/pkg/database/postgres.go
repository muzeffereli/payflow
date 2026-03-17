package database

import (
	"database/sql"
	"fmt"
	"time"

	"payment-platform/pkg/config"

	_ "github.com/lib/pq" // PostgreSQL driver â€” imported for side effects (registers the driver)
)

func Connect(cfg config.DBConfig) (*sql.DB, error) {
	db, err := sql.Open("postgres", cfg.DSN())
	if err != nil {
		return nil, fmt.Errorf("open db: %w", err)
	}

	db.SetMaxOpenConns(25)
	db.SetMaxIdleConns(10)
	db.SetConnMaxLifetime(5 * time.Minute) // recycle connections to avoid stale TCP

	if err := db.Ping(); err != nil {
		return nil, fmt.Errorf("ping db: %w", err)
	}

	return db, nil
}
