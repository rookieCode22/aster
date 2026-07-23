package server

import (
	"net/http"
)

// --- WebSocket Chat ---

func (s *Server) handleChatWS(w http.ResponseWriter, r *http.Request) {
	sessionID := r.PathValue("sessionId")

	// Extract user from query param or auth header (WS can't easily set Authorization header)
	tokenStr := r.URL.Query().Get("token")
	var userID string
	if tokenStr != "" {
		user, err := s.auth.VerifyToken(tokenStr)
		if err == nil && user != nil {
			userID = user.ID
		}
	}
	if userID == "" {
		// Try Authorization header too
		authHeader := r.Header.Get("Authorization")
		if len(authHeader) > 7 && authHeader[:7] == "Bearer " {
			user, err := s.auth.VerifyToken(authHeader[7:])
			if err == nil && user != nil {
				userID = user.ID
			}
		}
	}
	if userID == "" {
		writeJSON(w, http.StatusUnauthorized, map[string]string{"error": "unauthorized"})
		return
	}

	// Verify session exists
	if _, err := s.sessions.Get(sessionID); err != nil {
		writeJSON(w, http.StatusNotFound, map[string]string{"error": err.Error()})
		return
	}

	if err := s.hub.HandleUpgrade(w, r, sessionID, userID); err != nil {
		logOnce.Printf("[ws] upgrade error: %v", err)
	}
}
