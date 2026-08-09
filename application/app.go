package application

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"os/signal"
	"syscall"
	"time"

	_ "github.com/golang-migrate/migrate/v4/database/postgres"
	_ "github.com/golang-migrate/migrate/v4/source/file"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/golang-migrate/migrate/v4"
	"github.com/maxim-shishkov/auth-service/internal/config"
	"github.com/maxim-shishkov/auth-service/internal/repository"
	"github.com/maxim-shishkov/auth-service/internal/service"
	"github.com/maxim-shishkov/auth-service/internal/token"
	transporthttp "github.com/maxim-shishkov/auth-service/internal/transport/http"
)

const (
	DefaultConf   = "config.json"
	migrationsURL = "file://migrations"
)

func Run(log *slog.Logger) error {
	cfg, err := config.Load(DefaultConf)
	if err != nil {
		return fmt.Errorf("config load: %w", err)
	}

	if err = migrateUp(cfg); err != nil {
		return fmt.Errorf("migrate: %w", err)
	}

	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	pool, err := pgxpool.New(ctx, cfg.PostgresDSN)
	if err != nil {
		return fmt.Errorf("pgpool new: %w", err)
	}
	defer pool.Close()

	jwtTtl, err := time.ParseDuration(cfg.JWTTTL)
	if err != nil {
		return fmt.Errorf("parse duration: %w", err)
	}

	userRepo := repository.NewUserRepository(pool)
	tokenManager := token.NewManager(cfg.JWTSecret, jwtTtl)
	authService := service.NewAuthService(userRepo, tokenManager)
	handler := transporthttp.NewHandler(authService, log)
	router := transporthttp.NewRouter(handler, tokenManager, log)

	srv := &http.Server{
		Addr:              fmt.Sprintf(":%d", cfg.Port),
		Handler:           router,
		ReadHeaderTimeout: 5 * time.Second,
	}

	errCh := make(chan error, 1)
	go func() {
		log.Info("http server started", "port", cfg.Port)

		if err := srv.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			errCh <- err
		}
	}()

	select {
	case <-ctx.Done():
		log.Info("shutdown signal received")
	case err := <-errCh:
		return err
	}

	shutdownCtx, cancel := context.WithTimeout(context.Background(), time.Second*5)
	defer cancel()

	if err := srv.Shutdown(shutdownCtx); err != nil {
		return fmt.Errorf("shutdown: %w", err)
	}

	log.Info("server stopped gracefully")

	return nil
}

func migrateUp(cfg config.Config) error {
	m, err := migrate.New(migrationsURL, cfg.PostgresDSN)
	if err != nil {
		return fmt.Errorf("migrate.New: %w", err)
	}
	defer func() {
		_, _ = m.Close()
	}()

	err = m.Up()
	if err != nil && !errors.Is(err, migrate.ErrNoChange) {
		var errDirty migrate.ErrDirty
		if errors.As(err, &errDirty) {
			if err := m.Force(errDirty.Version); err != nil {
				return fmt.Errorf("migrate.Force: %w", err)
			}

			return nil
		}

		return fmt.Errorf("migrate.Up: %w", err)
	}

	return nil
}
