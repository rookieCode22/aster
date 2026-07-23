package api

import (
	"encoding/json"
	"net/http"
	"strings"

	"aster/internal/server/auth"
	"aster/internal/store"
)

type SkillHandler struct {
	store *store.DB
}

func NewSkillHandler(db *store.DB) *SkillHandler {
	return &SkillHandler{store: db}
}

type skillItem struct {
	Name        string   `json:"name"`
	Description string   `json:"description"`
	Instructions string  `json:"instructions,omitempty"`
	Agent       string   `json:"agent"`
	Tags        []string `json:"tags"`
	Source      string   `json:"source"`
}

type createSkillRequest struct {
	Name         string `json:"name"`
	Description  string `json:"description"`
	Instructions string `json:"instructions"`
	Agent        string `json:"agent"`
	Tags         []string `json:"tags"`
}

func (h *SkillHandler) HandleList(w http.ResponseWriter, r *http.Request) {
	user := auth.GetUser(r)
	if user == nil {
		writeJSON(w, 401, map[string]string{"error": "unauthorized"})
		return
	}

	skills, err := h.store.ListSkillsByUser(user.ID)
	if err != nil {
		writeJSON(w, 500, map[string]string{"error": err.Error()})
		return
	}

	items := make([]skillItem, 0, len(skills))
	for _, s := range skills {
		items = append(items, skillItem{
			Name:        s.Name,
			Description: s.Description,
			Agent:       s.Agent,
			Tags:        s.Tags,
			Source:      "custom",
		})
	}
	writeJSON(w, 200, map[string]any{"skills": items})
}

func (h *SkillHandler) HandleCreate(w http.ResponseWriter, r *http.Request) {
	user := auth.GetUser(r)
	if user == nil {
		writeJSON(w, 401, map[string]string{"error": "unauthorized"})
		return
	}

	var req createSkillRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSON(w, 400, map[string]string{"error": "invalid body"})
		return
	}
	req.Name = strings.TrimSpace(req.Name)
	if req.Name == "" || req.Instructions == "" {
		writeJSON(w, 400, map[string]string{"error": "name and instructions required"})
		return
	}
	if req.Agent == "" {
		req.Agent = "all"
	}

	skill, err := h.store.CreateSkill(user.ID, req.Name, req.Description, req.Instructions, req.Agent, req.Tags)
	if err != nil {
		writeJSON(w, 409, map[string]string{"error": err.Error()})
		return
	}

	writeJSON(w, 201, skillItem{
		Name:        skill.Name,
		Description: skill.Description,
		Agent:       skill.Agent,
		Tags:        skill.Tags,
		Source:      "custom",
	})
}

func (h *SkillHandler) HandleDelete(w http.ResponseWriter, r *http.Request) {
	user := auth.GetUser(r)
	if user == nil {
		writeJSON(w, 401, map[string]string{"error": "unauthorized"})
		return
	}
	name := r.PathValue("name")
	if err := h.store.DeleteSkill(user.ID, name); err != nil {
		writeJSON(w, 404, map[string]string{"error": err.Error()})
		return
	}
	writeJSON(w, 200, map[string]string{"ok": "deleted"})
}
