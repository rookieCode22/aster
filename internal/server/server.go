package server

import (
	"fmt"
	"net/http"
	"os"
	"path/filepath"

	"context"

	"aster/internal/license"
	"aster/internal/server/api"
	"aster/internal/server/auth"
	"aster/internal/server/middleware"
	"aster/internal/server/provider"
	"aster/internal/server/session"
	"aster/internal/server/ws"
	"aster/internal/service"
	"aster/internal/store"
)

type Config struct {
	Host        string
	Port        int
	DataDir     string
	ModulesDir  string
	UpdateDir   string
	FrontendDir string
	JWTSecret   string
	DatabaseURL string
}

func LoadConfig() *Config {
	dataDir := envStr("ASTER_DATA_DIR", defaultDataDir())
	return &Config{
		Host:        envStr("ASTER_HOST", "0.0.0.0"),
		Port:        envInt("ASTER_PORT", 8080),
		DataDir:     dataDir,
		ModulesDir:  filepath.Join(dataDir, "modules"),
		UpdateDir:   envStr("ASTER_UPDATE_DIR", filepath.Join(dataDir, "updates")),
		FrontendDir: envStr("ASTER_FRONTEND_DIR", "web/dist"),
		JWTSecret:   envStr("ASTER_JWT_SECRET", "change-me-in-production"),
		DatabaseURL: envStr("ASTER_DATABASE_URL", ""),
	}
}

type Server struct {
	cfg          *Config
	db           *store.DB
	auth         *auth.Service
	sessions     *session.Manager
	hub          *ws.Hub
	skills       *api.SkillHandler
	modules      *api.ModuleHandler
	updates      *UpdateHandler
	verifier     *license.Verifier
	skillService *service.SkillService
	runner       *AgentRunner
}

func New(cfg *Config) (*Server, error) {
	dsn := cfg.DatabaseURL
	if dsn == "" {
		dsn = "sqlite://" + filepath.Join(cfg.DataDir, "aster.db")
	}

	db, err := store.Open(dsn)
	if err != nil {
		return nil, fmt.Errorf("database: %w", err)
	}

	authSvc := auth.NewService(cfg.JWTSecret, db)
	sessMgr := session.NewManager(db)
	hub := ws.NewHub()
	go hub.Run()

	// Load built-in skills from disk into an in-memory catalog. Non-fatal: an
	// empty catalog just means the agent starts with no preloaded skills.
	skillService := service.NewSkillServiceWithMemory()
	if _, err := skillService.ImportSkillsFromMultipleSources(context.Background(), cfg.DataDir, defaultDataDir()); err != nil {
		logOnce.Printf("import skills: %v (continuing with empty catalog)", err)
	}

	s := &Server{
		cfg:          cfg,
		db:           db,
		auth:         authSvc,
		sessions:     sessMgr,
		hub:          hub,
		skills:       api.NewSkillHandler(db, skillService),
		modules:      api.NewModuleHandler(cfg.ModulesDir),
		updates:      NewUpdateHandler(cfg.UpdateDir, cfg.ModulesDir),
		verifier:     license.NewVerifier([]byte(licenseSecret(cfg))),
		skillService: skillService,
	}

	// Assemble the agent engine if a model provider is configured. Chat is only
	// available when this succeeds; otherwise the server runs in a degraded mode
	// where the UI works but chat returns a clear "not configured" error.
	if provCfg, err := provider.LoadFromEnv(); err != nil {
		logOnce.Printf("model provider not configured: %v (chat disabled until configured)", err)
	} else {
		runner, err := NewAgentRunner(hub, db, skillService, provCfg)
		if err != nil {
			logOnce.Printf("agent runner init failed: %v (chat disabled)", err)
		} else {
			s.runner = runner
			hub.SetChatHandler(runner.HandleChat)
			logOnce.Printf("agent engine ready (provider=%s model=%s)", provCfg.Preset, provCfg.Model)
		}
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
	mux.HandleFunc("GET /api/v1/sessions/{id}/messages", protected(s.handleListMessages))
	mux.HandleFunc("GET /api/v1/sessions/{id}/report", protected(s.handleExportReport))
	mux.HandleFunc("GET /ws/chat/{sessionId}", s.handleChatWS)
	mux.HandleFunc("GET /api/v1/skills", protected(s.skills.HandleList))
	mux.HandleFunc("POST /api/v1/skills/custom", protected(s.skills.HandleCreate))
	mux.HandleFunc("DELETE /api/v1/skills/custom/{name}", protected(s.skills.HandleDelete))
	mux.HandleFunc("GET /api/v1/modules", protected(s.modules.HandleList))
	mux.HandleFunc("GET /api/v1/updates/check", protected(s.updates.HandleCheck))
	mux.HandleFunc("POST /api/v1/updates/apply", protected(s.updates.HandleApply))

	// License routes
	licHandler := api.NewLicenseHandler(s.db, s.verifier)
	RegisterLicenseRoutes(mux, s.db, licHandler)

	if s.cfg.FrontendDir != "" {
		frontendDir := s.cfg.FrontendDir
		// Resolve a relative frontend dir robustly: try it as-is (cwd-relative),
		// then relative to the executable so the server works regardless of the
		// working directory it was launched from.
		if !filepath.IsAbs(frontendDir) {
			candidates := []string{frontendDir}
			if exe, err := os.Executable(); err == nil {
				candidates = append(candidates, filepath.Join(filepath.Dir(exe), frontendDir))
			}
			for _, c := range candidates {
				if info, err := os.Stat(c); err == nil && info.IsDir() {
					frontendDir = c
					break
				}
			}
		}

		if info, err := os.Stat(frontendDir); err == nil && info.IsDir() {
			mux.Handle("GET /", newSPAHandler(frontendDir, "index.html"))
			logOnce.Printf("serving frontend from %s", frontendDir)
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

func defaultDataDir() string {
	if d := os.Getenv("ASTER_DATA_DIR"); d != "" {
		return d
	}
	home, err := os.UserHomeDir()
	if err == nil && home != "" {
		return filepath.Clean(filepath.Join(home, ".aster"))
	}
	return ".aster"
}

func mustHome() string { return defaultDataDir() }

func licenseSecret(cfg *Config) string {
	if s := os.Getenv("ASTER_LICENSE_SECRET"); s != "" {
		return s
	}
	return "dev-license-secret-change-me"
}
