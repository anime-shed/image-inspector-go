package main

import (
	"context"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"go-image-inspector/internal/container"
	"go-image-inspector/internal/transport"

	"github.com/sirupsen/logrus"
)

func main() {
	// Initialize dependency injection container
	c, err := container.NewContainer()
	if err != nil {
		log.Fatalf("Failed to initialize container: %v", err)
	}

	cfg := c.GetConfig()

	// Setup structured logging
	logrus.SetFormatter(&logrus.JSONFormatter{})
	logrus.SetLevel(logrus.InfoLevel)

	// Create HTTP handler with dependencies from container
	handler := transport.NewHandler(c.GetImageAnalyzer(), c.GetImageFetcher(), cfg)

	// Create HTTP server with configurable timeouts
	server := &http.Server{
		Addr:         cfg.ServerAddress(),
		Handler:      handler,
		ReadTimeout:  cfg.RequestTimeout,
		WriteTimeout: cfg.RequestTimeout,
	}

	// Start server in a goroutine
	go func() {
		logrus.WithFields(logrus.Fields{
			"address": cfg.ServerAddress(),
			"timeout": cfg.RequestTimeout,
		}).Info("Starting HTTP server")

		if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			logrus.WithError(err).Fatal("Failed to start server")
		}
	}()

	// Wait for interrupt signal to gracefully shutdown the server
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	logrus.Info("Shutting down server...")

	// Create a deadline for shutdown
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	// Attempt graceful shutdown
	if err := server.Shutdown(ctx); err != nil {
		logrus.WithError(err).Fatal("Server forced to shutdown")
	}

	logrus.Info("Server exited")
}
