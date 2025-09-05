package storage

import (
	"context"
	"crypto/tls"
	"fmt"
	"image"
	"io"
	"net"
	"net/url"
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

		// SSRF protection - resolve with context, dial vetted IP, and verify final remote IP
		DialContext: func(ctx context.Context, network, addr string) (net.Conn, error) {
			host, port, err := net.SplitHostPort(addr)
			if err != nil {
				return nil, err
			}
			ips, err := net.DefaultResolver.LookupIPAddr(ctx, host)
			if err != nil {
				return nil, fmt.Errorf("dns lookup failed: %w", err)
			}
			var target net.IP
			for _, ipa := range ips {
				if isPrivateOrLoopback(ipa.IP.String()) {
					return nil, fmt.Errorf("blocked private address: %s", ipa.IP.String())
				}
				if target == nil {
					target = ipa.IP
				}
			}
			if target == nil {
				return nil, fmt.Errorf("no public IPs found for host %q", host)
			}
			d := &net.Dialer{
				Timeout:   30 * time.Second,
				KeepAlive: 30 * time.Second,
			}
			c, err := d.DialContext(ctx, network, net.JoinHostPort(target.String(), port))
			if err != nil {
				return nil, err
			}
			if ra, ok := c.RemoteAddr().(*net.TCPAddr); ok && ra != nil && isPrivateOrLoopback(ra.IP.String()) {
				_ = c.Close()
				return nil, fmt.Errorf("blocked private address after dial: %s", ra.IP.String())
			}
			return c, nil
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

			// Limit redirects and validate redirect URLs to prevent SSRF via redirects
			CheckRedirect: func(req *http.Request, via []*http.Request) error {
				if len(via) >= 3 {
					return fmt.Errorf("too many redirects (limit: 3)")
				}
				// Validate redirect URL to prevent SSRF
				if req.URL.Scheme != "http" && req.URL.Scheme != "https" {
					return fmt.Errorf("invalid redirect scheme: %s", req.URL.Scheme)
				}
				if req.URL.Host == "" {
					return fmt.Errorf("invalid redirect: missing host")
				}
				return nil
			},
		},
	}
}

func (h *HTTPImageFetcher) FetchImage(ctx context.Context, imageURL string) (image.Image, error) {
	// Validate URL scheme and host before making any requests
	u, err := url.Parse(imageURL)
	if err != nil || (u.Scheme != "http" && u.Scheme != "https") || u.Host == "" {
		return nil, fmt.Errorf("invalid URL: only http/https with host are allowed")
	}
	
	req, err := http.NewRequestWithContext(ctx, "GET", u.String(), nil)
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
		if err != nil {
			if ctx.Err() != nil { // cancelled or deadline exceeded
				lastErr = ctx.Err()
				break
			}
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

// isPrivateOrLoopback reports whether the given IP (string form) is non-public.
// Expect a literal IP string; DNS resolution is handled by the dialer.
func isPrivateOrLoopback(host string) bool {
	ip := net.ParseIP(host)
	if ip == nil {
		// Be conservative; callers should pass literal IPs.
		return true
	}
	return ip.IsLoopback() || ip.IsUnspecified() || ip.IsPrivate() || ip.IsLinkLocalUnicast() || ip.IsLinkLocalMulticast()
}
