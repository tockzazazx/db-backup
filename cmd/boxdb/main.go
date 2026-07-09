package main

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"os"
	"path"
	"path/filepath"
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
	case "upload":
		err = uploadCmd()
	case "list":
		err = listCmd(os.Args[2:])
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
	folder := fs.String("folder", "", "folder (prefix) in the bucket for this machine's files, e.g. ubuntu-server-01")
	paths := fs.String("paths", "", "comma-separated local directories to upload, e.g. /var/backups,/opt/data")
	fs.Usage = func() {
		fmt.Fprintln(os.Stderr, "Usage: boxdb config --endpoint <host:port> --access <key> --secret <key> --bucket <name> [--ssl] [--folder <name>] [--paths <dir,dir>]")
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
	if *folder != "" {
		cfg.Folder = strings.Trim(*folder, "/")
	}
	if *paths != "" {
		cfg.Paths = splitPaths(*paths)
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
	fmt.Println("  folder   :", valueOr(cfg.Folder, "(not set)"))
	fmt.Println("  paths    :", valueOr(strings.Join(cfg.Paths, ", "), "(not set)"))
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

	client, err := s3.New(cfg)
	if err != nil {
		return err
	}

	fmt.Printf("connecting to %s (bucket %q, ssl=%v)...\n", cfg.Endpoint, cfg.Bucket, cfg.UseSSL)
	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()

	if err := client.Check(ctx); err != nil {
		return err
	}
	fmt.Println("OK: connection successful, bucket is accessible")

	if cfg.Folder != "" {
		created, err := client.EnsureFolder(ctx)
		if err != nil {
			return err
		}
		if created {
			fmt.Printf("OK: folder %q created in bucket\n", cfg.Folder)
		} else {
			fmt.Printf("OK: folder %q already exists\n", cfg.Folder)
		}
	}

	for _, p := range cfg.Paths {
		info, err := os.Stat(p)
		switch {
		case err != nil:
			fmt.Printf("WARN: local path %s is not accessible: %v\n", p, err)
		case !info.IsDir():
			fmt.Printf("WARN: local path %s is not a directory\n", p)
		default:
			fmt.Printf("OK: local path %s\n", p)
		}
	}
	return nil
}

// uploadCmd sweeps the configured local paths and uploads files that have
// never been uploaded before into <folder>/<upload-date>/. Files that
// disappeared locally are never deleted from the bucket.
func uploadCmd() error {
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
	if cfg.Folder == "" {
		return errors.New("no folder configured — set one with: boxdb config --folder <name>")
	}
	if len(cfg.Paths) == 0 {
		return errors.New("no local paths configured — set them with: boxdb config --paths <dir,dir>")
	}

	client, err := s3.New(cfg)
	if err != nil {
		return err
	}

	// Quick connectivity check before doing any work; uploads themselves get
	// no deadline since dump files can be large.
	checkCtx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	err = client.Check(checkCtx)
	cancel()
	if err != nil {
		return err
	}

	ctx := context.Background()
	uploaded, err := client.UploadedNames(ctx)
	if err != nil {
		return err
	}

	dateFolder := time.Now().Format("2006-01-02")
	var nUploaded, nSkipped int
	for _, dir := range cfg.Paths {
		entries, err := os.ReadDir(dir)
		if err != nil {
			fmt.Printf("WARN: cannot read %s: %v\n", dir, err)
			continue
		}
		for _, e := range entries {
			name := e.Name()
			if e.IsDir() || strings.HasPrefix(name, ".") {
				continue
			}
			if uploaded[name] {
				fmt.Printf("skip:   %s (already uploaded)\n", name)
				nSkipped++
				continue
			}
			localPath := filepath.Join(dir, name)
			key := path.Join(cfg.Folder, dateFolder, name)
			size := ""
			if info, err := e.Info(); err == nil {
				size = " (" + humanSize(info.Size()) + ")"
			}
			fmt.Printf("upload: %s -> %s%s\n", localPath, key, size)
			if err := client.Upload(ctx, key, localPath); err != nil {
				return err
			}
			uploaded[name] = true
			nUploaded++
		}
	}

	fmt.Printf("done: %d uploaded, %d skipped\n", nUploaded, nSkipped)
	return nil
}

// listCmd shows the files stored in one date folder on S3, or the available
// date folders when called without an argument.
func listCmd(args []string) error {
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
	if cfg.Folder == "" {
		return errors.New("no folder configured — set one with: boxdb config --folder <name>")
	}

	client, err := s3.New(cfg)
	if err != nil {
		return err
	}
	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()

	if len(args) == 0 {
		folders, err := client.ListDateFolders(ctx)
		if err != nil {
			return err
		}
		if len(folders) == 0 {
			fmt.Printf("no date folders under %s/ yet — run: boxdb upload\n", cfg.Folder)
			return nil
		}
		fmt.Printf("date folders under %s/ (pick one: boxdb list <name>):\n", cfg.Folder)
		for _, f := range folders {
			fmt.Println(" ", f)
		}
		return nil
	}

	sub := args[0]
	files, err := client.ListFiles(ctx, sub)
	if err != nil {
		return err
	}
	if len(files) == 0 {
		return fmt.Errorf("no files in %s/%s — run boxdb list to see available date folders", cfg.Folder, sub)
	}

	var total int64
	fmt.Printf("%s/%s:\n", cfg.Folder, sub)
	for _, f := range files {
		fmt.Printf("  %-40s %10s  %s\n", f.Name, humanSize(f.Size), f.LastModified.Local().Format("2006-01-02 15:04:05"))
		total += f.Size
	}
	fmt.Printf("total: %d files, %s\n", len(files), humanSize(total))
	return nil
}

func humanSize(n int64) string {
	const unit = 1024
	if n < unit {
		return fmt.Sprintf("%d B", n)
	}
	div, exp := int64(unit), 0
	for m := n / unit; m >= unit; m /= unit {
		div *= unit
		exp++
	}
	return fmt.Sprintf("%.1f %cB", float64(n)/float64(div), "KMGTP"[exp])
}

// splitPaths parses a comma-separated list of directories, dropping blanks.
func splitPaths(s string) []string {
	var out []string
	for _, p := range strings.Split(s, ",") {
		if p = strings.TrimSpace(p); p != "" {
			out = append(out, p)
		}
	}
	return out
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
  boxdb upload          upload new files from the configured paths to S3
  boxdb list [date]     list files in a date folder on S3 (no arg: list date folders)
  boxdb run             run a backup
  boxdb --version       print version

Config flags:
  --endpoint <host:port>  S3 endpoint (scheme optional, https:// implies --ssl)
  --access <key>          access key
  --secret <key>          secret key
  --bucket <name>         bucket name
  --ssl                   use TLS
  --folder <name>         folder (prefix) in the bucket for this machine's files
  --paths <dir,dir>       comma-separated local directories to upload`)
}
