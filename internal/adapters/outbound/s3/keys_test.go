package s3_test

import (
	"testing"

	"github.com/morphy76/vuhive-cloud/internal/adapters/outbound/s3"
	"github.com/morphy76/vuhive-cloud/internal/domain/model"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestStorageKeys(t *testing.T) {
	t.Run("KeySourceTarball returns expected path", func(t *testing.T) {
		key, err := s3.KeySourceTarball("suite-123", "v1.0.0")
		require.NoError(t, err)
		assert.Equal(t, "suites/suite-123/sources/source-v1.0.0.tar.gz", key)
	})

	t.Run("KeySourceTarball with empty version defaults cleanly", func(t *testing.T) {
		key, err := s3.KeySourceTarball("suite-123", "")
		require.NoError(t, err)
		assert.Equal(t, "suites/suite-123/sources/source.tar.gz", key)
	})

	t.Run("KeySourceTarball validates suite ID", func(t *testing.T) {
		_, err := s3.KeySourceTarball("", "v1.0.0")
		assert.ErrorIs(t, err, model.ErrValidation)
	})

	t.Run("KeyBinaryArtifact returns expected path", func(t *testing.T) {
		key, err := s3.KeyBinaryArtifact("suite-123", "art-456", model.PlatformLinuxAmd64)
		require.NoError(t, err)
		assert.Equal(t, "suites/suite-123/artifacts/art-456/linux-amd64/runner", key)
	})

	t.Run("KeyBinaryArtifact validates invalid platform", func(t *testing.T) {
		_, err := s3.KeyBinaryArtifact("suite-123", "art-456", model.Platform("darwin/arm64"))
		assert.ErrorIs(t, err, model.ErrInvalidPlatform)
	})

	t.Run("KeyBinaryArtifact validates empty suite or artifact ID", func(t *testing.T) {
		_, err := s3.KeyBinaryArtifact("", "art-456", model.PlatformLinuxAmd64)
		assert.ErrorIs(t, err, model.ErrValidation)

		_, err = s3.KeyBinaryArtifact("suite-123", "", model.PlatformLinuxAmd64)
		assert.ErrorIs(t, err, model.ErrValidation)
	})

	t.Run("KeyBuildLogs returns expected path", func(t *testing.T) {
		key, err := s3.KeyBuildLogs("suite-123", "art-456")
		require.NoError(t, err)
		assert.Equal(t, "suites/suite-123/artifacts/art-456/build.log", key)
	})

	t.Run("KeyBuildLogs validates empty IDs", func(t *testing.T) {
		_, err := s3.KeyBuildLogs("", "art-456")
		assert.ErrorIs(t, err, model.ErrValidation)
	})

	t.Run("KeyConfiguration returns expected path", func(t *testing.T) {
		key, err := s3.KeyConfiguration("suite-123", "cfg-789")
		require.NoError(t, err)
		assert.Equal(t, "suites/suite-123/configs/cfg-789.yaml", key)
	})

	t.Run("KeyConfiguration validates empty IDs", func(t *testing.T) {
		_, err := s3.KeyConfiguration("", "cfg-789")
		assert.ErrorIs(t, err, model.ErrValidation)
	})

	t.Run("KeyExecutionLogs returns expected path", func(t *testing.T) {
		key, err := s3.KeyExecutionLogs("run-321")
		require.NoError(t, err)
		assert.Equal(t, "runs/run-321/run.log", key)
	})

	t.Run("KeyExecutionLogs validates empty run ID", func(t *testing.T) {
		_, err := s3.KeyExecutionLogs("")
		assert.ErrorIs(t, err, model.ErrValidation)
	})

	t.Run("KeySummaryReport returns expected path", func(t *testing.T) {
		key, err := s3.KeySummaryReport("run-321")
		require.NoError(t, err)
		assert.Equal(t, "runs/run-321/summary.json", key)
	})

	t.Run("KeySummaryReport validates empty run ID", func(t *testing.T) {
		_, err := s3.KeySummaryReport("")
		assert.ErrorIs(t, err, model.ErrValidation)
	})
}
