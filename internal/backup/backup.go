// Package backup contains the core backup logic.
package backup

import "context"

// Runner performs database backups.
type Runner struct {
	databaseURL string
	outputDir   string
}

// New creates a Runner for the given database and output directory.
func New(databaseURL, outputDir string) *Runner {
	return &Runner{databaseURL: databaseURL, outputDir: outputDir}
}

// Run executes a single backup. Implementation TBD.
func (r *Runner) Run(ctx context.Context) error {
	// TODO: implement backup (e.g. pg_dump / mysqldump wrapper)
	return nil
}
