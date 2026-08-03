package server

import (
	"encoding/json"
	"net/http"
	"os"
	"path/filepath"

	"aster/internal/update"
)

// UpdateHandler exposes content-update endpoints backed by the internal/update
// package. Updates are staged on disk under stageDir and applied into modulesDir.
type UpdateHandler struct {
	stageDir   string // where new .astm packages are staged
	modulesDir string // the active modules directory
}

func NewUpdateHandler(stageDir, modulesDir string) *UpdateHandler {
	return &UpdateHandler{stageDir: stageDir, modulesDir: modulesDir}
}

// ensureStageDir lazily creates the staging directory so the "no updates yet"
// case returns an empty list rather than a 500.
func (h *UpdateHandler) ensureStageDir() error {
	if h.stageDir == "" {
		return nil
	}
	return os.MkdirAll(h.stageDir, 0o755)
}

// HandleCheck reports installed modules plus any staged updates newer than what
// is installed.
//
//	GET /api/v1/updates/check
func (h *UpdateHandler) HandleCheck(w http.ResponseWriter, r *http.Request) {
	if err := h.ensureStageDir(); err != nil {
		writeJSON(w, http.StatusInternalServerError, errResp("ensure stage dir: "+err.Error()))
		return
	}
	res, err := update.Check(h.stageDir, h.modulesDir)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, errResp("check updates: "+err.Error()))
		return
	}
	writeJSON(w, http.StatusOK, res)
}

// HandleApply installs a staged update by filename.
//
//	POST /api/v1/updates/apply  {"filename": "<name.astm>"}
func (h *UpdateHandler) HandleApply(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Filename string `json:"filename"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSON(w, http.StatusBadRequest, errResp("invalid body: "+err.Error()))
		return
	}
	if req.Filename == "" {
		writeJSON(w, http.StatusBadRequest, errResp("filename required"))
		return
	}

	// Defense in depth: reject any path traversal before handing to update.Apply.
	if filepath.Base(req.Filename) != req.Filename {
		writeJSON(w, http.StatusBadRequest, errResp("invalid filename"))
		return
	}

	info, err := update.Apply(h.stageDir, h.modulesDir, req.Filename)
	if err != nil {
		writeJSON(w, http.StatusBadRequest, errResp("apply failed: "+err.Error()))
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{
		"applied":  true,
		"module_id": info.ModuleID,
		"version":  info.Version,
		"asset_type": info.AssetType,
	})
}
