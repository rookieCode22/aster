package server

import (
	"encoding/json"
	"net/http"

	"aster/internal/server/auth"
	"aster/internal/store"
)

type createSessionRequest struct {
	Title     string `json:"title"`
	AgentName string `json:"agent_name"`
}

func (s *Server) handleCreateSession(w http.ResponseWriter, r *http.Request) {
	user := auth.GetUser(r)
	if user == nil {
		writeJSON(w, 401, map[string]string{"error": "unauthorized"})
		return
	}
	var req createSessionRequest
	json.NewDecoder(r.Body).Decode(&req)
	if req.Title == "" {
		req.Title = "新会话"
	}
	if req.AgentName == "" {
		req.AgentName = "code-audit"
	}
	sess, err := s.sessions.Create(user.ID, req.Title, req.AgentName)
	if err != nil {
		writeJSON(w, 500, map[string]string{"error": err.Error()})
		return
	}
	writeJSON(w, 201, sess)
}

func (s *Server) handleListSessions(w http.ResponseWriter, r *http.Request) {
	user := auth.GetUser(r)
	if user == nil {
		writeJSON(w, 401, map[string]string{"error": "unauthorized"})
		return
	}
	sessions := s.sessions.ListByUser(user.ID)
	if sessions == nil {
		sessions = []*store.SessionRow{}
	}
	writeJSON(w, 200, map[string]any{"sessions": sessions})
}

func (s *Server) handleDeleteSession(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	if err := s.sessions.Delete(id); err != nil {
		writeJSON(w, 404, map[string]string{"error": err.Error()})
		return
	}
	writeJSON(w, 200, map[string]string{"ok": "deleted"})
}
