// server is the web API entry point for Aster.
// It serves the REST API, WebSocket streaming, and the React dashboard.
package main

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"aster/internal/server"
)

func main() {
	cfg := server.LoadConfig()

	// Ensure data directories
	for _, dir := range []string{cfg.DataDir, cfg.ModulesDir} {
		if err := os.MkdirAll(dir, 0o755); err != nil {
			log.Fatalf("mkdir %s: %v", dir, err)
		}
	}

	srv, err := server.New(cfg)
	if err != nil {
		log.Fatalf("init server: %v", err)
	}

	addr := fmt.Sprintf("%s:%d", cfg.Host, cfg.Port)
	httpServer := &http.Server{
		Addr:         addr,
		Handler:      srv.Router(),
		ReadTimeout:  30 * time.Second,
		WriteTimeout: 10 * time.Minute, // long for streaming
		IdleTimeout:  120 * time.Second,
	}

	// Graceful shutdown
	go func() {
		sigCh := make(chan os.Signal, 1)
		signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)
		<-sigCh
		log.Println("shutting down...")
		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()
		httpServer.Shutdown(ctx)
		srv.Close()
	}()

	log.Printf("Aster server starting on http://%s", addr)
	log.Printf("  data dir:    %s", cfg.DataDir)
	log.Printf("  modules dir: %s", cfg.ModulesDir)
	log.Printf("  frontend:     %s", cfg.FrontendDir)

	if err := httpServer.ListenAndServe(); err != http.ErrServerClosed {
		log.Fatalf("listen: %v", err)
	}
	log.Println("server stopped")
}
