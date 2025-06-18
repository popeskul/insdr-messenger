// Package main is the entry point for the insdr-messenger HTTP server.
package main

import (
	"context"
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/go-redis/redis/v8"
	"github.com/jmoiron/sqlx"
	_ "github.com/lib/pq"
	"go.uber.org/zap"
	"golang.org/x/time/rate"

	"github.com/popeskul/insdr-messenger/internal/config"
	"github.com/popeskul/insdr-messenger/internal/handler"
	"github.com/popeskul/insdr-messenger/internal/middleware"
	"github.com/popeskul/insdr-messenger/internal/repository"
	"github.com/popeskul/insdr-messenger/internal/service"
)

func main() {
	logger, err := zap.NewProduction()
	if err != nil {
		panic(fmt.Sprintf("failed to initialize logger: %v", err))
	}
	defer func() {
		if syncErr := logger.Sync(); syncErr != nil {
			// Handle error silently
			_ = syncErr
		}
	}()

	cfg, err := config.LoadConfig("config.yaml")
	if err != nil {
		logger.Fatal("Failed to load configuration", zap.Error(err))
	}

	db, err := sqlx.Connect("postgres", cfg.Database.GetDSN())
	if err != nil {
		logger.Fatal("Failed to connect to database", zap.Error(err))
	}
	defer func() {
		if err := db.Close(); err != nil {
			logger.Error("Failed to close database connection", zap.Error(err))
		}
	}()

	db.SetMaxOpenConns(25)
	db.SetMaxIdleConns(5)
	db.SetConnMaxLifetime(5 * time.Minute)

	redisClient := redis.NewClient(&redis.Options{
		Addr:     fmt.Sprintf("%s:%d", cfg.Redis.Host, cfg.Redis.Port),
		Password: cfg.Redis.Password,
		DB:       cfg.Redis.DB,
	})

	ctx := context.Background()
	if err := redisClient.Ping(ctx).Err(); err != nil {
		logger.Fatal("Failed to connect to Redis", zap.Error(err))
	}
	defer func() {
		if err := redisClient.Close(); err != nil {
			logger.Error("Failed to close Redis connection", zap.Error(err))
		}
	}()

	repo := repository.NewRepository(db)
	svc := service.NewService(cfg, repo, redisClient, logger)

	handler := handler.NewHandler(svc, logger)

	router := setupRouter(handler)

	middlewareConfig := &middleware.Config{
		Logger: logger,
		CORS: &middleware.CORSConfig{
			AllowedOrigins:   cfg.Middleware.AllowedOrigins,
			AllowedMethods:   []string{"GET", "POST", "PUT", "DELETE", "OPTIONS"},
			AllowedHeaders:   []string{"Accept", "Authorization", "Content-Type", "X-Request-ID"},
			ExposedHeaders:   []string{"X-Request-ID"},
			AllowCredentials: false,
			MaxAge:           86400,
		},
		RateLimit:      rate.Limit(cfg.Middleware.RateLimit),
		RateLimitBurst: cfg.Middleware.RateLimitBurst,
		RequestTimeout: 30 * time.Second,
	}

	finalHandler := middleware.Chain(middlewareConfig)(router)

	srv := &http.Server{
		Addr:         ":" + cfg.Server.Port,
		Handler:      finalHandler,
		ReadTimeout:  time.Duration(cfg.Server.ReadTimeout) * time.Second,
		WriteTimeout: time.Duration(cfg.Server.WriteTimeout) * time.Second,
		IdleTimeout:  60 * time.Second,
	}

	// Start scheduler automatically
	if err := svc.Scheduler.Start(); err != nil {
		logger.Error("Failed to start scheduler on startup", zap.Error(err))
	} else {
		logger.Info("Scheduler started automatically on application startup")
	}

	// Start server in goroutine
	go func() {
		logger.Info("Starting server", zap.String("address", srv.Addr))
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			logger.Fatal("Failed to start server", zap.Error(err))
		}
	}()

	// Wait for interrupt signal
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	logger.Info("Shutting down server...")

	// Stop scheduler
	if svc.Scheduler.IsRunning() {
		if err := svc.Scheduler.Stop(); err != nil {
			logger.Error("Failed to stop scheduler", zap.Error(err))
		}
	}

	// Graceful shutdown
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	if err := srv.Shutdown(ctx); err != nil {
		logger.Fatal("Server forced to shutdown", zap.Error(err))
	}

	logger.Info("Server exited")
}
