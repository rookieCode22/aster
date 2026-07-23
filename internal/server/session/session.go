package session

import (
	"aster/internal/store"
)

// Manager wraps the database-backed session storage.
type Manager struct {
	store *store.DB
}

func NewManager(db *store.DB) *Manager {
	return &Manager{store: db}
}

func (m *Manager) Create(userID, title, agentName string) (*store.SessionRow, error) {
	return m.store.CreateSession(userID, title, agentName)
}

func (m *Manager) Get(id string) (*store.SessionRow, error) {
	return m.store.GetSession(id)
}

func (m *Manager) ListByUser(userID string) []*store.SessionRow {
	sessions, _ := m.store.ListSessionsByUser(userID)
	return sessions
}

func (m *Manager) Delete(id string) error {
	return m.store.DeleteSession(id)
}
