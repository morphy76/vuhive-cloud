package outbound

import (
	"context"
	"io"
	"time"
)

// ObjectInfo holds metadata about an object in storage.
type ObjectInfo struct {
	Key          string
	Size         int64
	LastModified time.Time
}

// LifecycleRule defines an expiration rule for storage objects with a prefix.
type LifecycleRule struct {
	ID             string
	Prefix         string
	ExpirationDays int
	Enabled        bool
}

// StoragePort defines the driven port for S3-compatible object storage.
type StoragePort interface {
	Upload(ctx context.Context, key string, content io.Reader, size int64, contentType string) error
	Download(ctx context.Context, key string) (io.ReadCloser, error)
	Delete(ctx context.Context, key string) error
	Exists(ctx context.Context, key string) (bool, error)
	PresignDownload(ctx context.Context, key string, lifetime time.Duration) (string, error)
	PresignUpload(ctx context.Context, key string, lifetime time.Duration) (string, error)
	EnsureBucket(ctx context.Context) error
	ListObjects(ctx context.Context, prefix string) ([]ObjectInfo, error)
	PutBucketLifecycleConfiguration(ctx context.Context, rules []LifecycleRule) error
}
