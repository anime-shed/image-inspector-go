package main

import (
	"context"
	"errors"
	"go-image-inspector/internal/analyzer"
	"go-image-inspector/internal/storage"
	"go-image-inspector/internal/transport"
	"go-image-inspector/pkg/config"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"
)

const (
	shutdownTimeout = 10 * time.Second
	readTimeout     = 15 * time.Second
	writeTimeout    = 30 * time.Second
)

func main() {
	// Initialize configuration
	cfg, err := config.LoadFromEnv()
	if err != nil {
		log.Fatalf("Failed to load configuration: %v", err)
	}

	// Initialize dependencies
	imageFetcher := storage.NewHTTPImageFetcher()
	imageAnalyzer, err := analyzer.NewImageAnalyzer()
	if err != nil {
		log.Fatalf("Failed to create image analyzer: %v", err)
	}

	// Create HTTP handler with dependencies
	router := transport.NewHandler(imageAnalyzer, imageFetcher, cfg)

	// Configure HTTP server with config-based timeouts
	server := &http.Server{
		Addr:         cfg.ServerAddress(),
		Handler:      router,
		ReadTimeout:  cfg.RequestTimeout,
		WriteTimeout: cfg.RequestTimeout + 5*time.Second, // Add buffer for response
	}

	// Start server in goroutine
	go func() {
		log.Printf("Starting server on %s", server.Addr)
		if err := server.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			log.Fatalf("Server failed: %v", err)
		}
	}()

	// Graceful shutdown handling
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	log.Println("Shutting down server...")

	ctx, cancel := context.WithTimeout(context.Background(), shutdownTimeout)
	defer cancel()

	if err := server.Shutdown(ctx); err != nil {
		log.Fatalf("Server forced to shutdown: %v", err)
	}

	log.Println("Server exited properly")
}
