package api

import (
	"encoding/json"
	"net/http"
	"os"
	"path/filepath"
	"strings"

	"aster/internal/module"
)

type ModuleHandler struct {
	modulesDir string
}

func NewModuleHandler(modulesDir string) *ModuleHandler {
	return &ModuleHandler{modulesDir: modulesDir}
}

type moduleInfo struct {
	ID          string `json:"id"`
	Version     string `json:"version"`
	Type        string `json:"type"`
	Description string `json:"description"`
	Count       int    `json:"count"`
	Loaded      bool   `json:"loaded"`
}

func (h *ModuleHandler) HandleList(w http.ResponseWriter, r *http.Request) {
	var modules []moduleInfo

	entries, err := os.ReadDir(h.modulesDir)
	if err == nil {
		for _, entry := range entries {
			if entry.IsDir() || !strings.HasSuffix(entry.Name(), ".astm") {
				continue
			}
			modPath := filepath.Join(h.modulesDir, entry.Name())
			mod, err := module.Load(modPath)
			if err != nil {
				modules = append(modules, moduleInfo{
					ID:     strings.TrimSuffix(entry.Name(), ".astm"),
					Loaded: false,
				})
				continue
			}
			modules = append(modules, moduleInfo{
				ID:          mod.Manifest.ModuleID,
				Version:     mod.Manifest.Version,
				Type:        mod.Manifest.AssetType,
				Description: mod.Manifest.Description,
				Count:       mod.Manifest.Count,
				Loaded:      true,
			})
		}
	}

	writeJSON(w, http.StatusOK, map[string]any{"modules": modules})
}

func writeJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(v)
}

func errResp(msg string) map[string]string {
	return map[string]string{"error": msg}
}
