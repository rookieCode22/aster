package store

import "time"

// MessageRow is a single chat message belonging to a session.
type MessageRow struct {
	ID        string    `json:"id"`
	SessionID string    `json:"session_id"`
	Role      string    `json:"role"` // "user" | "assistant" | "system"
	Content   string    `json:"content"`
	CreatedAt time.Time `json:"created_at"`
}

// CreateMessage inserts a message and returns the stored row.
func (db *DB) CreateMessage(sessionID, role, content string) (*MessageRow, error) {
	m := &MessageRow{
		ID:        genID(),
		SessionID: sessionID,
		Role:      role,
		Content:   content,
		CreatedAt: time.Now(),
	}
	_, err := db.DB.Exec(
		"INSERT INTO messages (id, session_id, role, content, created_at) VALUES (?, ?, ?, ?, ?)",
		m.ID, m.SessionID, m.Role, m.Content, m.CreatedAt,
	)
	if err != nil {
		return nil, err
	}
	// Touch the session's updated_at so recent conversations sort first.
	_, _ = db.DB.Exec("UPDATE sessions SET updated_at = ? WHERE id = ?", m.CreatedAt, sessionID)
	return m, nil
}

// ListMessagesBySession returns all messages for a session in chronological order.
func (db *DB) ListMessagesBySession(sessionID string) ([]*MessageRow, error) {
	rows, err := db.DB.Query(
		"SELECT id, session_id, role, content, created_at FROM messages WHERE session_id = ? ORDER BY created_at ASC",
		sessionID,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var msgs []*MessageRow
	for rows.Next() {
		var m MessageRow
		if err := rows.Scan(&m.ID, &m.SessionID, &m.Role, &m.Content, &m.CreatedAt); err != nil {
			return nil, err
		}
		msgs = append(msgs, &m)
	}
	return msgs, rows.Err()
}
