package main

import (
	"context"
	"errors"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"homelink-monitor/services/api/internal/auth"
	"homelink-monitor/services/api/internal/config"
	"homelink-monitor/services/api/internal/db"
	"homelink-monitor/services/api/internal/httpapi"
	"homelink-monitor/services/api/internal/monitoring"
	"homelink-monitor/services/api/internal/store"
)

func main() {
	log := slog.New(slog.NewJSONHandler(os.Stdout, nil))
	cfg := config.Load()

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	database, err := db.Open(ctx, cfg.DBPath)
	if err != nil {
		log.Error("open database", "error", err)
		os.Exit(1)
	}
	defer database.Close()

	st := store.New(database)
	authService := auth.NewService(st)
	if err := authService.EnsureInitialAdmin(ctx, cfg.AdminUsername, cfg.AdminPassword); err != nil {
		log.Error("seed initial admin", "error", err)
		os.Exit(1)
	}
	monitor := monitoring.NewService(st, log)
	monitor.Start(ctx)

	server := &http.Server{
		Addr:              cfg.Addr,
		Handler:           httpapi.New(st, monitor, authService, log, cfg.StaticPath).Routes(),
		ReadHeaderTimeout: 5 * time.Second,
	}

	go func() {
		log.Info("server listening", "addr", cfg.Addr)
		if err := server.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			log.Error("server failed", "error", err)
			stop()
		}
	}()

	<-ctx.Done()
	shutdownCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	if err := server.Shutdown(shutdownCtx); err != nil {
		log.Error("server shutdown", "error", err)
	}
}
