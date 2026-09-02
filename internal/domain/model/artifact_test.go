package model_test

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/morphy76/vuhive-cloud/internal/domain/model"
)

func TestPlatform(t *testing.T) {
	t.Run("valid platforms", func(t *testing.T) {
		p1, err := model.ParsePlatform("linux/amd64")
		require.NoError(t, err)
		assert.Equal(t, model.PlatformLinuxAmd64, p1)
		assert.True(t, p1.IsValid())

		p2, err := model.ParsePlatform("linux/arm64")
		require.NoError(t, err)
		assert.Equal(t, model.PlatformLinuxArm64, p2)
		assert.True(t, p2.IsValid())
	})

	t.Run("invalid platform returns ErrInvalidPlatform", func(t *testing.T) {
		_, err := model.ParsePlatform("windows/amd64")
		assert.ErrorIs(t, err, model.ErrInvalidPlatform)

		_, err = model.ParsePlatform("darwin/arm64")
		assert.ErrorIs(t, err, model.ErrInvalidPlatform)

		_, err = model.ParsePlatform("")
		assert.ErrorIs(t, err, model.ErrInvalidPlatform)
	})
}

func TestNewArtifact(t *testing.T) {
	t.Run("successfully create artifact in PENDING state", func(t *testing.T) {
		art, err := model.NewArtifact("suite-123", model.PlatformLinuxAmd64)
		require.NoError(t, err)
		require.NotNil(t, art)

		assert.NotEmpty(t, art.ID())
		assert.Equal(t, art.ID(), art.EntityID())
		assert.Equal(t, "suite-123", art.SuiteID())
		assert.Equal(t, model.PlatformLinuxAmd64, art.Platform())
		assert.Equal(t, model.ArtifactStatusPending, art.Status())
		assert.Empty(t, art.S3BinaryKey())
		assert.Empty(t, art.SHA256Checksum())
		assert.Empty(t, art.ErrorMessage())
		assert.False(t, art.CreatedAt().IsZero())
	})

	t.Run("fail with empty suiteID", func(t *testing.T) {
		art, err := model.NewArtifact("", model.PlatformLinuxAmd64)
		assert.ErrorIs(t, err, model.ErrValidation)
		assert.Nil(t, art)
	})

	t.Run("fail with invalid platform", func(t *testing.T) {
		art, err := model.NewArtifact("suite-123", model.Platform("invalid/arch"))
		assert.ErrorIs(t, err, model.ErrInvalidPlatform)
		assert.Nil(t, art)
	})
}

func TestArtifact_StateTransitions(t *testing.T) {
	validChecksum := "e3b0c44298fc1c149afbf4c8996fb92427ae41e4649b934ca495991b7852b855"

	t.Run("happy path: PENDING -> BUILDING -> READY", func(t *testing.T) {
		art, err := model.NewArtifact("suite-123", model.PlatformLinuxAmd64)
		require.NoError(t, err)

		err = art.MarkBuilding()
		require.NoError(t, err)
		assert.Equal(t, model.ArtifactStatusBuilding, art.Status())

		err = art.MarkReady("vuhive-binaries/suite-123/runner", validChecksum)
		require.NoError(t, err)
		assert.Equal(t, model.ArtifactStatusReady, art.Status())
		assert.Equal(t, "vuhive-binaries/suite-123/runner", art.S3BinaryKey())
		assert.Equal(t, validChecksum, art.SHA256Checksum())
	})

	t.Run("fail path: PENDING -> BUILDING -> FAILED", func(t *testing.T) {
		art, err := model.NewArtifact("suite-123", model.PlatformLinuxArm64)
		require.NoError(t, err)

		require.NoError(t, art.MarkBuilding())

		err = art.MarkFailed("compilation error: syntax error in test.go", "vuhive-logs/suite-123/build.log")
		require.NoError(t, err)
		assert.Equal(t, model.ArtifactStatusFailed, art.Status())
		assert.Equal(t, "compilation error: syntax error in test.go", art.ErrorMessage())
		assert.Equal(t, "vuhive-logs/suite-123/build.log", art.BuildLogsS3Key())
	})

	t.Run("fail path directly from PENDING to FAILED", func(t *testing.T) {
		art, err := model.NewArtifact("suite-123", model.PlatformLinuxAmd64)
		require.NoError(t, err)

		err = art.MarkFailed("pre-build AST validation failed", "vuhive-logs/suite-123/ast.log")
		require.NoError(t, err)
		assert.Equal(t, model.ArtifactStatusFailed, art.Status())
	})

	t.Run("illegal transition: PENDING directly to READY without BUILDING", func(t *testing.T) {
		art, err := model.NewArtifact("suite-123", model.PlatformLinuxAmd64)
		require.NoError(t, err)

		err = art.MarkReady("key", validChecksum)
		assert.ErrorIs(t, err, model.ErrInvalidStateTransition)
	})

	t.Run("terminal state READY cannot transition to BUILDING or FAILED", func(t *testing.T) {
		art, err := model.NewArtifact("suite-123", model.PlatformLinuxAmd64)
		require.NoError(t, err)
		require.NoError(t, art.MarkBuilding())
		require.NoError(t, art.MarkReady("key", validChecksum))

		assert.ErrorIs(t, art.MarkBuilding(), model.ErrTerminalState)
		assert.ErrorIs(t, art.MarkFailed("err", "logs"), model.ErrTerminalState)
	})

	t.Run("terminal state FAILED cannot transition to BUILDING or READY", func(t *testing.T) {
		art, err := model.NewArtifact("suite-123", model.PlatformLinuxAmd64)
		require.NoError(t, err)
		require.NoError(t, art.MarkFailed("build failed", "logs"))

		assert.ErrorIs(t, art.MarkBuilding(), model.ErrTerminalState)
		assert.ErrorIs(t, art.MarkReady("key", validChecksum), model.ErrTerminalState)
	})

	t.Run("validation on MarkReady: empty s3Key or invalid checksum", func(t *testing.T) {
		art, err := model.NewArtifact("suite-123", model.PlatformLinuxAmd64)
		require.NoError(t, err)
		require.NoError(t, art.MarkBuilding())

		err = art.MarkReady("", validChecksum)
		assert.ErrorIs(t, err, model.ErrEmptyS3Key)

		err = art.MarkReady("key", "invalid-short-checksum")
		assert.ErrorIs(t, err, model.ErrInvalidChecksum)
	})
}

func TestNewArtifactWithID(t *testing.T) {
	validChecksum := "e3b0c44298fc1c149afbf4c8996fb92427ae41e4649b934ca495991b7852b855"
	now := time.Now()

	t.Run("reconstitute valid artifact", func(t *testing.T) {
		art, err := model.NewArtifactWithID(
			"art-123", "suite-123", model.PlatformLinuxAmd64,
			"key", validChecksum, "logs-key",
			model.ArtifactStatusReady, "", now,
		)
		require.NoError(t, err)
		assert.Equal(t, "art-123", art.ID())
		assert.Equal(t, model.ArtifactStatusReady, art.Status())
	})

	t.Run("reconstitute with invalid platform fails", func(t *testing.T) {
		_, err := model.NewArtifactWithID(
			"art-123", "suite-123", "darwin/arm64",
			"key", validChecksum, "logs-key",
			model.ArtifactStatusReady, "", now,
		)
		assert.ErrorIs(t, err, model.ErrInvalidPlatform)
	})

	t.Run("reconstitute with invalid status fails", func(t *testing.T) {
		_, err := model.NewArtifactWithID(
			"art-123", "suite-123", model.PlatformLinuxAmd64,
			"key", validChecksum, "logs-key",
			"UNKNOWN_STATUS", "", now,
		)
		assert.ErrorIs(t, err, model.ErrInvalidStateTransition)
	})
}
