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

	"github.com/spdedsec/gopherd/internal/config"
	"github.com/spdedsec/gopherd/internal/database"
	"github.com/spdedsec/gopherd/internal/handler"
	"github.com/spdedsec/gopherd/internal/middleware"
	"github.com/spdedsec/gopherd/internal/repository"
	"github.com/spdedsec/gopherd/internal/server"
	"github.com/spdedsec/gopherd/internal/service"
)

func main() {
	cfg, err := config.Load()
	if err != nil {
		logError(err)
		os.Exit(1)
	}

	logger := slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{Level: cfg.LogLevel}))
	slog.SetDefault(logger)

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	db, err := database.Open(ctx, cfg.DatabaseURL, cfg.DBMaxConns, cfg.DBMinConns)
	if err != nil {
		logger.Error("database connection failed", "error", err)
		os.Exit(1)
	}
	defer db.Close()

	if err := database.Migrate(ctx, db); err != nil {
		logger.Error("database migration failed", "error", err)
		os.Exit(1)
	}

	users := repository.NewUserRepository(db)
	sessions := repository.NewSessionRepository(db)
	tasks := repository.NewTaskRepository(db)

	authService := service.NewAuthService(users, sessions, cfg.SessionTTL)
	taskService := service.NewTaskService(tasks)

	metrics := middleware.NewMetrics()
	limiter := middleware.NewRateLimiter(cfg.RateLimitRPS, cfg.RateLimitBurst)

	router := server.NewRouter(server.Dependencies{
		Config:      cfg,
		Auth:        handler.NewAuthHandler(authService, cfg.MaxBodyBytes),
		Tasks:       handler.NewTaskHandler(taskService, cfg.MaxBodyBytes),
		Health:      handler.NewHealthHandler(db),
		Metrics:     metrics,
		RateLimiter: limiter,
		AuthService: authService,
	})

	httpServer := &http.Server{
		Addr:              cfg.ListenAddr,
		Handler:           middleware.RequestID(middleware.Recover(logger, metrics, middleware.Logging(logger, metrics, router))),
		ReadHeaderTimeout: 5 * time.Second,
		ReadTimeout:       cfg.ReadTimeout,
		WriteTimeout:      cfg.WriteTimeout,
		IdleTimeout:       cfg.IdleTimeout,
		MaxHeaderBytes:    1 << 20,
	}

	go func() {
		logger.Info("gopherd started", "addr", cfg.ListenAddr, "environment", cfg.Environment)
		if err := httpServer.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			logger.Error("http server failed", "error", err)
			stop()
		}
	}()

	<-ctx.Done()
	shutdownCtx, cancel := context.WithTimeout(context.Background(), cfg.ShutdownTimeout)
	defer cancel()
	logger.Info("shutting down")
	if err := httpServer.Shutdown(shutdownCtx); err != nil {
		logger.Error("graceful shutdown failed", "error", err)
		os.Exit(1)
	}
	logger.Info("shutdown complete")
}

func logError(err error) { _, _ = os.Stderr.WriteString("gopherd: " + err.Error() + "\n") }
