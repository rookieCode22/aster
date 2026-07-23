package store

import (
	"crypto/rand"
	"crypto/sha256"
	"database/sql"
	"encoding/hex"
	"fmt"
	"time"
)

// UserRow represents a user record.
type UserRow struct {
	ID        string    `json:"id"`
	Username  string    `json:"username"`
	Password  string    `json:"-"` // hash, never serialized
	CreatedAt time.Time `json:"created_at"`
}

// CreateUser inserts a new user with hashed password.
func (db *DB) CreateUser(username, password string) (*UserRow, error) {
	id := genID()
	hash := hashPassword(password)

	_, err := db.DB.Exec(
		"INSERT INTO users (id, username, password) VALUES (?, ?, ?)",
		id, username, hash,
	)
	if err != nil {
		if isUniqueViolation(err) {
			return nil, fmt.Errorf("username taken")
		}
		return nil, fmt.Errorf("create user: %w", err)
	}
	return &UserRow{ID: id, Username: username, CreatedAt: time.Now()}, nil
}

// GetUserByUsername returns a user by username.
func (db *DB) GetUserByUsername(username string) (*UserRow, error) {
	var u UserRow
	err := db.DB.QueryRow(
		"SELECT id, username, password, created_at FROM users WHERE username = ?",
		username,
	).Scan(&u.ID, &u.Username, &u.Password, &u.CreatedAt)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("get user: %w", err)
	}
	return &u, nil
}

// GetUserByID returns a user by ID.
func (db *DB) GetUserByID(id string) (*UserRow, error) {
	var u UserRow
	err := db.DB.QueryRow(
		"SELECT id, username, password, created_at FROM users WHERE id = ?",
		id,
	).Scan(&u.ID, &u.Username, &u.Password, &u.CreatedAt)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("get user: %w", err)
	}
	return &u, nil
}

// VerifyPassword checks a plaintext password against the stored hash.
func (u *UserRow) VerifyPassword(password string) bool {
	return checkPassword(password, u.Password)
}

// --- Password helpers ---

func hashPassword(password string) string {
	salt := make([]byte, 8)
	rand.Read(salt)
	hash := sha256.Sum256(append(salt, []byte(password)...))
	return hex.EncodeToString(salt) + ":" + hex.EncodeToString(hash[:])
}

func checkPassword(password, stored string) bool {
	if len(stored) < 17 { // 16 hex salt + ":"
		return false
	}
	idx := 16 // 8 bytes salt = 16 hex chars
	if stored[idx] != ':' {
		return false
	}
	salt, _ := hex.DecodeString(stored[:idx])
	expected, _ := hex.DecodeString(stored[idx+1:])
	hash := sha256.Sum256(append(salt, []byte(password)...))
	return hex.EncodeToString(hash[:]) == hex.EncodeToString(expected)
}

func genID() string {
	b := make([]byte, 16)
	rand.Read(b)
	return hex.EncodeToString(b)
}

func isUniqueViolation(err error) bool {
	if err == nil {
		return false
	}
	msg := err.Error()
	return contains(msg, "UNIQUE constraint") ||
		contains(msg, "duplicate key") ||
		contains(msg, "unique constraint")
}

func contains(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}
