// Package s3 wraps the MinIO client for talking to S3-compatible storage.
package s3

import (
	"bytes"
	"context"
	"fmt"
	"path"
	"sort"
	"strings"
	"time"

	"github.com/minio/minio-go/v7"
	"github.com/minio/minio-go/v7/pkg/credentials"

	"github.com/punnawish/db-backup/internal/config"
)

// Client talks to one configured S3 endpoint/bucket.
type Client struct {
	mc  *minio.Client
	cfg *config.S3
}

// New builds a client from the saved config.
func New(cfg *config.S3) (*Client, error) {
	mc, err := minio.New(cfg.Endpoint, &minio.Options{
		Creds:  credentials.NewStaticV4(cfg.AccessKey, cfg.SecretKey, ""),
		Secure: cfg.UseSSL,
	})
	if err != nil {
		return nil, fmt.Errorf("invalid S3 settings: %w", err)
	}
	return &Client{mc: mc, cfg: cfg}, nil
}

// Check verifies that the endpoint is reachable with the given credentials
// and that the configured bucket exists.
func (c *Client) Check(ctx context.Context) error {
	exists, err := c.mc.BucketExists(ctx, c.cfg.Bucket)
	if err != nil {
		return fmt.Errorf("cannot connect to %s: %w", c.cfg.Endpoint, err)
	}
	if !exists {
		return fmt.Errorf("connected to %s, but bucket %q does not exist or is not accessible", c.cfg.Endpoint, c.cfg.Bucket)
	}
	return nil
}

// EnsureFolder makes sure cfg.Folder exists in the bucket, creating it when
// absent via a zero-byte marker object with a trailing slash (the same
// convention the MinIO console uses). Returns true if it created the folder.
func (c *Client) EnsureFolder(ctx context.Context) (bool, error) {
	key := strings.Trim(c.cfg.Folder, "/") + "/"

	_, err := c.mc.StatObject(ctx, c.cfg.Bucket, key, minio.StatObjectOptions{})
	if err == nil {
		return false, nil
	}
	if minio.ToErrorResponse(err).Code != "NoSuchKey" {
		return false, fmt.Errorf("check folder %q: %w", c.cfg.Folder, err)
	}

	if _, err := c.mc.PutObject(ctx, c.cfg.Bucket, key, bytes.NewReader(nil), 0, minio.PutObjectOptions{}); err != nil {
		return false, fmt.Errorf("create folder %q: %w", c.cfg.Folder, err)
	}
	return true, nil
}

// UploadedNames returns the base names of every object already stored under
// cfg.Folder, across all date subfolders. Used to skip files that were
// uploaded on a previous run, whichever day they went up.
func (c *Client) UploadedNames(ctx context.Context) (map[string]bool, error) {
	prefix := strings.Trim(c.cfg.Folder, "/") + "/"
	names := make(map[string]bool)
	for obj := range c.mc.ListObjects(ctx, c.cfg.Bucket, minio.ListObjectsOptions{Prefix: prefix, Recursive: true}) {
		if obj.Err != nil {
			return nil, fmt.Errorf("list %s: %w", prefix, obj.Err)
		}
		if strings.HasSuffix(obj.Key, "/") { // folder marker objects
			continue
		}
		names[path.Base(obj.Key)] = true
	}
	return names, nil
}

// Object describes one stored file.
type Object struct {
	Name         string
	Size         int64
	LastModified time.Time
}

// ListFiles returns the files stored under cfg.Folder/<sub>/, sorted by name.
func (c *Client) ListFiles(ctx context.Context, sub string) ([]Object, error) {
	prefix := path.Join(strings.Trim(c.cfg.Folder, "/"), strings.Trim(sub, "/")) + "/"
	var files []Object
	for obj := range c.mc.ListObjects(ctx, c.cfg.Bucket, minio.ListObjectsOptions{Prefix: prefix, Recursive: true}) {
		if obj.Err != nil {
			return nil, fmt.Errorf("list %s: %w", prefix, obj.Err)
		}
		if strings.HasSuffix(obj.Key, "/") { // folder marker objects
			continue
		}
		files = append(files, Object{
			Name:         strings.TrimPrefix(obj.Key, prefix),
			Size:         obj.Size,
			LastModified: obj.LastModified,
		})
	}
	sort.Slice(files, func(i, j int) bool { return files[i].Name < files[j].Name })
	return files, nil
}

// ListDateFolders returns the subfolder names directly under cfg.Folder,
// sorted ascending (upload dates sort chronologically).
func (c *Client) ListDateFolders(ctx context.Context) ([]string, error) {
	prefix := strings.Trim(c.cfg.Folder, "/") + "/"
	var folders []string
	for obj := range c.mc.ListObjects(ctx, c.cfg.Bucket, minio.ListObjectsOptions{Prefix: prefix, Recursive: false}) {
		if obj.Err != nil {
			return nil, fmt.Errorf("list %s: %w", prefix, obj.Err)
		}
		if name := strings.Trim(strings.TrimPrefix(obj.Key, prefix), "/"); name != "" && strings.HasSuffix(obj.Key, "/") {
			folders = append(folders, name)
		}
	}
	sort.Strings(folders)
	return folders, nil
}

// Upload puts a local file into the bucket at key.
func (c *Client) Upload(ctx context.Context, key, localPath string) error {
	if _, err := c.mc.FPutObject(ctx, c.cfg.Bucket, key, localPath, minio.PutObjectOptions{}); err != nil {
		return fmt.Errorf("upload %s: %w", localPath, err)
	}
	return nil
}
