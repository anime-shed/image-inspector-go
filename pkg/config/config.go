package config

import (
	"fmt"
	"os"
	"strconv"
)

type Config struct {
	Host          string
	Port          string
	SkipTLSVerify bool
}

func (c *Config) ServerAddress() string {
	return fmt.Sprintf("%s:%s", c.Host, c.Port)
}

func LoadFromEnv() (*Config, error) {
	// Set defaults
	cfg := &Config{
		Host: os.Getenv("HOST"),
		Port: os.Getenv("PORT"),
	}

	if cfg.Host == "" {
		cfg.Host = "0.0.0.0"
	}
	if cfg.Port == "" {
		cfg.Port = "8080"
	}

	// Parse TLS skip verification setting
	if skipTLS := os.Getenv("SKIP_TLS_VERIFY"); skipTLS != "" {
		if parsed, err := strconv.ParseBool(skipTLS); err == nil {
			cfg.SkipTLSVerify = parsed
		}
	}

	return cfg, nil
}
