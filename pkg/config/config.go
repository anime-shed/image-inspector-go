package config

import (
	"fmt"
	"os"
)

type Config struct {
	Host string
	Port string
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

	return cfg, nil
}
