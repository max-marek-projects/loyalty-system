// Package db provides database configuration.
package db

import "time"

// DBConf holds database connection configuration (URL, pool limits, migrations path).
type DBConf struct {
	URL             string        // database connection url
	MaxOpenConns    int           // max amount of opened database connections
	MaxIdleConns    int           // max amount of idle database connections
	ConnMaxLifetime time.Duration // max database connection lifetime
	MigrationsPath  string        // path to folder with migrations files
}

// NewDBConf creates a DBConf with default connection pool settings and the given database URL.
// Parameters:
//   - dbURL: database connection string.
// Returns a pointer to the initialized DBConf.
func NewDBConf(dbURL string) *DBConf {
	return &DBConf{
		URL:             dbURL,
		MaxOpenConns:    10,
		MaxIdleConns:    5,
		ConnMaxLifetime: 5 * time.Minute,
		MigrationsPath:  "./migrations",
	}
}
