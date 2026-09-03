package s3_test

import (
	"testing"

	"github.com/morphy76/vuhive-cloud/internal/adapters/outbound/s3"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestConfig_Validation(t *testing.T) {
	t.Run("valid config with all fields", func(t *testing.T) {
		cfg := s3.Config{
			Endpoint:        "http://localhost:9000",
			Region:          "us-east-1",
			Bucket:          "vuhive-artifacts",
			AccessKeyID:     "minioadmin",
			SecretAccessKey: "minioadmin",
			UsePathStyle:    true,
		}
		require.NoError(t, cfg.Validate())
	})

	t.Run("defaults applied on empty region and path style", func(t *testing.T) {
		cfg := s3.Config{
			Bucket:          "vuhive-artifacts",
			AccessKeyID:     "key",
			SecretAccessKey: "secret",
		}
		require.NoError(t, cfg.Validate())
		assert.Equal(t, "us-east-1", cfg.Region)
		assert.True(t, cfg.UsePathStyle)
	})

	t.Run("missing bucket returns error", func(t *testing.T) {
		cfg := s3.Config{
			AccessKeyID:     "key",
			SecretAccessKey: "secret",
		}
		assert.Error(t, cfg.Validate())
	})

	t.Run("missing access key returns error", func(t *testing.T) {
		cfg := s3.Config{
			Bucket:          "bucket",
			SecretAccessKey: "secret",
		}
		assert.Error(t, cfg.Validate())
	})

	t.Run("missing secret key returns error", func(t *testing.T) {
		cfg := s3.Config{
			Bucket:      "bucket",
			AccessKeyID: "key",
		}
		assert.Error(t, cfg.Validate())
	})
}
