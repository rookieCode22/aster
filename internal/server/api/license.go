package api

import (
	"encoding/json"
	"io"
	"net/http"
	"time"

	"aster/internal/license"
	"aster/internal/store"
)

// LicenseHandler handles license activation, status, and history.
type LicenseHandler struct {
	store    *store.DB
	verifier *license.Verifier
}

func NewLicenseHandler(db *store.DB, verifier *license.Verifier) *LicenseHandler {
	return &LicenseHandler{store: db, verifier: verifier}
}

// Activate handles POST /api/v1/license/activate
func (h *LicenseHandler) Activate(w http.ResponseWriter, r *http.Request) {
	body, err := io.ReadAll(io.LimitReader(r.Body, 64*1024))
	if err != nil {
		writeJSON(w, http.StatusBadRequest, errResp("failed to read body"))
		return
	}
	defer r.Body.Close()

	lic, err := h.verifier.Verify(body)
	if err != nil {
		writeJSON(w, http.StatusBadRequest, errResp("invalid license: "+err.Error()))
		return
	}
	if lic.IsExpired() {
		writeJSON(w, http.StatusBadRequest, errResp("license has expired"))
		return
	}

	issuedAt, _ := time.Parse(time.RFC3339, lic.IssuedAt)
	var expiresAt *time.Time
	if lic.ExpiresAt != "" {
		t, err := time.Parse(time.RFC3339, lic.ExpiresAt)
		if err == nil {
			expiresAt = &t
		}
	}

	modJSON, _ := json.Marshal(lic.Modules)

	rec, err := h.store.ActivateLicense(
		string(body), lic.Customer, lic.Email,
		string(lic.Plan), string(modJSON), lic.HWID,
		issuedAt, expiresAt, lic.MaxAgents,
	)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, errResp("activation failed: "+err.Error()))
		return
	}

	writeJSON(w, http.StatusOK, map[string]interface{}{
		"activated": true,
		"license":   rec,
	})
}

// Status handles GET /api/v1/license/status
func (h *LicenseHandler) Status(w http.ResponseWriter, r *http.Request) {
	st, err := h.store.GetLicenseStatus()
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, errResp(err.Error()))
		return
	}
	writeJSON(w, http.StatusOK, st)
}

// History handles GET /api/v1/license/history
func (h *LicenseHandler) History(w http.ResponseWriter, r *http.Request) {
	list, err := h.store.ListLicenses()
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, errResp(err.Error()))
		return
	}
	writeJSON(w, http.StatusOK, list)
}

// CheckModule handles GET /api/v1/license/check/{module}
func (h *LicenseHandler) CheckModule(w http.ResponseWriter, r *http.Request) {
	moduleName := r.PathValue("module")
	if moduleName == "" {
		writeJSON(w, http.StatusBadRequest, errResp("module name required"))
		return
	}

	rec, err := h.store.GetActiveLicense()
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, errResp(err.Error()))
		return
	}

	authorized := false
	if rec != nil {
		var modules []string
		if json.Unmarshal([]byte(rec.Modules), &modules) == nil {
			for _, m := range modules {
				if m == moduleName || m == "*" {
					authorized = true
					break
				}
			}
		}
	}

	writeJSON(w, http.StatusOK, map[string]interface{}{
		"module":     moduleName,
		"authorized": authorized,
	})
}
