package repository

import (
	"context"
	"fmt"
	"image"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/anime-shed/image-inspector-go/internal/storage"
	"github.com/anime-shed/image-inspector-go/pkg/validation"
)

// HTTPImageRepository implements ImageRepository using HTTP storage
type HTTPImageRepository struct {
	fetcher   storage.ImageFetcher
	validator *validation.URLValidator
	timeout   time.Duration
}

// NewHTTPImageRepository creates a new HTTP-based image repository
func NewHTTPImageRepository(fetcher storage.ImageFetcher, timeout time.Duration) ImageRepository {
	return &HTTPImageRepository{
		fetcher:   fetcher,
		validator: validation.NewURLValidator(),
		timeout:   timeout,
	}
}

// FetchImage retrieves an image from a URL
func (r *HTTPImageRepository) FetchImage(ctx context.Context, imageURL string) (image.Image, error) {
	if err := r.validator.ValidateImageURL(imageURL); err != nil {
		return nil, fmt.Errorf("invalid image URL: %w", err)
	}
	return r.fetcher.FetchImage(ctx, imageURL)
}

// ValidateImageURL validates if the provided URL is acceptable
func (r *HTTPImageRepository) ValidateImageURL(imageURL string) error {
	return r.validator.ValidateImageURL(imageURL)
}

// GetImageMetadata retrieves metadata about an image without downloading it
func (r *HTTPImageRepository) GetImageMetadata(ctx context.Context, imageURL string) (*ImageMetadata, error) {
	// Validate URL before making any network calls
	if err := r.validator.ValidateImageURL(imageURL); err != nil {
		return nil, fmt.Errorf("invalid image URL: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, "HEAD", imageURL, nil)
	if err != nil {
		return nil, fmt.Errorf("invalid URL: %w", err)
	}

	client := &http.Client{
		Timeout: r.timeout,
		CheckRedirect: func(req *http.Request, via []*http.Request) error {
			if len(via) >= 3 {
				return fmt.Errorf("too many redirects (limit: 3)")
			}
			// Re-validate redirect URL to prevent SSRF via redirects
			return r.validator.ValidateImageURL(req.URL.String())
		},
	}

	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch metadata: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("unexpected status code %d", resp.StatusCode)
	}

	// Extract content length
	contentLength := int64(0)
	if contentLengthStr := resp.Header.Get("Content-Length"); contentLengthStr != "" {
		if cl, err := strconv.ParseInt(contentLengthStr, 10, 64); err == nil {
			contentLength = cl
		}
	}

	// Extract content type and format
	contentType := resp.Header.Get("Content-Type")
	if contentType == "" {
		contentType = "image/jpeg" // Default fallback
	}

	format := "JPEG" // Default
	ct := strings.ToLower(contentType)
	if strings.Contains(ct, "png") {
		format = "PNG"
	} else if strings.Contains(ct, "gif") {
		format = "GIF"
	} else if strings.Contains(ct, "webp") {
		format = "WEBP"
	}

	return &ImageMetadata{
		ContentType:   contentType,
		ContentLength: contentLength,
		Format:        format,
	}, nil
}
