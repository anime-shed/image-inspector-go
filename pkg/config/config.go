package config

import (
	"fmt"
	"os"
	"strconv"
	"time"
)

type Config struct {
	Host               string
	Port               string
	RequestTimeout     time.Duration
	ImageFetchTimeout  time.Duration
	AnalysisTimeout    time.Duration
	MaxRequestBodySize int64
}

func (c *Config) ServerAddress() string {
	return fmt.Sprintf("%s:%s", c.Host, c.Port)
}

func LoadFromEnv() (*Config, error) {
	// Set defaults
	cfg := &Config{
		Host:               getEnvOrDefault("HOST", "0.0.0.0"),
		Port:               getEnvOrDefault("PORT", "8080"),
		RequestTimeout:     parseDurationOrDefault("REQUEST_TIMEOUT", 30*time.Second),
		ImageFetchTimeout:  parseDurationOrDefault("IMAGE_FETCH_TIMEOUT", 15*time.Second),
		AnalysisTimeout:    parseDurationOrDefault("ANALYSIS_TIMEOUT", 20*time.Second),
		MaxRequestBodySize: parseIntOrDefault("MAX_REQUEST_BODY_SIZE", 10*1024*1024), // 10MB
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
		if duration, err := time.ParseDuration(value); err == nil {
			return duration
		}
	}
	return defaultValue
}

func parseIntOrDefault(key string, defaultValue int64) int64 {
	if value := os.Getenv(key); value != "" {
		if intValue, err := strconv.ParseInt(value, 10, 64); err == nil {
			return intValue
		}
	}
	return defaultValue
}
