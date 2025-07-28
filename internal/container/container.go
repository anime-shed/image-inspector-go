package container

import (
	"fmt"
	"go-image-inspector/internal/analyzer"
	"go-image-inspector/internal/repository"
	"go-image-inspector/internal/service"
	"go-image-inspector/internal/storage"
	"go-image-inspector/pkg/config"
)

// Container holds all application dependencies
type Container struct {
	Config               *config.Config
	ImageFetcher         storage.ImageFetcher
	ImageAnalyzer        analyzer.ImageAnalyzer
	ImageRepository      repository.ImageRepository
	ImageAnalysisService service.ImageAnalysisService
}

// NewContainer creates a new dependency injection container
func NewContainer() (*Container, error) {
	// Load configuration
	cfg, err := config.LoadFromEnv()
	if err != nil {
		return nil, fmt.Errorf("failed to load config: %w", err)
	}

	// Create image fetcher
	imageFetcher := storage.NewHTTPImageFetcher()

	// Create image analyzer
	imageAnalyzer, err := analyzer.NewImageAnalyzer()
	if err != nil {
		return nil, err
	}

	// Create image repository
	imageRepository := repository.NewHTTPImageRepository(imageFetcher)

	// Create image analysis service
	imageAnalysisService := service.NewImageAnalysisService(imageRepository, imageAnalyzer)

	return &Container{
		Config:               cfg,
		ImageFetcher:         imageFetcher,
		ImageAnalyzer:        imageAnalyzer,
		ImageRepository:      imageRepository,
		ImageAnalysisService: imageAnalysisService,
	}, nil
}

// GetConfig returns the configuration
func (c *Container) GetConfig() *config.Config {
	return c.Config
}

// GetImageFetcher returns the image fetcher
func (c *Container) GetImageFetcher() storage.ImageFetcher {
	return c.ImageFetcher
}

// GetImageAnalyzer returns the image analyzer
func (c *Container) GetImageAnalyzer() analyzer.ImageAnalyzer {
	return c.ImageAnalyzer
}

// GetImageRepository returns the image repository
func (c *Container) GetImageRepository() repository.ImageRepository {
	return c.ImageRepository
}

// GetImageAnalysisService returns the image analysis service
func (c *Container) GetImageAnalysisService() service.ImageAnalysisService {
	return c.ImageAnalysisService
}