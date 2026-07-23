// Package store provides database abstraction supporting PostgreSQL and SQLite.
package store

import (
	"database/sql"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	_ "modernc.org/sqlite"

	// PostgreSQL driver — imported for side-effect registration.
	// To enable, uncomment and run: go get github.com/lib/pq
	// _ "github.com/lib/pq"
)

// DB wraps sql.DB with application-specific query methods.
type DB struct {
	*sql.DB
	driver string // "postgres" or "sqlite"
}

// Open connects to a database based on the DSN.
//
//	postgres://user:pass@host:port/dbname?sslmode=disable
//	sqlite:///path/to/aster.db
func Open(dsn string) (*DB, error) {
	driver, connStr, err := parseDSN(dsn)
	if err != nil {
		return nil, err
	}

	if driver == "sqlite" {
		dir := filepath.Dir(connStr)
		if err := os.MkdirAll(dir, 0o755); err != nil {
			return nil, fmt.Errorf("mkdir %s: %w", dir, err)
		}
	}

	sqlDB, err := sql.Open(driver, connStr)
	if err != nil {
		return nil, fmt.Errorf("open %s: %w", driver, err)
	}

	if err := sqlDB.Ping(); err != nil {
		sqlDB.Close()
		return nil, fmt.Errorf("ping %s: %w", driver, err)
	}

	// SQLite pragmas
	if driver == "sqlite" {
		for _, pragma := range []string{
			"PRAGMA journal_mode=WAL",
			"PRAGMA foreign_keys=ON",
			"PRAGMA busy_timeout=5000",
		} {
			sqlDB.Exec(pragma)
		}
	}

	db := &DB{DB: sqlDB, driver: driver}
	if err := db.migrate(); err != nil {
		sqlDB.Close()
		return nil, fmt.Errorf("migrate: %w", err)
	}

	return db, nil
}

func OpenDefault(dataDir string) (*DB, error) {
	dsn := os.Getenv("ASTER_DATABASE_URL")
	if dsn == "" {
		dsn = "sqlite:///" + filepath.Join(dataDir, "aster.db")
	}
	return Open(dsn)
}

func parseDSN(dsn string) (driver, connStr string, err error) {
	if strings.HasPrefix(dsn, "postgres://") || strings.HasPrefix(dsn, "postgresql://") {
		return "postgres", dsn, nil
	}
	if strings.HasPrefix(dsn, "sqlite://") {
		return "sqlite", strings.TrimPrefix(dsn, "sqlite://"), nil
	}
	// Bare path → sqlite
	if !strings.Contains(dsn, "://") {
		return "sqlite", dsn, nil
	}
	return "", "", fmt.Errorf("unsupported DSN scheme: %s", dsn)
}

func (db *DB) DriverName() string { return db.driver }
