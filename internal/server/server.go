package server

import (
	"fmt"
	"net/http"
	"os"
	"path/filepath"

	"aster/internal/server/api"
	"aster/internal/server/auth"
	"aster/internal/server/middleware"
	"aster/internal/server/session"
	"aster/internal/server/ws"
)

// Config holds server configuration, sourced from environment variables.
type Config struct {
	Host        string
	Port        int
	DataDir     string // ~/.aster/
	ModulesDir  string // ~/.aster/modules/
	FrontendDir string // web/dashboard/dist/
	JWTSecret   string
}

func LoadConfig() *Config {
	home := mustHome()
	dataDir := envStr("ASTER_DATA_DIR", filepath.Join(home, ".aster"))
	return &Config{
		Host:        envStr("ASTER_HOST", "0.0.0.0"),
		Port:        envInt("ASTER_PORT", 8080),
		DataDir:     dataDir,
		ModulesDir:  filepath.Join(dataDir, "modules"),
		FrontendDir: envStr("ASTER_FRONTEND_DIR", "web/dashboard/dist"),
		JWTSecret:   envStr("ASTER_JWT_SECRET", "change-me-in-production"),
	}
}

// Server wires together all HTTP components.
type Server struct {
	cfg      *Config
	auth     *auth.Service
	sessions *session.Manager
	hub      *ws.Hub
	skills   *api.SkillHandler
	modules  *api.ModuleHandler
}

func New(cfg *Config) (*Server, error) {
	authSvc := auth.NewService(cfg.JWTSecret)
	sessMgr := session.NewManager()
	hub := ws.NewHub()
	go hub.Run()

	s := &Server{
		cfg:      cfg,
		auth:     authSvc,
		sessions: sessMgr,
		hub:      hub,
		skills:   api.NewSkillHandler(),
		modules:  api.NewModuleHandler(cfg.ModulesDir),
	}

	return s, nil
}

// Router returns the configured HTTP handler.
func (s *Server) Router() http.Handler {
	mux := http.NewServeMux()

	// Public routes
	mux.HandleFunc("GET /api/health", handleHealth)
	mux.HandleFunc("POST /api/v1/auth/login", s.auth.HandleLogin)
	mux.HandleFunc("POST /api/v1/auth/register", s.auth.HandleRegister)

	// Protected routes
	protected := middleware.Chain(
		middleware.WithCORS,
		middleware.WithLogging,
		s.auth.RequireAuth,
	)

	mux.HandleFunc("GET /api/v1/auth/me", protected(s.auth.HandleMe))

	// Sessions
	mux.HandleFunc("POST /api/v1/sessions", protected(s.handleCreateSession))
	mux.HandleFunc("GET /api/v1/sessions", protected(s.handleListSessions))
	mux.HandleFunc("DELETE /api/v1/sessions/{id}", protected(s.handleDeleteSession))

	// Chat WebSocket
	mux.HandleFunc("GET /ws/chat/{sessionId}", s.handleChatWS)

	// Skills
	mux.HandleFunc("GET /api/v1/skills", protected(s.skills.HandleList))
	mux.HandleFunc("POST /api/v1/skills/custom", protected(s.skills.HandleCreate))
	mux.HandleFunc("DELETE /api/v1/skills/custom/{name}", protected(s.skills.HandleDelete))

	// Modules
	mux.HandleFunc("GET /api/v1/modules", protected(s.modules.HandleList))

	// Static frontend (SPA fallback)
	if s.cfg.FrontendDir != "" {
		if info, err := os.Stat(s.cfg.FrontendDir); err == nil && info.IsDir() {
			fs := http.FileServer(http.Dir(s.cfg.FrontendDir))
			mux.Handle("GET /", fs)
			logOnce.Printf("serving frontend from %s", s.cfg.FrontendDir)
		}
	}

	return mux
}

func (s *Server) Close() {
	s.hub.Stop()
}

func handleHealth(w http.ResponseWriter, r *http.Request) {
	fmt.Fprintf(w, `{"ok":true,"service":"aster-server"}`)
}

func mustHome() string {
	h, _ := os.UserHomeDir()
	return h
}
