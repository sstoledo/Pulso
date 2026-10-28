package main

import (
	"context"
	"fmt"
	"log/slog"
	"os"
	"os/signal"
	"syscall"

	"github.com/jackc/pgx/v5/pgxpool"

	"pulso/internal/checks"
	"pulso/internal/notifications"
	"pulso/internal/platform/config"
)

func main() {
	logger := slog.New(slog.NewJSONHandler(os.Stdout, nil))
	if err := run(logger); err != nil {
		logger.Error("worker stopped", "error", err)
		os.Exit(1)
	}
}

func run(logger *slog.Logger) error {
	cfg, err := config.Load()
	if err != nil {
		return fmt.Errorf("load configuration: %w", err)
	}

	startupCtx, cancel := context.WithTimeout(context.Background(), cfg.ShutdownTimeout)
	defer cancel()
	pool, err := pgxpool.New(startupCtx, cfg.DatabaseURL)
	if err != nil {
		return fmt.Errorf("configure PostgreSQL pool: %w", err)
	}
	defer pool.Close()
	if err := pool.Ping(startupCtx); err != nil {
		return fmt.Errorf("connect to PostgreSQL: %w", err)
	}

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()
	worker := checks.NewWorker(pool, checks.NewExecutor(cfg.CheckAllowPrivateHosts), logger)
	worker.Concurrency = cfg.WorkerConcurrency
	worker.BatchSize = int32(cfg.WorkerBatchSize)
	worker.PollInterval = cfg.WorkerPollInterval
	notificationWorker := notifications.NewWorker(pool, notifications.NewSMTPMailer(notifications.SMTPConfig{
		Host: cfg.SMTPHost, Port: cfg.SMTPPort, From: cfg.SMTPFrom,
	}), logger)
	notificationWorker.PollInterval = cfg.WorkerPollInterval
	workerCtx, cancelWorkers := context.WithCancel(ctx)
	defer cancelWorkers()
	workerErrors := make(chan error, 2)
	go func() { workerErrors <- worker.Run(workerCtx) }()
	go func() { workerErrors <- notificationWorker.Run(workerCtx) }()
	logger.Info("worker started", "environment", cfg.Environment, "concurrency", worker.Concurrency, "batch_size", worker.BatchSize)
	for completed := 0; completed < 2; completed++ {
		if err := <-workerErrors; err != nil {
			cancelWorkers()
			return fmt.Errorf("run worker: %w", err)
		}
	}
	logger.Info("worker stopped gracefully")
	return nil
}
