package repository

import (
	"context"
	"image"
	"go-image-inspector/internal/storage"
	"go-image-inspector/pkg/validation"
)

// HTTPImageRepository implements ImageRepository using HTTP storage
type HTTPImageRepository struct {
	fetcher   storage.ImageFetcher
	validator *validation.URLValidator
}

// NewHTTPImageRepository creates a new HTTP-based image repository
func NewHTTPImageRepository(fetcher storage.ImageFetcher) ImageRepository {
	return &HTTPImageRepository{
		fetcher:   fetcher,
		validator: validation.NewURLValidator(),
	}
}

// FetchImage retrieves an image from a URL
func (r *HTTPImageRepository) FetchImage(ctx context.Context, imageURL string) (image.Image, error) {
	return r.fetcher.FetchImage(ctx, imageURL)
}

// ValidateImageURL validates if the provided URL is acceptable
func (r *HTTPImageRepository) ValidateImageURL(imageURL string) error {
	return r.validator.ValidateImageURL(imageURL)
}

// GetImageMetadata retrieves metadata about an image without downloading it
func (r *HTTPImageRepository) GetImageMetadata(ctx context.Context, imageURL string) (*ImageMetadata, error) {
	// This is a placeholder implementation
	// In a real scenario, this would make a HEAD request to get metadata
	return &ImageMetadata{
		ContentType: "image/jpeg",
		Format:      "JPEG",
	}, nil
}