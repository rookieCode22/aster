package store

import "time"

type LicenseRow struct {
	ID        string     `json:"id"`
	UserID    string     `json:"user_id"`
	ModuleID  string     `json:"module_id"`
	ExpiresAt *time.Time `json:"expires_at,omitempty"` // nil = permanent
	CreatedAt time.Time  `json:"created_at"`
}

// GrantLicense assigns a module license to a user.
func (db *DB) GrantLicense(userID, moduleID string, expiresAt *time.Time) (*LicenseRow, error) {
	l := &LicenseRow{
		ID:        genID(),
		UserID:    userID,
		ModuleID:  moduleID,
		ExpiresAt: expiresAt,
		CreatedAt: time.Now(),
	}
	_, err := db.DB.Exec(
		"INSERT INTO licenses (id, user_id, module_id, expires_at, created_at) VALUES (?, ?, ?, ?, ?)",
		l.ID, l.UserID, l.ModuleID, l.ExpiresAt, l.CreatedAt,
	)
	if err != nil {
		return nil, err
	}
	return l, nil
}

// HasLicense checks if a user has a valid (non-expired) license for a module.
func (db *DB) HasLicense(userID, moduleID string) (bool, error) {
	var count int
	err := db.DB.QueryRow(
		"SELECT COUNT(*) FROM licenses WHERE user_id = ? AND module_id = ? AND (expires_at IS NULL OR expires_at > ?)",
		userID, moduleID, time.Now(),
	).Scan(&count)
	return count > 0, err
}

// ListLicenses returns all active licenses for a user.
func (db *DB) ListLicenses(userID string) ([]*LicenseRow, error) {
	rows, err := db.DB.Query(
		"SELECT id, user_id, module_id, expires_at, created_at FROM licenses WHERE user_id = ? AND (expires_at IS NULL OR expires_at > ?)",
		userID, time.Now(),
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var licenses []*LicenseRow
	for rows.Next() {
		var l LicenseRow
		if err := rows.Scan(&l.ID, &l.UserID, &l.ModuleID, &l.ExpiresAt, &l.CreatedAt); err != nil {
			return nil, err
		}
		licenses = append(licenses, &l)
	}
	return licenses, rows.Err()
}
