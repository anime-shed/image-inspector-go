package container

import (
	"fmt"
	"net/http"
	"go-image-inspector/internal/analyzer"
	"go-image-inspector/internal/repository"
	"go-image-inspector/internal/service"
	"go-image-inspector/internal/storage"
	"go-image-inspector/internal/transport"
	"go-image-inspector/pkg/config"
)

// Container holds all application dependencies
type Container struct {
	config               *config.Config
	imageFetcher         storage.ImageFetcher
	imageAnalyzer        analyzer.ImageAnalyzer
	imageRepository      repository.ImageRepository
	imageAnalysisService service.ImageAnalysisService
	handler              http.Handler
}

// NewContainer creates a new dependency injection container
func NewContainer() (*Container, error) {
	// Load configuration
	cfg, err := config.LoadFromEnv()
	if err != nil {
		return nil, fmt.Errorf("failed to load config: %w", err)
	}

	// Build dependency graph
	imageFetcher := storage.NewHTTPImageFetcher()
	imageAnalyzer, err := analyzer.NewImageAnalyzer()
	if err != nil {
		return nil, err
	}

	imageRepository := repository.NewHTTPImageRepository(imageFetcher)
	imageAnalysisService := service.NewImageAnalysisService(imageRepository, imageAnalyzer)
	handler := transport.NewHandler(imageAnalysisService, cfg)

	return &Container{
		config:               cfg,
		imageFetcher:         imageFetcher,
		imageAnalyzer:        imageAnalyzer,
		imageRepository:      imageRepository,
		imageAnalysisService: imageAnalysisService,
		handler:              handler,
	}, nil
}

// Handler returns the HTTP handler
func (c *Container) Handler() http.Handler {
	return c.handler
}

// Config returns the configuration
func (c *Container) Config() *config.Config {
	return c.config
}