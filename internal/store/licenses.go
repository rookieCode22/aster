package store

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"time"
)

// LicenseRecord is the database representation of an activated license.
type LicenseRecord struct {
	ID          string    `json:"id"`
	Customer    string    `json:"customer"`
	Email       string    `json:"email"`
	Plan        string    `json:"plan"`
	Modules     string    `json:"modules"` // JSON array stored as text
	IssuedAt    time.Time `json:"issued_at"`
	ExpiresAt   *time.Time `json:"expires_at,omitempty"`
	MaxAgents   int       `json:"max_agents"`
	HWID        string    `json:"hwid,omitempty"`
	FullJSON    string    `json:"full_json"` // the original signed license JSON
	ActivatedAt time.Time `json:"activated_at"`
	IsActive    bool      `json:"is_active"`
}

// ─── License Store ───

// ActivateLicense stores a new license and deactivates any previous ones.
func (db *DB) ActivateLicense(fullJSON string, cust, email, plan, modules, hwid string, issuedAt time.Time, expiresAt *time.Time, maxAgents int) (*LicenseRecord, error) {
	// Deactivate all existing licenses
	if _, err := db.Exec(`UPDATE licenses SET is_active = false`); err != nil {
		return nil, fmt.Errorf("deactivate licenses: %w", err)
	}

	rec := &LicenseRecord{
		ID:          genID(),
		Customer:    cust,
		Email:       email,
		Plan:        plan,
		Modules:     modules,
		IssuedAt:    issuedAt,
		ExpiresAt:   expiresAt,
		MaxAgents:   maxAgents,
		HWID:        hwid,
		FullJSON:    fullJSON,
		ActivatedAt: time.Now().UTC(),
		IsActive:    true,
	}

	_, err := db.DB.Exec(
		`INSERT INTO licenses (id, customer, email, plan, modules, issued_at, expires_at, max_agents, hwid, full_json, activated_at, is_active)
		 VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		rec.ID, rec.Customer, rec.Email, rec.Plan, rec.Modules,
		rec.IssuedAt, rec.ExpiresAt, rec.MaxAgents, rec.HWID, rec.FullJSON,
		rec.ActivatedAt, rec.IsActive,
	)

	if err != nil {
		return nil, fmt.Errorf("insert license: %w", err)
	}
	return rec, nil
}

// GetActiveLicense returns the currently active license, or nil if none.
func (db *DB) GetActiveLicense() (*LicenseRecord, error) {
	rec := &LicenseRecord{}
	var expiresAt sql.NullTime
	err := db.QueryRow(
		`SELECT id, customer, email, plan, modules, issued_at, expires_at, max_agents, hwid, full_json, activated_at, is_active
		 FROM licenses WHERE is_active = true LIMIT 1`,
	).Scan(&rec.ID, &rec.Customer, &rec.Email, &rec.Plan, &rec.Modules,
		&rec.IssuedAt, &expiresAt, &rec.MaxAgents, &rec.HWID, &rec.FullJSON,
		&rec.ActivatedAt, &rec.IsActive)

	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("get active license: %w", err)
	}
	if expiresAt.Valid {
		rec.ExpiresAt = &expiresAt.Time
	}
	return rec, nil
}

// ListLicenses returns all licenses ordered by activation date descending.
func (db *DB) ListLicenses() ([]LicenseRecord, error) {
	rows, err := db.Query(
		`SELECT id, customer, email, plan, modules, issued_at, expires_at, max_agents, hwid, activated_at, is_active
		 FROM licenses ORDER BY activated_at DESC`,
	)
	if err != nil {
		return nil, fmt.Errorf("list licenses: %w", err)
	}
	defer rows.Close()

	var list []LicenseRecord
	for rows.Next() {
		var r LicenseRecord
		var expiresAt sql.NullTime
		if err := rows.Scan(&r.ID, &r.Customer, &r.Email, &r.Plan, &r.Modules,
			&r.IssuedAt, &expiresAt, &r.MaxAgents, &r.HWID, &r.ActivatedAt, &r.IsActive); err != nil {
			return nil, fmt.Errorf("scan license: %w", err)
		}
		if expiresAt.Valid {
			r.ExpiresAt = &expiresAt.Time
		}
		list = append(list, r)
	}
	return list, rows.Err()
}

// LicenseStatus returns a summary of the current license state.
type LicenseStatus struct {
	Active        bool     `json:"active"`
	Customer      string   `json:"customer,omitempty"`
	Plan          string   `json:"plan,omitempty"`
	Modules       []string `json:"modules,omitempty"`
	DaysRemaining int      `json:"days_remaining"` // -1 = unlimited, 0 = expired
	MaxAgents     int      `json:"max_agents"`
	ExpiresAt     string   `json:"expires_at,omitempty"`
}

// GetLicenseStatus returns the current license status for API responses.
func (db *DB) GetLicenseStatus() (*LicenseStatus, error) {
	rec, err := db.GetActiveLicense()
	if err != nil {
		return nil, err
	}
	if rec == nil {
		return &LicenseStatus{Active: false}, nil
	}

	var modules []string
	json.Unmarshal([]byte(rec.Modules), &modules)

	daysRemaining := -1
	if rec.ExpiresAt != nil {
		days := int(time.Until(*rec.ExpiresAt).Hours() / 24)
		if days < 0 {
			days = 0
		}
		daysRemaining = days
	}

	expStr := ""
	if rec.ExpiresAt != nil {
		expStr = rec.ExpiresAt.Format(time.RFC3339)
	}

	return &LicenseStatus{
		Active:        true,
		Customer:      rec.Customer,
		Plan:          rec.Plan,
		Modules:       modules,
		DaysRemaining: daysRemaining,
		MaxAgents:     rec.MaxAgents,
		ExpiresAt:     expStr,
	}, nil
}
