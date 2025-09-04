package container

import (
	"go-image-inspector/internal/analyzer"
	"go-image-inspector/internal/config"
	"go-image-inspector/internal/repository"
	"go-image-inspector/internal/service"
	"go-image-inspector/internal/storage"
	"go-image-inspector/internal/transport"
	"net/http"
)

// Container holds all application dependencies using dependency injection
type Container struct {
	config                     *config.Config
	imageFetcher              storage.ImageFetcher
	imageAnalyzer             analyzer.ImageAnalyzer
	imageRepository           repository.ImageRepository
	analysisService    service.ImageAnalysisService
	handler                   http.Handler
}

// NewContainer creates and initializes all dependencies using dependency injection
func NewContainer(cfg *config.Config) (*Container, error) {
	// Create image fetcher
	imageFetcher := storage.NewHTTPImageFetcher()

	// Create single image analyzer (remove duplication)
	imageAnalyzer, err := analyzer.NewCoreAnalyzer()
	if err != nil {
		return nil, err
	}

	// Create image repository
	imageRepository := repository.NewHTTPImageRepository(imageFetcher)

	// Create analysis service (single service for both endpoints)
	analysisService := service.NewImageAnalysisService(imageRepository, imageAnalyzer)

	// Create HTTP handler with service
	handler := transport.NewHandler(analysisService, cfg)

	return &Container{
		config:                  cfg,
		imageFetcher:            imageFetcher,
		imageAnalyzer:           imageAnalyzer,
		imageRepository:         imageRepository,
		analysisService:  analysisService,
		handler:                 handler,
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

// GetAnalysisService returns the analysis service
func (c *Container) GetAnalysisService() service.ImageAnalysisService {
	return c.analysisService
}
