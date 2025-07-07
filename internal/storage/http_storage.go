package storage

import (
	"context"
	"crypto/tls"
	"fmt"
	"image"
	_ "image/jpeg"
	_ "image/png"
	"net/http"
	"time"
)

type ImageFetcher interface {
	FetchImage(ctx context.Context, imageURL string) (image.Image, error)
}

type HTTPImageFetcher struct {
	client *http.Client
}

func NewHTTPImageFetcher() ImageFetcher {
	// Create HTTP client with TLS verification completely disabled
	transport := &http.Transport{
		TLSClientConfig: &tls.Config{
			InsecureSkipVerify: true,
		},
		MaxIdleConns:        100,
		MaxIdleConnsPerHost: 10,
		IdleConnTimeout:     90 * time.Second,
	}

	return &HTTPImageFetcher{
		client: &http.Client{
			Transport: transport,
			Timeout:   30 * time.Second,
		},
	}
}

func (h *HTTPImageFetcher) FetchImage(ctx context.Context, imageURL string) (image.Image, error) {
	req, err := http.NewRequestWithContext(ctx, "GET", imageURL, nil)
	if err != nil {
		return nil, fmt.Errorf("invalid URL: %w", err)
	}

	// Set headers for proper image handling
	req.Header.Set("Accept", "image/jpeg, image/png, image/webp, */*")
	req.Header.Set("User-Agent", "Go-Image-Inspector/1.0")

	resp, err := h.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch image from %s: %w", imageURL, err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("unexpected status code %d when fetching image from %s", resp.StatusCode, imageURL)
	}

	img, _, err := image.Decode(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to decode image: %w", err)
	}

	return img, nil
}
