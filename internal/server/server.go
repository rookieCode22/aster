package server

import (
	"fmt"
	"net/http"
	"os"
	"path/filepath"

	"aster/internal/license"
	"aster/internal/server/api"
	"aster/internal/server/auth"
	"aster/internal/server/middleware"
	"aster/internal/server/session"
	"aster/internal/server/ws"
	"aster/internal/store"
)

type Config struct {
	Host        string
	Port        int
	DataDir     string
	ModulesDir  string
	FrontendDir string
	JWTSecret   string
	DatabaseURL string
}

func LoadConfig() *Config {
	home := mustHome()
	dataDir := envStr("ASTER_DATA_DIR", filepath.Join(home, ".aster"))
	return &Config{
		Host:        envStr("ASTER_HOST", "0.0.0.0"),
		Port:        envInt("ASTER_PORT", 8080),
		DataDir:     dataDir,
		ModulesDir:  filepath.Join(dataDir, "modules"),
		FrontendDir: envStr("ASTER_FRONTEND_DIR", "web/dist"),
		JWTSecret:   envStr("ASTER_JWT_SECRET", "change-me-in-production"),
		DatabaseURL: envStr("ASTER_DATABASE_URL", ""),
	}
}

type Server struct {
	cfg      *Config
	db       *store.DB
	auth     *auth.Service
	sessions *session.Manager
	hub      *ws.Hub
	skills   *api.SkillHandler
	modules  *api.ModuleHandler
	verifier *license.Verifier
}

func New(cfg *Config) (*Server, error) {
	dsn := cfg.DatabaseURL
	if dsn == "" {
		dsn = "sqlite:///" + filepath.Join(cfg.DataDir, "aster.db")
	}

	db, err := store.Open(dsn)
	if err != nil {
		return nil, fmt.Errorf("database: %w", err)
	}

	authSvc := auth.NewService(cfg.JWTSecret, db)
	sessMgr := session.NewManager(db)
	hub := ws.NewHub()
	go hub.Run()

	s := &Server{
		cfg:      cfg,
		db:       db,
		auth:     authSvc,
		sessions: sessMgr,
		hub:      hub,
		skills:   api.NewSkillHandler(db),
		modules:  api.NewModuleHandler(cfg.ModulesDir),
		verifier: license.NewVerifier([]byte(licenseSecret(cfg))),
	}
	return s, nil
}

func (s *Server) Router() http.Handler {
	mux := http.NewServeMux()

	mux.HandleFunc("GET /api/health", handleHealth)
	mux.HandleFunc("POST /api/v1/auth/login", s.auth.HandleLogin)
	mux.HandleFunc("POST /api/v1/auth/register", s.auth.HandleRegister)

	protected := middleware.Chain(
		middleware.WithCORS,
		middleware.WithLogging,
		s.auth.RequireAuth,
	)

	mux.HandleFunc("GET /api/v1/auth/me", protected(s.auth.HandleMe))
	mux.HandleFunc("POST /api/v1/sessions", protected(s.handleCreateSession))
	mux.HandleFunc("GET /api/v1/sessions", protected(s.handleListSessions))
	mux.HandleFunc("DELETE /api/v1/sessions/{id}", protected(s.handleDeleteSession))
	mux.HandleFunc("GET /ws/chat/{sessionId}", s.handleChatWS)
	mux.HandleFunc("GET /api/v1/skills", protected(s.skills.HandleList))
	mux.HandleFunc("POST /api/v1/skills/custom", protected(s.skills.HandleCreate))
	mux.HandleFunc("DELETE /api/v1/skills/custom/{name}", protected(s.skills.HandleDelete))
	mux.HandleFunc("GET /api/v1/modules", protected(s.modules.HandleList))

	// License routes
	licHandler := api.NewLicenseHandler(s.db, s.verifier)
	RegisterLicenseRoutes(mux, s.db, licHandler)

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
	s.db.Close()
}

func handleHealth(w http.ResponseWriter, r *http.Request) {
	fmt.Fprintf(w, `{"ok":true,"service":"aster-server"}`)
}

func mustHome() string {
	h, _ := os.UserHomeDir()
	if len(h) > 2 && h[0] == '\\' && h[2] == ':' {
		// Fix Windows paths like "\C:\Users\yummy" → "C:\Users\yummy"
		h = h[1:]
	}
	return h
}

func licenseSecret(cfg *Config) string {
	if s := os.Getenv("ASTER_LICENSE_SECRET"); s != "" {
		return s
	}
	return "dev-license-secret-change-me"
}
