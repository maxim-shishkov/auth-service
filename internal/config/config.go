package config

import (
	"encoding/json"
	"fmt"
	"os"
)

type Config struct {
	Port        int    `json:"port"`
	PostgresDSN string `json:"dsn"`
	JWTSecret   string `json:"-"`
	JWTTTL      string `json:"jwt_ttl"`
}

func Load(path string) (Config, error) {
	file, err := os.Open(path)
	if err != nil {
		return Config{}, fmt.Errorf("os open: %w", err)
	}
	defer func() {
		_ = file.Close()
	}()

	var cfg Config
	if err := json.NewDecoder(file).Decode(&cfg); err != nil {
		return Config{}, fmt.Errorf("json decoder: %w", err)
	}

	cfg.JWTSecret = os.Getenv("JWT_SECRET")
	if cfg.JWTSecret == "" {
		return Config{}, fmt.Errorf("JWT_SECRET environment variable is required")
	}

	if len(cfg.JWTSecret) < 32 {
		return Config{}, fmt.Errorf("JWT_SECRET must be at least 32 characters")

	}

	return cfg, nil
}
