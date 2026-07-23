package api

import (
	"encoding/json"
	"net/http"
	"strings"

	"aster/internal/server/auth"
)

// SkillHandler serves the skills REST API.
type SkillHandler struct{}

func NewSkillHandler() *SkillHandler {
	return &SkillHandler{}
}

type skillItem struct {
	Name        string `json:"name"`
	Description string `json:"description"`
	Source      string `json:"source"` // "custom" | "module:aster-core"
	Enabled     bool   `json:"enabled"`
}

type createSkillRequest struct {
	Name         string `json:"name"`
	Description  string `json:"description"`
	Instructions string `json:"instructions"`
	Agent        string `json:"agent"`
}

func (h *SkillHandler) HandleList(w http.ResponseWriter, r *http.Request) {
	// TODO: Phase 3 — load from DB
	writeJSON(w, http.StatusOK, map[string]any{
		"skills": []skillItem{},
	})
}

func (h *SkillHandler) HandleCreate(w http.ResponseWriter, r *http.Request) {
	user := auth.GetUser(r)
	if user == nil {
		writeJSON(w, http.StatusUnauthorized, map[string]string{"error": "unauthorized"})
		return
	}

	var req createSkillRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid body"})
		return
	}
	req.Name = strings.TrimSpace(req.Name)
	if req.Name == "" || req.Instructions == "" {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "name and instructions required"})
		return
	}

	// TODO: Phase 3 — save to DB
	_ = user

	writeJSON(w, http.StatusCreated, map[string]any{
		"ok":   true,
		"name": req.Name,
	})
}

func (h *SkillHandler) HandleDelete(w http.ResponseWriter, r *http.Request) {
	name := r.PathValue("name")
	// TODO: Phase 3 — delete from DB
	writeJSON(w, http.StatusOK, map[string]any{
		"ok":   true,
		"name": name,
	})
}
