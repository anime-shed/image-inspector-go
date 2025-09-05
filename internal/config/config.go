package config

import (
	"fmt"
	"net"
	"os"
	"strconv"
	"strings"
	"time"
)

type Config struct {
	Host                string
	Port                string
	RequestTimeout      time.Duration
	ImageFetchTimeout   time.Duration
	AnalysisTimeout     time.Duration
	MaxRequestBodySize  int64
}

func (c *Config) ServerAddress() string {
	// Trim any whitespace from host and port
	host := strings.TrimSpace(c.Host)
	port := strings.TrimSpace(c.Port)
	return net.JoinHostPort(host, port)
}

func LoadFromEnv() (*Config, error) {
	// Set defaults
	cfg := &Config{
		Host:                getEnvOrDefault("HOST", "0.0.0.0"),
		Port:                getEnvOrDefault("PORT", "8080"),
		RequestTimeout:      parseDurationOrDefault("REQUEST_TIMEOUT", 30*time.Second),
		ImageFetchTimeout:   parseDurationOrDefault("IMAGE_FETCH_TIMEOUT", 15*time.Second),
		AnalysisTimeout:     parseDurationOrDefault("ANALYSIS_TIMEOUT", 20*time.Second),
		MaxRequestBodySize:  parseIntOrDefault("MAX_REQUEST_BODY_SIZE", 10*1024*1024), // 10MB
	}

	// Validate port is numeric and in range
	p, err := strconv.Atoi(strings.TrimSpace(cfg.Port))
	if err != nil || p < 1 || p > 65535 {
		return nil, fmt.Errorf("invalid PORT: %q", cfg.Port)
	}
	if cfg.MaxRequestBodySize <= 0 {
		return nil, fmt.Errorf("MAX_REQUEST_BODY_SIZE must be > 0 (got %d)", cfg.MaxRequestBodySize)
	}
	if cfg.RequestTimeout <= 0 || cfg.ImageFetchTimeout <= 0 || cfg.AnalysisTimeout <= 0 {
		return nil, fmt.Errorf("timeouts must be > 0 (got request=%s, fetch=%s, analysis=%s)",
			cfg.RequestTimeout, cfg.ImageFetchTimeout, cfg.AnalysisTimeout)
	}
	return cfg, nil
}

func getEnvOrDefault(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}

func parseDurationOrDefault(key string, defaultValue time.Duration) time.Duration {
	if value := os.Getenv(key); value != "" {
		if duration, err := time.ParseDuration(strings.TrimSpace(value)); err == nil && duration > 0 {
			return duration
		}
	}
	return defaultValue
}

func parseIntOrDefault(key string, defaultValue int64) int64 {
	if value := os.Getenv(key); value != "" {
		if intValue, err := strconv.ParseInt(strings.TrimSpace(value), 10, 64); err == nil {
			return intValue
		}
	}
	return defaultValue
}