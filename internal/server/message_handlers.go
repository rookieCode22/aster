package server

import (
	"net/http"

	"aster/internal/server/auth"
	"aster/internal/store"
)

// handleListMessages returns all persisted messages for a session in
// chronological order. The session must belong to the authenticated user.
func (s *Server) handleListMessages(w http.ResponseWriter, r *http.Request) {
	sessionID := r.PathValue("id")
	if sessionID == "" {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "session id required"})
		return
	}

	user := auth.GetUser(r)
	if user == nil {
		writeJSON(w, http.StatusUnauthorized, map[string]string{"error": "unauthorized"})
		return
	}

	// Ownership check: the session must exist and belong to the caller.
	sess, err := s.sessions.Get(sessionID)
	if err != nil {
		writeJSON(w, http.StatusNotFound, map[string]string{"error": "session not found"})
		return
	}
	if sess.UserID != user.ID {
		writeJSON(w, http.StatusForbidden, map[string]string{"error": "forbidden"})
		return
	}

	msgs, err := s.db.ListMessagesBySession(sessionID)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
		return
	}
	if msgs == nil {
		msgs = []*store.MessageRow{}
	}

	writeJSON(w, http.StatusOK, map[string]any{"messages": msgs})
}
