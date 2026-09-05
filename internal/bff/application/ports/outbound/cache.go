package outbound

import (
	"context"
	"time"
)

// CachePort defines the driven outbound port for session or response caching.
type CachePort interface {
	Get(ctx context.Context, key string) ([]byte, bool, error)
	Set(ctx context.Context, key string, value []byte, ttl time.Duration) error
	Delete(ctx context.Context, key string) error
}
