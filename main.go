package main

import (
	"context"
	"embed"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"layerinvoice/internal/app"
	"layerinvoice/internal/auth"
	"layerinvoice/internal/config"
	"layerinvoice/internal/db"
	"layerinvoice/internal/jobs"
	"layerinvoice/internal/handlers"
)

//go:embed templates
var templatesFS embed.FS

//go:embed migrations
var migrationsFS embed.FS

//go:embed static
var staticFS embed.FS

func main() {
	// 1. Initialize Structured Logging
	logger := slog.New(slog.NewJSONHandler(os.Stdout, nil))
	slog.SetDefault(logger)

	slog.Info("Starting LayerInvoice server...")

	// 2. Load Configuration
	cfg, err := config.Load()
	if err != nil {
		slog.Error("Failed to load configuration", "error", err)
		os.Exit(1)
	}

	// 3. Open and Configure SQLite Database
	if cfg.AppEnv == "development" {
		slog.Info("Development environment detected. Resetting database to a clean state...")
		_ = os.Remove(cfg.DBPath)
		_ = os.Remove(cfg.DBPath + "-wal")
		_ = os.Remove(cfg.DBPath + "-shm")
	}

	database, err := db.Open(cfg.DBPath)
	if err != nil {
		slog.Error("Failed to open SQLite database", "error", err)
		os.Exit(1)
	}
	defer database.Close()
	slog.Info("SQLite database connected successfully", "db_path", cfg.DBPath)

	// 4. Run SQLite Migrations
	slog.Info("Running database migrations...")
	if err := database.RunMigrations(migrationsFS); err != nil {
		slog.Error("Database migrations failed", "error", err)
		os.Exit(1)
	}
	slog.Info("Database migrations completed successfully")

	// 5. Initialize Services & Shared Context
	sessionStore := auth.NewSessionStore(database.DB)
	appCtx := app.New(cfg, database, sessionStore, templatesFS)

	// 6. Launch Overdue Jobs Background Scheduler
	slog.Info("Launching background scheduler...")
	scheduler, err := jobs.NewScheduler(database.DB)
	if err != nil {
		slog.Error("Failed to initialize scheduler", "error", err)
		os.Exit(1)
	}
	if err := scheduler.Start(); err != nil {
		slog.Error("Failed to start scheduler", "error", err)
		os.Exit(1)
	}
	defer func() {
		_ = scheduler.Stop()
		slog.Info("Background scheduler stopped")
	}()

	// 7. Wire HTTP Router
	router := handlers.WireRoutes(appCtx, staticFS)

	serverAddr := fmt.Sprintf(":%s", cfg.Port)
	srv := &http.Server{
		Addr:         serverAddr,
		Handler:      router,
		ReadTimeout:  15 * time.Second,
		WriteTimeout: 15 * time.Second,
		IdleTimeout:  60 * time.Second,
	}

	// 8. Handle Graceful Shutdown
	go func() {
		slog.Info("LayerInvoice is running", "address", fmt.Sprintf("http://localhost%s", serverAddr))
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			slog.Error("HTTP server failure", "error", err)
			os.Exit(1)
		}
	}()

	// Listen for OS signals
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	slog.Info("Shutting down server gracefully...")
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	if err := srv.Shutdown(ctx); err != nil {
		slog.Error("Server forced to shutdown", "error", err)
	}

	slog.Info("Server exited gracefully. Goodbye!")
}
