package main

import (
	"context"
	"encoding/json"
	"errors"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"sync/atomic"
	"syscall"
	"time"
)

func main() {
	logger := slog.New(slog.NewJSONHandler(os.Stdout, nil))
	address := os.Getenv("DEMO_TARGET_ADDR")
	if address == "" {
		address = ":8080"
	}
	server := &http.Server{Addr: address, Handler: newDemoHandler(), ReadHeaderTimeout: 5 * time.Second}
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()
	serverError := make(chan error, 1)
	go func() {
		logger.Info("demo target listening", "address", address)
		serverError <- server.ListenAndServe()
	}()
	select {
	case err := <-serverError:
		if err != nil && !errors.Is(err, http.ErrServerClosed) {
			logger.Error("demo target failed", "error", err)
			os.Exit(1)
		}
	case <-ctx.Done():
	}
	shutdownCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	if err := server.Shutdown(shutdownCtx); err != nil {
		logger.Error("demo target shutdown failed", "error", err)
		os.Exit(1)
	}
}

func newDemoHandler() http.Handler {
	var healthy atomic.Bool
	healthy.Store(true)
	mux := http.NewServeMux()
	mux.HandleFunc("/healthz", func(w http.ResponseWriter, _ *http.Request) {
		writeDemoJSON(w, http.StatusOK, map[string]string{"status": "ok"})
	})
	mux.HandleFunc("/health", func(w http.ResponseWriter, _ *http.Request) {
		if !healthy.Load() {
			writeDemoJSON(w, http.StatusInternalServerError, map[string]string{"status": "unavailable"})
			return
		}
		writeDemoJSON(w, http.StatusOK, map[string]string{"status": "ok"})
	})
	mux.HandleFunc("/control", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			w.WriteHeader(http.StatusMethodNotAllowed)
			return
		}
		var input struct {
			Healthy *bool `json:"healthy"`
		}
		if err := json.NewDecoder(http.MaxBytesReader(w, r.Body, 1024)).Decode(&input); err != nil || input.Healthy == nil {
			writeDemoJSON(w, http.StatusBadRequest, map[string]string{"error": "healthy boolean required"})
			return
		}
		healthy.Store(*input.Healthy)
		writeDemoJSON(w, http.StatusOK, map[string]bool{"healthy": healthy.Load()})
	})
	return mux
}

func writeDemoJSON(w http.ResponseWriter, status int, body any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(body)
}
