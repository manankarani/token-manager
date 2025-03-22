package main

import (
	"context"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"strconv"
	"syscall"

	"github.com/manankarani/token-manager/datasources"
	"github.com/manankarani/token-manager/env"
	"github.com/manankarani/token-manager/internal/handlers"
	"github.com/manankarani/token-manager/internal/repositories"
	"github.com/manankarani/token-manager/internal/services"
	"github.com/manankarani/token-manager/internal/workers"
)

func main() {
	// Initialize logger
	logger := slog.New(slog.NewJSONHandler(os.Stdout, nil))
	// Load environment variables
	env.Load()

	// Initialize Redis client
	redisClient := datasources.NewRedisClient()
	defer redisClient.Close()

	// Initialize repositories, services, and controllers
	tokenRepo := repositories.NewTokenRepository(redisClient)
	tokenService := services.NewTokenService(tokenRepo)
	tokenHandler := handlers.NewTokenHandler(tokenService)

	// Setup routes
	router := handlers.SetupRoutes(tokenHandler)

	// Context for graceful shutdown
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// TODO: can be migrated to a new microservice
	go workers.StartCleanupWorker(ctx, tokenService.CleanupExpiredTokens, logger)

	// Create HTTP server
	srv := &http.Server{Addr: ":" + strconv.Itoa(env.Conf.Server.Port), Handler: router}

	// Handle OS signals for graceful shutdown
	stop := make(chan os.Signal, 1)
	signal.Notify(stop, os.Interrupt, syscall.SIGTERM)

	go func() {
		<-stop
		logger.Info("Shutting down server...")

		// Stop cleanup worker
		cancel()

		// Gracefully shutdown HTTP server
		if err := srv.Shutdown(context.Background()); err != nil {
			logger.Error("HTTP server shutdown error", slog.String("error", err.Error()))
		}
	}()

	logger.Info("Server running on :8080")
	if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
		logger.Error("Server error", slog.String("error", err.Error()))
	}
}
