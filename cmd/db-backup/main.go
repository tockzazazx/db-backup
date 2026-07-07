package main

import (
	"fmt"
	"os"

	"github.com/punnawish/db-backup/internal/config"
)

func main() {
	cfg, err := config.Load()
	if err != nil {
		fmt.Fprintln(os.Stderr, "error:", err)
		os.Exit(1)
	}

	fmt.Printf("db-backup starting (database: %s)\n", cfg.DatabaseURL)
}
