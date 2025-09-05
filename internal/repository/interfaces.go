package repository

import (
	"context"
	"image"

	"github.com/anime-shed/image-inspector-go/pkg/models"
)

// ImageRepository defines the interface for image data access operations
type ImageRepository interface {
	// FetchImage retrieves an image from a URL
	FetchImage(ctx context.Context, imageURL string) (image.Image, error)

	// ValidateImageURL validates if the provided URL is acceptable
	ValidateImageURL(imageURL string) error

	// GetImageMetadata retrieves metadata about an image without downloading it
	GetImageMetadata(ctx context.Context, imageURL string) (*ImageMetadata, error)
}

// ImageMetadata is now an alias to the shared models.ImageMetadata
type ImageMetadata = models.ImageMetadata

// AnalysisRepository defines the interface for analysis result operations
type AnalysisRepository interface {
	// SaveAnalysisResult stores an analysis result
	SaveAnalysisResult(ctx context.Context, result *models.AnalysisResult) error

	// GetAnalysisResult retrieves a stored analysis result
	GetAnalysisResult(ctx context.Context, id string) (*models.AnalysisResult, error)

	// GetAnalysisHistory retrieves analysis history for a specific image URL
	GetAnalysisHistory(ctx context.Context, imageURL string) ([]*models.AnalysisResult, error)
}

// AnalysisResult is now an alias to the shared models.AnalysisResult
type AnalysisResult = models.AnalysisResult
