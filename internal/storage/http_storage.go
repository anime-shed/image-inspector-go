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

// HTTPImageFetcher implements ImageFetcher with performance enhancements
type HTTPImageFetcher struct {
	client *http.Client
}

// NewHTTPImageFetcher creates an HTTP image fetcher
// Implements optimizations from PERFORMANCE_OPTIMIZATION_ANALYSIS.md Phase 1
func NewHTTPImageFetcher() ImageFetcher {
	// Optimized transport configuration for single image downloads
	transport := &http.Transport{
		// Connection pooling optimized for image fetching
		MaxIdleConns:        10, // Reduced from 100 (memory efficient)
		MaxIdleConnsPerHost: 2,  // Reduced from 10 (single image focus)
		IdleConnTimeout:     30 * time.Second,

		// Timeouts optimized for image downloads
		TLSHandshakeTimeout:   10 * time.Second,
		ResponseHeaderTimeout: 10 * time.Second,
		ExpectContinueTimeout: 1 * time.Second,

		// Memory optimizations
		DisableCompression:     false, // Enable compression for images
		MaxResponseHeaderBytes: 4096,  // Limit header size

		// TLS configuration (as per current requirements)
		TLSClientConfig: &tls.Config{
			InsecureSkipVerify: true,
		},
	}

	return &HTTPImageFetcher{
		client: &http.Client{
			Transport: transport,
			Timeout:   30 * time.Second,

			// Prevent redirects to avoid unexpected behavior
			CheckRedirect: func(req *http.Request, via []*http.Request) error {
				if len(via) >= 3 {
					return fmt.Errorf("too many redirects (limit: 3)")
				}
				return nil
			},
		},
	}
}

func (h *HTTPImageFetcher) FetchImage(ctx context.Context, imageURL string) (image.Image, error) {
	req, err := http.NewRequestWithContext(ctx, "GET", imageURL, nil)
	if err != nil {
		return nil, fmt.Errorf("invalid URL: %w", err)
	}

	// Headers for image downloads
	req.Header.Set("Accept", "image/jpeg, image/png, image/webp, image/gif, */*")
	req.Header.Set("User-Agent", "Go-Image-Inspector/2.0")
	req.Header.Set("Accept-Encoding", "gzip, deflate")

	// Retry logic (3 attempts) - only retry on transient errors
	var resp *http.Response
	var lastErr error

	for attempt := 0; attempt < 3; attempt++ {
		resp, err = h.client.Do(req)

		// Always capture the last non-nil error
		if err != nil {
			lastErr = err
		}

		// Handle successful response
		if err == nil && resp != nil && resp.StatusCode == http.StatusOK {
			break
		}

		// Handle response with error status code
		if err == nil && resp != nil {
			// Use closure to ensure body is always closed
			func() {
				defer resp.Body.Close()

				// 4xx client errors are non-retryable - break immediately
				if resp.StatusCode >= 400 && resp.StatusCode < 500 {
					lastErr = fmt.Errorf("client error: status code %d", resp.StatusCode)
					return
				}

				// 5xx server errors are retryable
				if resp.StatusCode >= 500 {
					lastErr = fmt.Errorf("server error: status code %d", resp.StatusCode)
				}
			}()

			// Break immediately for 4xx errors (non-retryable)
			if resp.StatusCode >= 400 && resp.StatusCode < 500 {
				resp = nil // Clear resp so we don't try to use it later
				break
			}
		}

		// Sleep before next retry (only for retryable cases and not on last attempt)
		if attempt < 2 && (err != nil || (resp != nil && resp.StatusCode >= 500)) {
			time.Sleep(time.Duration(attempt+1) * time.Second)
		}

		// Clear resp for next iteration if it's not the successful response
		if resp != nil && (err != nil || resp.StatusCode != http.StatusOK) {
			resp = nil
		}
	}

	// Check final result
	if resp == nil || (err == nil && resp.StatusCode != http.StatusOK) {
		if lastErr != nil {
			return nil, fmt.Errorf("failed to fetch image after 3 attempts: %w", lastErr)
		}
		return nil, fmt.Errorf("failed to fetch image after 3 attempts: unknown error")
	}

	defer resp.Body.Close()

	// Memory-efficient image decoding
	img, _, err := image.Decode(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to decode image: %w", err)
	}

	return img, nil
}
