package store

import "time"

type SessionRow struct {
	ID        string    `json:"id"`
	UserID    string    `json:"user_id"`
	Title     string    `json:"title"`
	AgentName string    `json:"agent_name"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

func (db *DB) CreateSession(userID, title, agentName string) (*SessionRow, error) {
	s := &SessionRow{
		ID:        genID(),
		UserID:    userID,
		Title:     title,
		AgentName: agentName,
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}
	_, err := db.DB.Exec(
		"INSERT INTO sessions (id, user_id, title, agent_name, created_at, updated_at) VALUES (?, ?, ?, ?, ?, ?)",
		s.ID, s.UserID, s.Title, s.AgentName, s.CreatedAt, s.UpdatedAt,
	)
	if err != nil {
		return nil, err
	}
	return s, nil
}

func (db *DB) GetSession(id string) (*SessionRow, error) {
	var s SessionRow
	err := db.DB.QueryRow(
		"SELECT id, user_id, title, agent_name, created_at, updated_at FROM sessions WHERE id = ?",
		id,
	).Scan(&s.ID, &s.UserID, &s.Title, &s.AgentName, &s.CreatedAt, &s.UpdatedAt)
	if err != nil {
		return nil, err
	}
	return &s, nil
}

func (db *DB) ListSessionsByUser(userID string) ([]*SessionRow, error) {
	rows, err := db.DB.Query(
		"SELECT id, user_id, title, agent_name, created_at, updated_at FROM sessions WHERE user_id = ? ORDER BY updated_at DESC",
		userID,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var sessions []*SessionRow
	for rows.Next() {
		var s SessionRow
		if err := rows.Scan(&s.ID, &s.UserID, &s.Title, &s.AgentName, &s.CreatedAt, &s.UpdatedAt); err != nil {
			return nil, err
		}
		sessions = append(sessions, &s)
	}
	return sessions, rows.Err()
}

func (db *DB) DeleteSession(id string) error {
	_, err := db.DB.Exec("DELETE FROM sessions WHERE id = ?", id)
	return err
}
