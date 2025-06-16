package main

import (
	"context"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/wcygan/llm-json-parse/internal/client"
	"github.com/wcygan/llm-json-parse/internal/config"
	"github.com/wcygan/llm-json-parse/internal/logging"
	"github.com/wcygan/llm-json-parse/internal/middleware"
	"github.com/wcygan/llm-json-parse/internal/server"
)

func main() {
	// Load configuration
	cfg, err := config.LoadConfig()
	if err != nil {
		log.Fatalf("Failed to load configuration: %v", err)
	}

	// Create structured logger
	logger := logging.NewLogger(logging.LogConfig{
		Level:  cfg.Log.Level,
		Format: cfg.Log.Format,
	})

	// Log startup information
	startupConfig := map[string]interface{}{
		"address":       cfg.Address(),
		"llm_server":    cfg.LLM.ServerURL,
		"cache_size":    cfg.Cache.MaxSize,
		"log_level":     cfg.Log.Level,
		"log_format":    cfg.Log.Format,
		"read_timeout":  cfg.Server.ReadTimeout.String(),
		"write_timeout": cfg.Server.WriteTimeout.String(),
		"idle_timeout":  cfg.Server.IdleTimeout.String(),
	}
	logger.LogStartup(startupConfig)

	// Create LLM client with configuration
	llmClient := client.NewLlamaServerClientWithTimeout(cfg.LLM.ServerURL, cfg.LLM.Timeout)

	// Create server with configuration and logger
	srv := server.NewServerWithConfig(llmClient, cfg.Cache.MaxSize, logger)

	// Setup HTTP server with timeouts
	httpServer := &http.Server{
		Addr:         cfg.Address(),
		ReadTimeout:  cfg.Server.ReadTimeout,
		WriteTimeout: cfg.Server.WriteTimeout,
		IdleTimeout:  cfg.Server.IdleTimeout,
	}

	// Register routes with middleware
	mux := http.NewServeMux()
	srv.RegisterRoutes(mux)

	// Apply middleware chain
	handler := middleware.Recovery(logger)(
		middleware.CORS()(
			middleware.RequestTimeout(cfg.Server.WriteTimeout)(
				middleware.ContentType("application/json")(
					middleware.RequestLogging(logger)(mux),
				),
			),
		),
	)
	httpServer.Handler = handler

	// Channel to listen for interrupt signal to terminate gracefully
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)

	// Start server in a goroutine
	go func() {
		logger.WithComponent("http_server").Info("Server listening", "address", cfg.Address())
		if err := httpServer.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			logger.WithComponent("http_server").WithError(err).Error("Server failed to start")
			log.Fatalf("Server failed to start: %v", err)
		}
	}()

	// Wait for interrupt signal
	<-quit
	logger.WithComponent("http_server").Info("Shutdown signal received")

	// Create a context with timeout for graceful shutdown
	shutdownStart := time.Now()
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	// Graceful shutdown
	if err := httpServer.Shutdown(ctx); err != nil {
		logger.WithComponent("http_server").WithError(err).Error("Forced shutdown")
		logger.LogShutdown(false, time.Since(shutdownStart))
		log.Fatalf("Server forced to shutdown: %v", err)
	}

	logger.LogShutdown(true, time.Since(shutdownStart))
}
