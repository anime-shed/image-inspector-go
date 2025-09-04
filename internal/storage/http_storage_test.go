package storage

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

func TestHTTPImageFetcher_RetryLogic(t *testing.T) {
	tests := []struct {
		name          string
		responses     []int // Status codes to return in sequence
		expectRetries int   // Expected number of requests
		expectError   bool
		errorContains string
	}{
		{
			name:          "Success on first attempt",
			responses:     []int{200},
			expectRetries: 1,
			expectError:   false,
		},
		{
			name:          "Success on second attempt after 5xx",
			responses:     []int{500, 200},
			expectRetries: 2,
			expectError:   false,
		},
		{
			name:          "4xx client error - no retry",
			responses:     []int{404},
			expectRetries: 1,
			expectError:   true,
			errorContains: "client error: status code 404",
		},
		{
			name:          "4xx after 5xx - should retry until 4xx then stop",
			responses:     []int{500, 404},
			expectRetries: 2,
			expectError:   true,
			errorContains: "client error: status code 404",
		},
		{
			name:          "All 5xx errors - retry all attempts",
			responses:     []int{500, 502, 503},
			expectRetries: 3,
			expectError:   true,
			errorContains: "server error: status code 503",
		},
		{
			name:          "Mixed 4xx errors - stop on first 4xx",
			responses:     []int{400},
			expectRetries: 1,
			expectError:   true,
			errorContains: "client error: status code 400",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			requestCount := 0

			// Create test server that returns responses in sequence
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				if requestCount < len(tt.responses) {
					statusCode := tt.responses[requestCount]
					requestCount++

					if statusCode == 200 {
						// Return a valid minimal PNG image (1x1 pixel)
						w.Header().Set("Content-Type", "image/png")
						// Valid minimal PNG data for 1x1 transparent pixel
						pngData := []byte{
							0x89, 0x50, 0x4E, 0x47, 0x0D, 0x0A, 0x1A, 0x0A, // PNG signature
							0x00, 0x00, 0x00, 0x0D, 0x49, 0x48, 0x44, 0x52, // IHDR chunk
							0x00, 0x00, 0x00, 0x01, 0x00, 0x00, 0x00, 0x01, // 1x1 dimensions
							0x08, 0x06, 0x00, 0x00, 0x00, 0x1F, 0x15, 0xC4, // bit depth, color type, etc.
							0x89, 0x00, 0x00, 0x00, 0x0A, 0x49, 0x44, 0x41, // IDAT chunk start
							0x54, 0x78, 0x9C, 0x63, 0x00, 0x01, 0x00, 0x00, // compressed data
							0x05, 0x00, 0x01, 0x0D, 0x0A, 0x2D, 0xB4, 0x00, // compressed data end
							0x00, 0x00, 0x00, 0x49, 0x45, 0x4E, 0x44, 0xAE, // IEND chunk
							0x42, 0x60, 0x82,
						}
						w.Write(pngData)
					} else {
						w.WriteHeader(statusCode)
						w.Write([]byte(fmt.Sprintf("Error %d", statusCode)))
					}
				} else {
					// Shouldn't happen in our tests
					w.WriteHeader(500)
					w.Write([]byte("Unexpected request"))
				}
			}))
			defer server.Close()

			// Create fetcher
			fetcher := NewHTTPImageFetcher()

			// Test the fetch
			ctx := context.Background()
			_, err := fetcher.FetchImage(ctx, server.URL)

			// Verify request count
			if requestCount != tt.expectRetries {
				t.Errorf("Expected %d requests, got %d", tt.expectRetries, requestCount)
			}

			// Verify error expectation
			if tt.expectError {
				if err == nil {
					t.Errorf("Expected error, but got none")
				} else if tt.errorContains != "" && !strings.Contains(err.Error(), tt.errorContains) {
					t.Errorf("Expected error to contain '%s', got: %s", tt.errorContains, err.Error())
				}
			} else {
				if err != nil {
					t.Errorf("Expected no error, got: %s", err.Error())
				}
			}
		})
	}
}

func TestHTTPImageFetcher_NetworkError_Retry(t *testing.T) {
	// Test that network errors are retried
	requestCount := 0

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		requestCount++
		if requestCount < 3 {
			// Simulate network error by closing connection
			hj, ok := w.(http.Hijacker)
			if ok {
				conn, _, _ := hj.Hijack()
				conn.Close()
			}
			return
		}
		// Success on third attempt
		w.Header().Set("Content-Type", "image/png")
		// Valid minimal PNG data for 1x1 transparent pixel
		pngData := []byte{
			0x89, 0x50, 0x4E, 0x47, 0x0D, 0x0A, 0x1A, 0x0A, // PNG signature
			0x00, 0x00, 0x00, 0x0D, 0x49, 0x48, 0x44, 0x52, // IHDR chunk
			0x00, 0x00, 0x00, 0x01, 0x00, 0x00, 0x00, 0x01, // 1x1 dimensions
			0x08, 0x06, 0x00, 0x00, 0x00, 0x1F, 0x15, 0xC4, // bit depth, color type, etc.
			0x89, 0x00, 0x00, 0x00, 0x0A, 0x49, 0x44, 0x41, // IDAT chunk start
			0x54, 0x78, 0x9C, 0x63, 0x00, 0x01, 0x00, 0x00, // compressed data
			0x05, 0x00, 0x01, 0x0D, 0x0A, 0x2D, 0xB4, 0x00, // compressed data end
			0x00, 0x00, 0x00, 0x49, 0x45, 0x4E, 0x44, 0xAE, // IEND chunk
			0x42, 0x60, 0x82,
		}
		w.Write(pngData)
	}))
	defer server.Close()

	fetcher := NewHTTPImageFetcher()
	ctx := context.Background()

	start := time.Now()
	_, err := fetcher.FetchImage(ctx, server.URL)
	duration := time.Since(start)

	// Should succeed after retries
	if err != nil {
		t.Errorf("Expected success after retries, got error: %s", err.Error())
	}

	// Should have made 3 requests
	if requestCount != 3 {
		t.Errorf("Expected 3 requests, got %d", requestCount)
	}

	// Should have taken at least 3 seconds due to backoff (1s + 2s)
	if duration < 3*time.Second {
		t.Errorf("Expected at least 3 seconds due to backoff, took %v", duration)
	}
}
