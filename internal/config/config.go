// Package config loads application configuration from the environment.
package config

import "os"

// Config holds runtime configuration for db-backup.
type Config struct {
	// DatabaseURL is the connection string of the database to back up.
	DatabaseURL string
	// OutputDir is the directory where backup files are written.
	OutputDir string
}

// Load reads configuration from environment variables, applying defaults.
func Load() (*Config, error) {
	cfg := &Config{
		DatabaseURL: getEnv("DB_URL", "postgres://localhost:5432/postgres"),
		OutputDir:   getEnv("BACKUP_OUTPUT_DIR", "./backups"),
	}
	return cfg, nil
}

func getEnv(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}
