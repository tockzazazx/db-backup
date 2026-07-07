package main

import (
	"context"
	"fmt"
	"os"

	"github.com/punnawish/db-backup/internal/backup"
	"github.com/punnawish/db-backup/internal/config"
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

	switch os.Args[1] {
	case "--version", "-v", "version":
		fmt.Printf("boxdb version %s (commit %s, built %s)\n", version, commit, date)
	case "run":
		if err := run(); err != nil {
			fmt.Fprintln(os.Stderr, "error:", err)
			os.Exit(1)
		}
	default:
		fmt.Fprintf(os.Stderr, "unknown command: %s\n\n", os.Args[1])
		usage()
		os.Exit(1)
	}
}

func run() error {
	cfg, err := config.Load()
	if err != nil {
		return err
	}
	fmt.Printf("boxdb starting (database: %s, output: %s)\n", cfg.DatabaseURL, cfg.OutputDir)
	return backup.New(cfg.DatabaseURL, cfg.OutputDir).Run(context.Background())
}

func usage() {
	fmt.Println(`boxdb — database backup tool

Usage:
  boxdb run        run a backup
  boxdb --version  print version`)
}
