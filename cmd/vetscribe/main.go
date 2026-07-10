// Command vetscribe is the single-binary VetScribe server: it serves the embedded
// frontend and the JSON API from one static executable.
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

	"github.com/Vero-Ventures/vetscribe/internal/api"
	"github.com/Vero-Ventures/vetscribe/internal/config"
)

// build metadata, overridden at link time via -ldflags.
var (
	version = "dev"
	commit  = "none"
	date    = "unknown"
)

func main() {
	logger := slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelInfo}))
	slog.SetDefault(logger)

	cfg, err := config.Load()
	if err != nil {
		logger.Error("configuration error", "error", err)
		os.Exit(2)
	}

	srv := api.NewServer(api.ServerOptions{
		BuildInfo: api.BuildInfo{Version: version, Commit: commit, Date: date},
		Logger:    logger,
	})

	httpServer := &http.Server{
		Addr:              cfg.ListenAddr,
		Handler:           srv.Handler(),
		ReadHeaderTimeout: 10 * time.Second,
		ReadTimeout:       5 * time.Minute,
		WriteTimeout:      5 * time.Minute,
		IdleTimeout:       2 * time.Minute,
	}

	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	// serveErr carries a fatal ListenAndServe failure (e.g. the port is taken) out
	// of the goroutine so the process can exit non-zero instead of masking it.
	serveErr := make(chan error, 1)
	go func() {
		logger.Info("vetscribe listening", "addr", cfg.ListenAddr, "version", version)
		if err := httpServer.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			serveErr <- err
			stop()
			return
		}
		serveErr <- nil
	}()

	select {
	case <-ctx.Done():
		logger.Info("shutting down")
	case err := <-serveErr:
		if err != nil {
			logger.Error("server failed", "error", err)
			os.Exit(1)
		}
		return
	}

	shutdownCtx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()
	if err := httpServer.Shutdown(shutdownCtx); err != nil {
		logger.Error("graceful shutdown failed", "error", err)
		os.Exit(1)
	}
}
