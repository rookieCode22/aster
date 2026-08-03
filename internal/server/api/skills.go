package api

import (
	"context"
	"encoding/json"
	"net/http"
	"sort"
	"strings"

	"aster/internal/server/auth"
	"aster/internal/service"
	"aster/internal/store"
)

// builtinCatalog is the subset of *service.SkillService the handler needs.
// Declared as an interface to keep the handler decoupled and testable.
type builtinCatalog interface {
	ListSkills(ctx context.Context, filter *service.SkillFilter) ([]*service.Skill, error)
}

type SkillHandler struct {
	store   *store.DB
	builtin builtinCatalog
}

// NewSkillHandler builds a handler. builtin may be nil, in which case only
// user-created custom skills are returned.
func NewSkillHandler(db *store.DB, builtin builtinCatalog) *SkillHandler {
	return &SkillHandler{store: db, builtin: builtin}
}

type skillItem struct {
	Name         string   `json:"name"`
	Description  string   `json:"description"`
	Instructions string   `json:"instructions,omitempty"`
	Agent        string   `json:"agent"`
	Tags         []string `json:"tags"`
	Source       string   `json:"source"`   // "builtin" | "custom"
	Deletable    bool     `json:"deletable"` // false for built-in skills
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

	items := make([]skillItem, 0, 64)

	// 1) Built-in skills loaded from disk (shared across all users, read-only).
	if h.builtin != nil {
		builtins, err := h.builtin.ListSkills(r.Context(), nil)
		if err == nil {
			for _, s := range builtins {
				agent := s.Agent
				if agent == "" {
					agent = "all"
				}
				items = append(items, skillItem{
					Name:        s.Name,
					Description: s.Description,
					Agent:       agent,
					Tags:        s.Tags,
					Source:      "builtin",
					Deletable:   false,
				})
			}
		}
	}

	// 2) User-created custom skills (owned, deletable).
	skills, err := h.store.ListSkillsByUser(user.ID)
	if err != nil {
		writeJSON(w, 500, map[string]string{"error": err.Error()})
		return
	}
	for _, s := range skills {
		items = append(items, skillItem{
			Name:        s.Name,
			Description: s.Description,
			Agent:       s.Agent,
			Tags:        s.Tags,
			Source:      "custom",
			Deletable:   true,
		})
	}

	// Stable ordering: built-in first (by name), then custom (by name).
	sort.SliceStable(items, func(i, j int) bool {
		if items[i].Source != items[j].Source {
			return items[i].Source == "builtin"
		}
		return items[i].Name < items[j].Name
	})

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
		Deletable:   true,
	})
}

func (h *SkillHandler) HandleDelete(w http.ResponseWriter, r *http.Request) {
	user := auth.GetUser(r)
	if user == nil {
		writeJSON(w, 401, map[string]string{"error": "unauthorized"})
		return
	}
	name := r.PathValue("name")

	// Built-in skills are shared, disk-backed, and read-only. Reject any attempt
	// to delete one rather than silently no-op'ing against the custom table.
	if h.builtin != nil {
		if builtins, err := h.builtin.ListSkills(r.Context(), &service.SkillFilter{Name: name}); err == nil {
			for _, s := range builtins {
				if s.Name == name {
					writeJSON(w, 403, map[string]string{"error": "built-in skills cannot be deleted"})
					return
				}
			}
		}
	}

	if err := h.store.DeleteSkill(user.ID, name); err != nil {
		writeJSON(w, 404, map[string]string{"error": err.Error()})
		return
	}
	writeJSON(w, 200, map[string]string{"ok": "deleted"})
}
