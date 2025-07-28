package repository

import (
	"context"
	"image"
	"go-image-inspector/internal/storage"
)

// HTTPImageRepository implements ImageRepository using HTTP storage
type HTTPImageRepository struct {
	fetcher storage.ImageFetcher
}

// NewHTTPImageRepository creates a new HTTP-based image repository
func NewHTTPImageRepository(fetcher storage.ImageFetcher) ImageRepository {
	return &HTTPImageRepository{
		fetcher: fetcher,
	}
}

// FetchImage retrieves an image from a URL
func (r *HTTPImageRepository) FetchImage(ctx context.Context, imageURL string) (image.Image, error) {
	return r.fetcher.FetchImage(ctx, imageURL)
}

// ValidateImageURL validates if the provided URL is acceptable
func (r *HTTPImageRepository) ValidateImageURL(imageURL string) error {
	// This would typically use a validator from the storage layer
	// For now, we'll implement basic validation
	if imageURL == "" {
		return ErrInvalidImageURL
	}
	return nil
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