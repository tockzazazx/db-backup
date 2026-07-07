package main

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/punnawish/db-backup/internal/backup"
	"github.com/punnawish/db-backup/internal/config"
	"github.com/punnawish/db-backup/internal/s3"
)

// Set at build time via -ldflags "-X main.version=... -X main.commit=... -X main.date=..."
var (
	version = "dev"
	commit  = "none"
	date    = "unknown"
)

func main() {
	if len(os.Args) < 2 {
		usage()
		return
	}

	var err error
	switch os.Args[1] {
	case "--version", "-v", "version":
		fmt.Printf("boxdb version %s (commit %s, built %s)\n", version, commit, date)
	case "config":
		err = configCmd(os.Args[2:])
	case "test":
		err = testCmd()
	case "run":
		err = runCmd()
	default:
		fmt.Fprintf(os.Stderr, "unknown command: %s\n\n", os.Args[1])
		usage()
		os.Exit(1)
	}
	if err != nil {
		fmt.Fprintln(os.Stderr, "error:", err)
		os.Exit(1)
	}
}

// configCmd saves S3 credentials, or shows the current config when called
// without flags.
func configCmd(args []string) error {
	fs := flag.NewFlagSet("config", flag.ExitOnError)
	endpoint := fs.String("endpoint", "", "S3 endpoint, e.g. s3.example.com:9000 or https://s3.example.com")
	access := fs.String("access", "", "S3 access key")
	secret := fs.String("secret", "", "S3 secret key")
	bucket := fs.String("bucket", "", "S3 bucket name")
	ssl := fs.Bool("ssl", false, "use TLS (auto-detected when endpoint has an http:// or https:// scheme)")
	fs.Usage = func() {
		fmt.Fprintln(os.Stderr, "Usage: boxdb config --endpoint <host:port> --access <key> --secret <key> --bucket <name> [--ssl]")
		fmt.Fprintln(os.Stderr, "       boxdb config          (show current config)")
		fs.PrintDefaults()
	}
	if err := fs.Parse(args); err != nil {
		return err
	}

	if fs.NFlag() == 0 {
		return showConfig()
	}

	// Start from the existing config so single fields can be updated.
	cfg, err := config.LoadS3()
	if err != nil {
		if !errors.Is(err, config.ErrNoConfig) {
			return err
		}
		cfg = &config.S3{}
	}

	if *endpoint != "" {
		host, useSSL := normalizeEndpoint(*endpoint, *ssl)
		cfg.Endpoint = host
		cfg.UseSSL = useSSL
	} else if isFlagSet(fs, "ssl") {
		cfg.UseSSL = *ssl
	}
	if *access != "" {
		cfg.AccessKey = *access
	}
	if *secret != "" {
		cfg.SecretKey = *secret
	}
	if *bucket != "" {
		cfg.Bucket = *bucket
	}

	path, err := cfg.Save()
	if err != nil {
		return err
	}
	fmt.Println("config saved to", path)
	return nil
}

func showConfig() error {
	cfg, err := config.LoadS3()
	if err != nil {
		return err
	}
	path, _ := config.S3ConfigPath()
	fmt.Println("config file:", path)
	fmt.Println("  endpoint :", valueOr(cfg.Endpoint, "(not set)"))
	fmt.Println("  access   :", valueOr(cfg.AccessKey, "(not set)"))
	fmt.Println("  secret   :", maskSecret(cfg.SecretKey))
	fmt.Println("  bucket   :", valueOr(cfg.Bucket, "(not set)"))
	fmt.Println("  ssl      :", cfg.UseSSL)
	return nil
}

// testCmd checks connectivity to the configured S3 endpoint.
func testCmd() error {
	cfg, err := config.LoadS3()
	if err != nil {
		if errors.Is(err, config.ErrNoConfig) {
			return fmt.Errorf("%v\nconfigure credentials first:\n  boxdb config --endpoint <host:port> --access <key> --secret <key> --bucket <name>", err)
		}
		return err
	}
	if err := cfg.Validate(); err != nil {
		return err
	}

	fmt.Printf("connecting to %s (bucket %q, ssl=%v)...\n", cfg.Endpoint, cfg.Bucket, cfg.UseSSL)
	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()

	if err := s3.Check(ctx, cfg); err != nil {
		return err
	}
	fmt.Println("OK: connection successful, bucket is accessible")
	return nil
}

func runCmd() error {
	cfg, err := config.Load()
	if err != nil {
		return err
	}
	fmt.Printf("boxdb starting (database: %s, output: %s)\n", cfg.DatabaseURL, cfg.OutputDir)
	return backup.New(cfg.DatabaseURL, cfg.OutputDir).Run(context.Background())
}

// normalizeEndpoint strips an optional scheme; the scheme wins over the --ssl
// flag when present, since minio-go expects a bare host:port.
func normalizeEndpoint(endpoint string, ssl bool) (string, bool) {
	switch {
	case strings.HasPrefix(endpoint, "https://"):
		return strings.TrimSuffix(strings.TrimPrefix(endpoint, "https://"), "/"), true
	case strings.HasPrefix(endpoint, "http://"):
		return strings.TrimSuffix(strings.TrimPrefix(endpoint, "http://"), "/"), false
	default:
		return strings.TrimSuffix(endpoint, "/"), ssl
	}
}

func isFlagSet(fs *flag.FlagSet, name string) bool {
	set := false
	fs.Visit(func(f *flag.Flag) {
		if f.Name == name {
			set = true
		}
	})
	return set
}

func maskSecret(s string) string {
	if s == "" {
		return "(not set)"
	}
	if len(s) <= 4 {
		return "****"
	}
	return s[:2] + strings.Repeat("*", len(s)-4) + s[len(s)-2:]
}

func valueOr(s, fallback string) string {
	if s == "" {
		return fallback
	}
	return s
}

func usage() {
	fmt.Println(`boxdb — database backup tool

Usage:
  boxdb config [flags]  save S3 credentials (no flags: show current config)
  boxdb test            test the S3 connection using the saved config
  boxdb run             run a backup
  boxdb --version       print version

Config flags:
  --endpoint <host:port>  S3 endpoint (scheme optional, https:// implies --ssl)
  --access <key>          access key
  --secret <key>          secret key
  --bucket <name>         bucket name
  --ssl                   use TLS`)
}
