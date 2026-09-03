package outbound

import (
	"context"
	"io"
	"time"
)

// StoragePort defines the driven port for S3-compatible object storage.
type StoragePort interface {
	Upload(ctx context.Context, key string, content io.Reader, size int64, contentType string) error
	Download(ctx context.Context, key string) (io.ReadCloser, error)
	Delete(ctx context.Context, key string) error
	Exists(ctx context.Context, key string) (bool, error)
	PresignDownload(ctx context.Context, key string, lifetime time.Duration) (string, error)
	PresignUpload(ctx context.Context, key string, lifetime time.Duration) (string, error)
	EnsureBucket(ctx context.Context) error
}
