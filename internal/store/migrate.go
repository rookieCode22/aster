package store

import (
	"fmt"
	"sort"
)

type migration struct {
	version int
	name    string
	sql     string // can use {{PLACEHOLDER}} for driver-specific SQL
}

var migrations = []migration{
	{1, "create_users", `
CREATE TABLE IF NOT EXISTS users (
	id         TEXT PRIMARY KEY,
	username   TEXT UNIQUE NOT NULL,
	password   TEXT NOT NULL,
	created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);
CREATE INDEX IF NOT EXISTS idx_users_username ON users(username);
`},
	{2, "create_sessions", `
CREATE TABLE IF NOT EXISTS sessions (
	id         TEXT PRIMARY KEY,
	user_id    TEXT NOT NULL REFERENCES users(id) ON DELETE CASCADE,
	title      TEXT NOT NULL DEFAULT '',
	agent_name TEXT NOT NULL DEFAULT 'code-audit',
	created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
	updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);
CREATE INDEX IF NOT EXISTS idx_sessions_user ON sessions(user_id);
`},
	{3, "create_skills", `
CREATE TABLE IF NOT EXISTS custom_skills (
	id          TEXT PRIMARY KEY,
	user_id     TEXT NOT NULL REFERENCES users(id) ON DELETE CASCADE,
	name        TEXT NOT NULL,
	description TEXT NOT NULL DEFAULT '',
	instructions TEXT NOT NULL,
	agent       TEXT NOT NULL DEFAULT 'all',
	tags        TEXT NOT NULL DEFAULT '[]',
	created_at  TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
	updated_at  TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
	UNIQUE(user_id, name)
);
CREATE INDEX IF NOT EXISTS idx_skills_user ON custom_skills(user_id);
`},
	{4, "create_licenses", `
CREATE TABLE IF NOT EXISTS licenses (
	id           TEXT PRIMARY KEY,
	customer     TEXT NOT NULL DEFAULT '',
	email        TEXT NOT NULL DEFAULT '',
	plan         TEXT NOT NULL DEFAULT 'trial',
	modules      TEXT NOT NULL DEFAULT '["*"]',
	issued_at    TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
	expires_at   TIMESTAMP,
	max_agents   INTEGER NOT NULL DEFAULT 0,
	hwid         TEXT NOT NULL DEFAULT '',
	full_json    TEXT NOT NULL DEFAULT '',
	activated_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
	is_active    BOOLEAN NOT NULL DEFAULT false
);
CREATE INDEX IF NOT EXISTS idx_licenses_active ON licenses(is_active);
`},
	{5, "create_schema_version", `
CREATE TABLE IF NOT EXISTS schema_version (
	version INTEGER PRIMARY KEY,
	applied_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);
`},
}

func (db *DB) migrate() error {
	// Ensure schema_version exists first
	for _, m := range migrations {
		if m.version == 5 {
			db.DB.Exec(m.sql)
		}
	}

	current := db.currentVersion()

	sort.Slice(migrations, func(i, j int) bool {
		return migrations[i].version < migrations[j].version
	})

	for _, m := range migrations {
		if m.version <= current {
			continue
		}
		sql := db.adaptSQL(m.sql)
		if _, err := db.DB.Exec(sql); err != nil {
			return fmt.Errorf("migration %d (%s): %w", m.version, m.name, err)
		}
		if _, err := db.DB.Exec("INSERT INTO schema_version (version) VALUES (?)", m.version); err != nil {
			return fmt.Errorf("record migration %d: %w", m.version, err)
		}
	}
	return nil
}

func (db *DB) currentVersion() int {
	var v int
	err := db.DB.QueryRow("SELECT COALESCE(MAX(version), 0) FROM schema_version").Scan(&v)
	if err != nil {
		return 0
	}
	return v
}

// adaptSQL handles driver-specific SQL differences.
func (db *DB) adaptSQL(sql string) string {
	return sql // PG and SQLite share these simple DDL statements
}
