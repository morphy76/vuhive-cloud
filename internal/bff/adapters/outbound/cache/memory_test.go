package cache_test

import (
	"context"
	"testing"
	"time"

	"github.com/morphy76/vuhive-cloud/internal/bff/adapters/outbound/cache"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestMemoryCache(t *testing.T) {
	ctx := context.Background()
	c := cache.NewMemoryCache()

	t.Run("set and get item", func(t *testing.T) {
		err := c.Set(ctx, "key1", []byte("val1"), 1*time.Minute)
		require.NoError(t, err)

		val, found, err := c.Get(ctx, "key1")
		require.NoError(t, err)
		assert.True(t, found)
		assert.Equal(t, []byte("val1"), val)
	})

	t.Run("missing item returns not found", func(t *testing.T) {
		val, found, err := c.Get(ctx, "nonexistent")
		require.NoError(t, err)
		assert.False(t, found)
		assert.Nil(t, val)
	})

	t.Run("expired item returns not found", func(t *testing.T) {
		err := c.Set(ctx, "expiring", []byte("val2"), 10*time.Millisecond)
		require.NoError(t, err)

		time.Sleep(15 * time.Millisecond)
		val, found, err := c.Get(ctx, "expiring")
		require.NoError(t, err)
		assert.False(t, found)
		assert.Nil(t, val)
	})

	t.Run("delete item", func(t *testing.T) {
		err := c.Set(ctx, "to_delete", []byte("val3"), 1*time.Minute)
		require.NoError(t, err)

		err = c.Delete(ctx, "to_delete")
		require.NoError(t, err)

		val, found, err := c.Get(ctx, "to_delete")
		require.NoError(t, err)
		assert.False(t, found)
		assert.Nil(t, val)
	})
}
