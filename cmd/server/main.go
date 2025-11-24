package main

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	_ "github.com/jackc/pgx/v5/stdlib"

	"leveltalk/internal/config"
	"leveltalk/internal/dialogs"
	apphttp "leveltalk/internal/http"
	"leveltalk/internal/llm"
	"leveltalk/internal/storage"
	"leveltalk/internal/tts"
	"leveltalk/internal/ui"
	"leveltalk/migrations"
)

func main() {
	logger := slog.New(slog.NewTextHandler(os.Stdout, nil))
	if err := run(logger); err != nil {
		logger.Error("startup failed", slog.String("error", err.Error()))
		os.Exit(1)
	}
}

func run(logger *slog.Logger) error {
	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	cfg, err := config.Load()
	if err != nil {
		return fmt.Errorf("load config: %w", err)
	}

	db, err := sql.Open("pgx", cfg.DBDSN)
	if err != nil {
		return fmt.Errorf("open db: %w", err)
	}
	defer db.Close()

	// ensure DB is reachable
	if err := pingDB(ctx, db); err != nil {
		return err
	}

	if err := storage.RunMigrations(ctx, db, migrations.Files); err != nil {
		return fmt.Errorf("run migrations: %w", err)
	}

	repo := storage.NewDialogRepository(db)
	dialogService := dialogs.NewService(repo, llm.NewStubClient(logger), tts.NewStubClient())

	tmpl, err := ui.ParseTemplates()
	if err != nil {
		return fmt.Errorf("parse templates: %w", err)
	}

	staticFS := ui.StaticFiles()

	handler := apphttp.NewServer(logger, dialogService, tmpl, staticFS)

	server := &http.Server{
		Addr:    ":" + cfg.Port,
		Handler: handler,
	}

	errCh := make(chan error, 1)
	go func() {
		logger.Info("server listening", slog.String("addr", server.Addr))
		errCh <- server.ListenAndServe()
	}()

	select {
	case <-ctx.Done():
		logger.Info("shutdown signal received")
		shutdownCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()
		if err := server.Shutdown(shutdownCtx); err != nil && !errors.Is(err, http.ErrServerClosed) {
			return fmt.Errorf("shutdown server: %w", err)
		}
	case err := <-errCh:
		if err != nil && !errors.Is(err, http.ErrServerClosed) {
			return fmt.Errorf("server error: %w", err)
		}
	}

	return nil
}

func pingDB(ctx context.Context, db *sql.DB) error {
	const (
		maxAttempts = 10
		baseDelay   = time.Second
	)

	var err error
	for attempt := 1; attempt <= maxAttempts; attempt++ {
		pingCtx, cancel := context.WithTimeout(ctx, 5*time.Second)
		err = db.PingContext(pingCtx)
		cancel()

		if err == nil {
			return nil
		}

		// allow caller to abort early
		select {
		case <-ctx.Done():
			return fmt.Errorf("ping db: %w", err)
		case <-time.After(time.Duration(attempt) * baseDelay):
		}
	}

	return fmt.Errorf("ping db: %w", err)
}
