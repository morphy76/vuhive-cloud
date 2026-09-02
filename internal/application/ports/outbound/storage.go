package outbound

import (
	"context"
	"io"
)

// StoragePort defines the driven port for S3-compatible object storage.
type StoragePort interface {
	Upload(ctx context.Context, key string, content io.Reader, size int64) error
	Download(ctx context.Context, key string) (io.ReadCloser, error)
	Delete(ctx context.Context, key string) error
}
