// Command api runs the Fajr LMS HTTP API.
package main

import (
	"context"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/ebnsina/fajr-lms/internal/config"
	"github.com/ebnsina/fajr-lms/internal/httpx"
)

func main() {
	if err := run(); err != nil {
		slog.Error("fatal", "error", err)
		os.Exit(1)
	}
}

func run() error {
	cfg, err := config.Load()
	if err != nil {
		return err
	}

	slog.SetDefault(slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{Level: cfg.LogLevel})))

	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	mux := http.NewServeMux()
	mux.Handle("GET /healthz", httpx.Handler(func(w http.ResponseWriter, r *http.Request) error {
		return httpx.JSON(w, http.StatusOK, map[string]string{"status": "ok"})
	}))

	srv := &http.Server{
		Addr:              cfg.Addr,
		Handler:           httpx.Chain(mux, httpx.RequestID, httpx.Recover, httpx.Log),
		ReadHeaderTimeout: 10 * time.Second,
		ReadTimeout:       30 * time.Second,
		WriteTimeout:      60 * time.Second,
		IdleTimeout:       2 * time.Minute,
	}
	return httpx.Serve(ctx, srv, cfg.ShutdownTimeout)
}
