// Package db provides database connection configuration.
package db

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	_ "github.com/jackc/pgx/v5/stdlib"
)

// Connect establishes a connection to the database using the provided DBConf.
// Parameters:
//   - cfg: configuration containing the database URL and pool settings.
//
// Returns a ready-to-use sql.DB connection or an error if connection or ping fails.
func Connect(cfg *DBConf) (*sql.DB, error) {
	db, err := sql.Open("pgx", cfg.URL)
	if err != nil {
		return nil, fmt.Errorf("failed to open a connection to the DB: %w", err)
	}
	db.SetMaxOpenConns(cfg.MaxOpenConns)
	db.SetMaxIdleConns(cfg.MaxIdleConns)
	db.SetConnMaxLifetime(cfg.ConnMaxLifetime)

	ctx, cancel := context.WithTimeout(context.TODO(), 5*time.Second)
	defer cancel()
	if err := db.PingContext(ctx); err != nil {
		return nil, fmt.Errorf("connection to the DB is not ready: %w", err)
	}

	return db, nil
}
