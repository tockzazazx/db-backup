// Package s3 wraps the MinIO client for talking to S3-compatible storage.
package s3

import (
	"context"
	"fmt"

	"github.com/minio/minio-go/v7"
	"github.com/minio/minio-go/v7/pkg/credentials"

	"github.com/punnawish/db-backup/internal/config"
)

// Check verifies that the configured S3 endpoint is reachable with the given
// credentials and that the configured bucket exists.
func Check(ctx context.Context, cfg *config.S3) error {
	client, err := minio.New(cfg.Endpoint, &minio.Options{
		Creds:  credentials.NewStaticV4(cfg.AccessKey, cfg.SecretKey, ""),
		Secure: cfg.UseSSL,
	})
	if err != nil {
		return fmt.Errorf("invalid S3 settings: %w", err)
	}

	exists, err := client.BucketExists(ctx, cfg.Bucket)
	if err != nil {
		return fmt.Errorf("cannot connect to %s: %w", cfg.Endpoint, err)
	}
	if !exists {
		return fmt.Errorf("connected to %s, but bucket %q does not exist or is not accessible", cfg.Endpoint, cfg.Bucket)
	}
	return nil
}
