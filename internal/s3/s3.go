// Package s3 wraps the MinIO client for talking to S3-compatible storage.
package s3

import (
	"bytes"
	"context"
	"fmt"
	"strings"

	"github.com/minio/minio-go/v7"
	"github.com/minio/minio-go/v7/pkg/credentials"

	"github.com/punnawish/db-backup/internal/config"
)

func newClient(cfg *config.S3) (*minio.Client, error) {
	client, err := minio.New(cfg.Endpoint, &minio.Options{
		Creds:  credentials.NewStaticV4(cfg.AccessKey, cfg.SecretKey, ""),
		Secure: cfg.UseSSL,
	})
	if err != nil {
		return nil, fmt.Errorf("invalid S3 settings: %w", err)
	}
	return client, nil
}

// Check verifies that the configured S3 endpoint is reachable with the given
// credentials and that the configured bucket exists.
func Check(ctx context.Context, cfg *config.S3) error {
	client, err := newClient(cfg)
	if err != nil {
		return err
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

// EnsureFolder makes sure cfg.Folder exists in the bucket, creating it when
// absent via a zero-byte marker object with a trailing slash (the same
// convention the MinIO console uses). Returns true if it created the folder.
func EnsureFolder(ctx context.Context, cfg *config.S3) (bool, error) {
	client, err := newClient(cfg)
	if err != nil {
		return false, err
	}

	key := strings.Trim(cfg.Folder, "/") + "/"

	_, err = client.StatObject(ctx, cfg.Bucket, key, minio.StatObjectOptions{})
	if err == nil {
		return false, nil
	}
	if minio.ToErrorResponse(err).Code != "NoSuchKey" {
		return false, fmt.Errorf("check folder %q: %w", cfg.Folder, err)
	}

	if _, err := client.PutObject(ctx, cfg.Bucket, key, bytes.NewReader(nil), 0, minio.PutObjectOptions{}); err != nil {
		return false, fmt.Errorf("create folder %q: %w", cfg.Folder, err)
	}
	return true, nil
}
