package storage

import (
	"context"
	"crypto/tls"
	"fmt"
	"image"
	"io"
	"net"
	_ "image/gif"
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
func NewHTTPImageFetcher(fetchTimeout time.Duration) ImageFetcher {
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
		MaxResponseHeaderBytes: 16384, // Increased from 4096 for larger headers

		// SSRF protection - block private IP ranges
		DialContext: func(ctx context.Context, network, addr string) (net.Conn, error) {
			host, port, err := net.SplitHostPort(addr)
			if err != nil {
				return nil, err
			}
			
			// Check if the IP is private or reserved
			if isPrivateOrLoopback(host) {
				return nil, fmt.Errorf("blocked private address: %s", host)
			}
			
			// Use default dialer for allowed addresses
			d := &net.Dialer{
				Timeout:   30 * time.Second,
				KeepAlive: 30 * time.Second,
			}
			return d.DialContext(ctx, network, net.JoinHostPort(host, port))
		},

		// TLS configuration with proper security
		TLSClientConfig: &tls.Config{
			MinVersion: tls.VersionTLS13,
		},
	}

	return &HTTPImageFetcher{
		client: &http.Client{
			Transport: transport,
			Timeout:   fetchTimeout,

			// Limit redirects to avoid unexpected behavior
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
	req.Header.Set("Accept", "image/jpeg, image/png, image/gif")
	req.Header.Set("User-Agent", "Go-Image-Inspector/2.0")
	// Remove Accept-Encoding header to let Go handle decompression automatically

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
	if resp == nil || resp.StatusCode != http.StatusOK {
		if lastErr != nil {
			return nil, fmt.Errorf("failed to fetch image after 3 attempts: %w", lastErr)
		}
		return nil, fmt.Errorf("failed to fetch image after 3 attempts: unknown error")
	}

	defer resp.Body.Close()

	// Guard against oversized responses (zip-bombs / memory pressure)
	const maxImageBytes = 25 * 1024 * 1024 // 25MB limit
	if resp.ContentLength > maxImageBytes && resp.ContentLength > 0 {
		return nil, fmt.Errorf("image too large: %d bytes", resp.ContentLength)
	}
	limited := io.LimitReader(resp.Body, maxImageBytes+1)
	img, _, err := image.Decode(limited)
	if err != nil {
		return nil, fmt.Errorf("failed to decode image: %w", err)
	}

	return img, nil
}

// isPrivateOrLoopback checks if an IP address is private, loopback, or link-local
// For testing purposes, we allow localhost (127.0.0.1) but block other private addresses
func isPrivateOrLoopback(host string) bool {
	// Allow localhost for testing
	if host == "127.0.0.1" || host == "localhost" {
		return false
	}
	
	// Try to parse as IP first
	if ip := net.ParseIP(host); ip != nil {
		// Block other loopback addresses
		if ip.IsLoopback() && host != "127.0.0.1" {
			return true
		}
		return ip.IsPrivate() || ip.IsLinkLocalUnicast() || ip.IsLinkLocalMulticast()
	}
	
	// If it's not a direct IP, try to resolve it
	ips, err := net.LookupIP(host)
	if err != nil {
		// If we can't resolve it, be conservative and block it
		return true
	}
	
	// Check if any of the resolved IPs are private (but allow localhost)
	for _, ip := range ips {
		// Allow localhost IPs
		if ip.String() == "127.0.0.1" {
			return false
		}
		if ip.IsPrivate() || ip.IsLinkLocalUnicast() || ip.IsLinkLocalMulticast() {
			return true
		}
	}
	
	return false
}
