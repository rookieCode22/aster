// Package server — License route registration.
//
// To integrate, add this to your Server struct initialization in server.go:
//
//   import "github.com/Q16G/aster/internal/license"
//
//   verifier := license.NewVerifier(licenseSecret)
//   licHandler := api.NewLicenseHandler(store, verifier)
//   s.RegisterLicenseRoutes(licHandler)
//
// And add the agent counter if needed:
//   var agentCounter int32
//   s.mux.Handle("/ws/chat/", middleware.AgentLimit(store, &agentCounter)(wsHandler))

package server

import (
	"net/http"

	"github.com/Q16G/aster/internal/server/api"
	"github.com/Q16G/aster/internal/server/middleware"
	"github.com/Q16G/aster/internal/store"
)

// RegisterLicenseRoutes adds license endpoints to an existing mux.
// Call this during server initialization after creating the LicenseHandler.
//
// Usage in server.go:
//
//	licHandler := api.NewLicenseHandler(s.store, s.verifier)
//	RegisterLicenseRoutes(s.mux, s.store, licHandler)
func RegisterLicenseRoutes(mux *http.ServeMux, db *store.DB, handler *api.LicenseHandler) {
	// Public: no auth required for activation/status
	mux.HandleFunc("POST /api/v1/license/activate", handler.Activate)
	mux.HandleFunc("GET /api/v1/license/status", handler.Status)

	// Protected: require valid license
	protected := middleware.LicenseRequired(db)

	mux.Handle("GET /api/v1/license/history", protected(http.HandlerFunc(handler.History)))
	mux.Handle("GET /api/v1/license/check/{module}", protected(http.HandlerFunc(handler.CheckModule)))
}
