package main

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/phoenix-panel/phoenix/internal/api"
	"github.com/phoenix-panel/phoenix/internal/audit"
	"github.com/phoenix-panel/phoenix/internal/config"
	"github.com/phoenix-panel/phoenix/internal/database"
	"github.com/phoenix-panel/phoenix/internal/security"
)

// version is stamped at build time via -ldflags="-X main.version=<tag>".
var version = "dev"

func main() {
	setupLogger()

	slog.Info("phoenix panel starting", "version", version)

	// --- Config ---
	cfg, err := config.Load()
	if err != nil {
		slog.Error("failed to load config", "error", err)
		os.Exit(1)
	}

	// --- Database ---
	db, err := database.Open(cfg)
	if err != nil {
		slog.Error("failed to open database", "error", err)
		os.Exit(1)
	}

	if err := database.Migrate(db); err != nil {
		slog.Error("database migration failed", "error", err)
		os.Exit(1)
	}

	if err := database.Seed(db, cfg); err != nil {
		slog.Error("database seed failed", "error", err)
		os.Exit(1)
	}

	// --- Security ---
	jwtManager := security.NewJWTManager(cfg.Security.JWTSecret, cfg.Security.JWTTTL)

	// --- Audit logger ---
	auditLogger := audit.New(db)

	// --- Router ---
	// Stamp the build version into health responses.
	api.Version = version

	deps := api.Dependencies{
		Cfg:   cfg,
		DB:    db,
		JWT:   jwtManager,
		Audit: auditLogger,
	}
	router := api.NewRouter(deps)

	// --- HTTP Server ---
	addr := fmt.Sprintf("%s:%d", cfg.Server.Host, cfg.Server.Port)
	srv := &http.Server{
		Addr:         addr,
		Handler:      router,
		ReadTimeout:  15 * time.Second,
		WriteTimeout: 30 * time.Second,
		IdleTimeout:  60 * time.Second,
	}

	// Start server in a goroutine so the main goroutine can handle signals.
	go func() {
		slog.Info("server listening", "addr", addr, "base_url", cfg.Server.BaseURL)
		if err := srv.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			slog.Error("server error", "error", err)
			os.Exit(1)
		}
	}()

	// --- Graceful shutdown ---
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	slog.Info("shutting down gracefully…")
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	if err := srv.Shutdown(ctx); err != nil {
		slog.Error("server forced to shutdown", "error", err)
	}

	// Close the DB connection pool.
	if sqlDB, err := db.DB(); err == nil {
		_ = sqlDB.Close()
	}

	slog.Info("phoenix panel stopped")
}

func setupLogger() {
	level := slog.LevelInfo
	if os.Getenv("PHOENIX_MODE") == "debug" {
		level = slog.LevelDebug
	}
	slog.SetDefault(slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{
		Level: level,
	})))
}
