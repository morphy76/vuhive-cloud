//go:build integration

package s3_test

import (
	"bytes"
	"context"
	"io"
	"net/http"
	"strings"
	"testing"
	"time"

	"github.com/morphy76/vuhive-cloud/internal/adapters/outbound/s3"
	"github.com/morphy76/vuhive-cloud/internal/domain/model"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	tcminio "github.com/testcontainers/testcontainers-go/modules/minio"
)

func setupMinIOContainer(t *testing.T) (*s3.Adapter, func()) {
	t.Helper()
	ctx := context.Background()

	minioContainer, err := tcminio.Run(
		ctx,
		"minio/minio:RELEASE.2024-01-16T16-07-38Z",
		tcminio.WithUsername("minioadmin"),
		tcminio.WithPassword("minioadmin"),
	)
	require.NoError(t, err, "failed to start minio container")

	connStr, err := minioContainer.ConnectionString(ctx)
	require.NoError(t, err, "failed to get minio connection string")

	endpoint := connStr
	if !strings.HasPrefix(endpoint, "http://") && !strings.HasPrefix(endpoint, "https://") {
		endpoint = "http://" + endpoint
	}

	bucket := "vuhive-test-bucket"
	cfg := s3.Config{
		Endpoint:        endpoint,
		Region:          "us-east-1",
		Bucket:          bucket,
		AccessKeyID:     "minioadmin",
		SecretAccessKey: "minioadmin",
		UsePathStyle:    true,
	}

	adapter, err := s3.NewAdapter(ctx, cfg)
	require.NoError(t, err, "failed to instantiate s3 adapter")

	cleanup := func() {
		_ = minioContainer.Terminate(ctx)
	}

	return adapter, cleanup
}

func TestS3Adapter_Integration(t *testing.T) {
	ctx := context.Background()
	adapter, cleanup := setupMinIOContainer(t)
	defer cleanup()

	t.Run("EnsureBucket creates bucket and is idempotent", func(t *testing.T) {
		err := adapter.EnsureBucket(ctx)
		require.NoError(t, err, "first EnsureBucket must succeed")

		err = adapter.EnsureBucket(ctx)
		require.NoError(t, err, "second EnsureBucket must be idempotent")
	})

	t.Run("Upload and Download source tarball", func(t *testing.T) {
		key, err := s3.KeySourceTarball("suite-1", "v1.0.0")
		require.NoError(t, err)

		tarData := []byte("dummy tarball gzip content")
		err = adapter.Upload(ctx, key, bytes.NewReader(tarData), int64(len(tarData)), "application/gzip")
		require.NoError(t, err)

		rc, err := adapter.Download(ctx, key)
		require.NoError(t, err)
		defer rc.Close()

		downloaded, err := io.ReadAll(rc)
		require.NoError(t, err)
		assert.Equal(t, tarData, downloaded)
	})

	t.Run("Upload and Download static binary artifact", func(t *testing.T) {
		key, err := s3.KeyBinaryArtifact("suite-1", "art-1", model.PlatformLinuxAmd64)
		require.NoError(t, err)

		binData := []byte("\x7fELF\x02\x01\x01\x00executable-code")
		err = adapter.Upload(ctx, key, bytes.NewReader(binData), int64(len(binData)), "application/octet-stream")
		require.NoError(t, err)

		rc, err := adapter.Download(ctx, key)
		require.NoError(t, err)
		defer rc.Close()

		downloaded, err := io.ReadAll(rc)
		require.NoError(t, err)
		assert.Equal(t, binData, downloaded)
	})

	t.Run("Upload and Download configuration YAML", func(t *testing.T) {
		key, err := s3.KeyConfiguration("suite-1", "cfg-1")
		require.NoError(t, err)

		yamlData := []byte("version: '1'\nsettings:\n  concurrency: 50\n")
		err = adapter.Upload(ctx, key, bytes.NewReader(yamlData), int64(len(yamlData)), "application/x-yaml")
		require.NoError(t, err)

		rc, err := adapter.Download(ctx, key)
		require.NoError(t, err)
		defer rc.Close()

		downloaded, err := io.ReadAll(rc)
		require.NoError(t, err)
		assert.Equal(t, yamlData, downloaded)
	})

	t.Run("Upload and Download execution logs", func(t *testing.T) {
		key, err := s3.KeyExecutionLogs("run-1")
		require.NoError(t, err)

		logData := []byte("[INFO] 2026-09-03 Starting run 1\n[INFO] 2026-09-03 Completed run 1\n")
		err = adapter.Upload(ctx, key, bytes.NewReader(logData), int64(len(logData)), "text/plain")
		require.NoError(t, err)

		rc, err := adapter.Download(ctx, key)
		require.NoError(t, err)
		defer rc.Close()

		downloaded, err := io.ReadAll(rc)
		require.NoError(t, err)
		assert.Equal(t, logData, downloaded)
	})

	t.Run("Upload and Download summary.json report", func(t *testing.T) {
		key, err := s3.KeySummaryReport("run-1")
		require.NoError(t, err)

		summaryData := []byte(`{"suite_id":"suite-1","run_id":"run-1","p95_ms":12.4,"passed":true}`)
		err = adapter.Upload(ctx, key, bytes.NewReader(summaryData), int64(len(summaryData)), "application/json")
		require.NoError(t, err)

		rc, err := adapter.Download(ctx, key)
		require.NoError(t, err)
		defer rc.Close()

		downloaded, err := io.ReadAll(rc)
		require.NoError(t, err)
		assert.Equal(t, summaryData, downloaded)
	})

	t.Run("Exists checks presence of objects", func(t *testing.T) {
		summaryKey, _ := s3.KeySummaryReport("run-1")
		exists, err := adapter.Exists(ctx, summaryKey)
		require.NoError(t, err)
		assert.True(t, exists)

		exists, err = adapter.Exists(ctx, "non-existent-key")
		require.NoError(t, err)
		assert.False(t, exists)
	})

	t.Run("Delete removes object", func(t *testing.T) {
		key := "to-delete.txt"
		err := adapter.Upload(ctx, key, strings.NewReader("temporary data"), -1, "text/plain")
		require.NoError(t, err)

		exists, err := adapter.Exists(ctx, key)
		require.NoError(t, err)
		assert.True(t, exists)

		err = adapter.Delete(ctx, key)
		require.NoError(t, err)

		exists, err = adapter.Exists(ctx, key)
		require.NoError(t, err)
		assert.False(t, exists)

		_, err = adapter.Download(ctx, key)
		assert.ErrorIs(t, err, model.ErrNotFound)
	})

	t.Run("PresignDownload allows downloading via HTTP GET", func(t *testing.T) {
		key := "presign-download.txt"
		content := "content for presigned download"
		err := adapter.Upload(ctx, key, strings.NewReader(content), int64(len(content)), "text/plain")
		require.NoError(t, err)

		url, err := adapter.PresignDownload(ctx, key, 5*time.Minute)
		require.NoError(t, err)
		assert.NotEmpty(t, url)

		resp, err := http.Get(url)
		require.NoError(t, err)
		defer resp.Body.Close()

		assert.Equal(t, http.StatusOK, resp.StatusCode)
		bodyBytes, err := io.ReadAll(resp.Body)
		require.NoError(t, err)
		assert.Equal(t, content, string(bodyBytes))
	})

	t.Run("PresignUpload allows uploading via HTTP PUT", func(t *testing.T) {
		key := "presign-upload.txt"
		url, err := adapter.PresignUpload(ctx, key, 5*time.Minute)
		require.NoError(t, err)
		assert.NotEmpty(t, url)

		uploadPayload := "uploaded via presigned PUT url"
		req, err := http.NewRequestWithContext(ctx, http.MethodPut, url, strings.NewReader(uploadPayload))
		require.NoError(t, err)
		req.ContentLength = int64(len(uploadPayload))

		resp, err := http.DefaultClient.Do(req)
		require.NoError(t, err)
		defer resp.Body.Close()

		assert.Equal(t, http.StatusOK, resp.StatusCode)

		rc, err := adapter.Download(ctx, key)
		require.NoError(t, err)
		defer rc.Close()

		downloaded, err := io.ReadAll(rc)
		require.NoError(t, err)
		assert.Equal(t, uploadPayload, string(downloaded))
	})

	t.Run("Download non-existent key returns model.ErrNotFound", func(t *testing.T) {
		_, err := adapter.Download(ctx, "does-not-exist-at-all")
		assert.ErrorIs(t, err, model.ErrNotFound)
	})

	t.Run("Empty key validation returns model.ErrEmptyS3Key", func(t *testing.T) {
		assert.ErrorIs(t, adapter.Upload(ctx, "   ", strings.NewReader("a"), 1, ""), model.ErrEmptyS3Key)

		_, err := adapter.Download(ctx, "")
		assert.ErrorIs(t, err, model.ErrEmptyS3Key)

		assert.ErrorIs(t, adapter.Delete(ctx, ""), model.ErrEmptyS3Key)

		_, err = adapter.Exists(ctx, "")
		assert.ErrorIs(t, err, model.ErrEmptyS3Key)

		_, err = adapter.PresignDownload(ctx, "", 1*time.Minute)
		assert.ErrorIs(t, err, model.ErrEmptyS3Key)

		_, err = adapter.PresignUpload(ctx, "", 1*time.Minute)
		assert.ErrorIs(t, err, model.ErrEmptyS3Key)
	})
}
