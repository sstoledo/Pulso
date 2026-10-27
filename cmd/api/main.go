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

	"github.com/jackc/pgx/v5/pgxpool"

	"pulso/internal/auth"
	"pulso/internal/monitors"
	"pulso/internal/platform/config"
	"pulso/internal/platform/database"
	"pulso/internal/platform/httpx"
)

func main() {
	logger := slog.New(slog.NewJSONHandler(os.Stdout, nil))
	if err := run(logger); err != nil {
		logger.Error("API stopped", "error", err)
		os.Exit(1)
	}
}

func run(logger *slog.Logger) error {
	cfg, err := config.Load()
	if err != nil {
		return fmt.Errorf("load configuration: %w", err)
	}

	pool, err := pgxpool.New(context.Background(), cfg.DatabaseURL)
	if err != nil {
		return fmt.Errorf("configure PostgreSQL pool: %w", err)
	}
	defer pool.Close()

	authService, err := auth.NewService(pool)
	if err != nil {
		return fmt.Errorf("initialize authentication: %w", err)
	}
	router := httpx.NewRouter(database.Checker{Pool: pool}, logger)
	router.Mount("/api/v1", auth.NewHTTPHandler(authService, cfg.CookieSecure, cfg.WebOrigin, logger))
	router.Mount("/api/v1/orgs/{organizationID}", monitors.NewHTTPHandler(monitors.NewService(pool), authService, cfg.WebOrigin, logger))

	server := &http.Server{
		Addr:              cfg.APIAddress,
		Handler:           router,
		ReadHeaderTimeout: 5 * time.Second,
		IdleTimeout:       60 * time.Second,
	}

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	serverError := make(chan error, 1)
	go func() {
		logger.Info("API listening", "address", cfg.APIAddress, "environment", cfg.Environment)
		serverError <- server.ListenAndServe()
	}()

	select {
	case err := <-serverError:
		if errors.Is(err, http.ErrServerClosed) {
			return nil
		}
		return fmt.Errorf("serve HTTP: %w", err)
	case <-ctx.Done():
	}

	shutdownCtx, cancel := context.WithTimeout(context.Background(), cfg.ShutdownTimeout)
	defer cancel()
	if err := server.Shutdown(shutdownCtx); err != nil {
		return fmt.Errorf("shut down HTTP server: %w", err)
	}
	logger.Info("API stopped gracefully")
	return nil
}
