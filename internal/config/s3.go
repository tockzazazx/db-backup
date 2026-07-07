package config

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// ErrNoConfig is returned when no S3 config file has been created yet.
var ErrNoConfig = errors.New("no S3 config found")

// S3 holds credentials and connection settings for the backup target.
type S3 struct {
	Endpoint  string `json:"endpoint"`
	AccessKey string `json:"access_key"`
	SecretKey string `json:"secret_key"`
	Bucket    string `json:"bucket"`
	UseSSL    bool   `json:"use_ssl"`
	// Folder is the prefix inside the bucket where this machine's files go.
	Folder string `json:"folder,omitempty"`
	// Paths are local directories whose files get uploaded into Folder.
	Paths []string `json:"paths,omitempty"`
}

// S3ConfigPath returns the per-user config file location following the XDG
// convention: ~/.config/boxdb/config.json on Linux (respects XDG_CONFIG_HOME).
func S3ConfigPath() (string, error) {
	dir, err := os.UserConfigDir()
	if err != nil {
		return "", fmt.Errorf("locate user config dir: %w", err)
	}
	return filepath.Join(dir, "boxdb", "config.json"), nil
}

// LoadS3 reads the S3 config from disk. Returns ErrNoConfig if absent.
func LoadS3() (*S3, error) {
	path, err := S3ConfigPath()
	if err != nil {
		return nil, err
	}
	data, err := os.ReadFile(path)
	if errors.Is(err, os.ErrNotExist) {
		return nil, fmt.Errorf("%w (expected at %s)", ErrNoConfig, path)
	}
	if err != nil {
		return nil, fmt.Errorf("read config: %w", err)
	}
	var cfg S3
	if err := json.Unmarshal(data, &cfg); err != nil {
		return nil, fmt.Errorf("parse config %s: %w", path, err)
	}
	return &cfg, nil
}

// Save writes the config next to the binary with owner-only permissions,
// since it contains the secret key.
func (s *S3) Save() (string, error) {
	path, err := S3ConfigPath()
	if err != nil {
		return "", err
	}
	data, err := json.MarshalIndent(s, "", "  ")
	if err != nil {
		return "", err
	}
	if err := os.MkdirAll(filepath.Dir(path), 0o700); err != nil {
		return "", fmt.Errorf("create config dir: %w", err)
	}
	if err := os.WriteFile(path, append(data, '\n'), 0o600); err != nil {
		return "", fmt.Errorf("write config: %w", err)
	}
	return path, nil
}

// Validate reports which required fields are missing.
func (s *S3) Validate() error {
	var missing []string
	if s.Endpoint == "" {
		missing = append(missing, "endpoint")
	}
	if s.AccessKey == "" {
		missing = append(missing, "access key")
	}
	if s.SecretKey == "" {
		missing = append(missing, "secret key")
	}
	if s.Bucket == "" {
		missing = append(missing, "bucket")
	}
	if len(missing) > 0 {
		return fmt.Errorf("incomplete S3 config: missing %s (run: boxdb config)", strings.Join(missing, ", "))
	}
	return nil
}
