package cli_test

import (
	"bytes"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/morphy76/vuhive-cloud/internal/cli"
	"github.com/morphy76/vuhive-cloud/internal/domain/model"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type roundTripperFunc func(*http.Request) (*http.Response, error)

func (f roundTripperFunc) RoundTrip(req *http.Request) (*http.Response, error) {
	return f(req)
}

func TestCLI_RoleGuards(t *testing.T) {
	tempDir, err := os.MkdirTemp("", "vuhive-roleguard-test-*")
	require.NoError(t, err)
	defer func() { _ = os.RemoveAll(tempDir) }()

	credPath := filepath.Join(tempDir, "credentials.json")
	store := cli.NewFileCredentialStore(credPath)

	t.Run("deployer cannot upload test suite", func(t *testing.T) {
		deployerJWT := createFakeJWT("deployer-bob", []string{model.RoleDeployer}, []string{"/deployers"}, time.Now().Add(time.Hour))
		_ = store.Save(&cli.Credentials{
			AccessToken: deployerJWT,
			ExpiresAt:   time.Now().Add(time.Hour),
		})

		dummyFile := filepath.Join(tempDir, "source.tar.gz")
		_ = os.WriteFile(dummyFile, []byte("dummy"), 0644)

		stdout := &bytes.Buffer{}
		stderr := &bytes.Buffer{}
		app := cli.NewApp(store)

		exitCode := app.Execute([]string{"suite", "upload", "--suite-id", "suite-1", dummyFile}, stdout, stderr)
		assert.Equal(t, 1, exitCode)
		assert.Contains(t, stderr.String(), "role guard violation")
		assert.Contains(t, stderr.String(), model.RoleDeveloper)
	})

	t.Run("developer cannot trigger test run", func(t *testing.T) {
		devJWT := createFakeJWT("dev-alice", []string{model.RoleDeveloper}, []string{"/developers"}, time.Now().Add(time.Hour))
		_ = store.Save(&cli.Credentials{
			AccessToken: devJWT,
			ExpiresAt:   time.Now().Add(time.Hour),
		})

		stdout := &bytes.Buffer{}
		stderr := &bytes.Buffer{}
		app := cli.NewApp(store)

		exitCode := app.Execute([]string{"run", "start", "suite-1", "--profile-id", "prof-1", "--artifact-id", "art-1"}, stdout, stderr)
		assert.Equal(t, 1, exitCode)
		assert.Contains(t, stderr.String(), "role guard violation")
		assert.Contains(t, stderr.String(), model.RoleDeployer)
	})

	t.Run("developer cannot create schedule", func(t *testing.T) {
		devJWT := createFakeJWT("dev-alice", []string{model.RoleDeveloper}, []string{"/developers"}, time.Now().Add(time.Hour))
		_ = store.Save(&cli.Credentials{
			AccessToken: devJWT,
			ExpiresAt:   time.Now().Add(time.Hour),
		})

		stdout := &bytes.Buffer{}
		stderr := &bytes.Buffer{}
		app := cli.NewApp(store)

		exitCode := app.Execute([]string{"schedule", "create", "--name", "nightly", "--suite-id", "suite-1", "--profile-id", "prof-1", "--artifact-id", "art-1", "--cron", "0 2 * * *"}, stdout, stderr)
		assert.Equal(t, 1, exitCode)
		assert.Contains(t, stderr.String(), "role guard violation")
		assert.Contains(t, stderr.String(), model.RoleDeployer)
	})

	t.Run("admin is permitted to trigger run", func(t *testing.T) {
		adminJWT := createFakeJWT("admin-root", []string{model.RoleAdmin}, []string{"/administrators"}, time.Now().Add(time.Hour))
		_ = store.Save(&cli.Credentials{
			AccessToken: adminJWT,
			ExpiresAt:   time.Now().Add(time.Hour),
			ServerURL:   "http://localhost:8080",
		})

		mockClient := &http.Client{
			Transport: roundTripperFunc(func(r *http.Request) (*http.Response, error) {
				assert.Equal(t, "Bearer "+adminJWT, r.Header.Get("Authorization"))
				return &http.Response{
					StatusCode: http.StatusCreated,
					Body:       io.NopCloser(bytes.NewReader([]byte(`{"id":"run-123","suite_id":"suite-1","status":"QUEUED","created_at":"2026-09-06T00:00:00Z"}`))),
					Header:     make(http.Header),
				}, nil
			}),
		}

		stdout := &bytes.Buffer{}
		stderr := &bytes.Buffer{}
		app := cli.NewAppWithClient(store, mockClient)

		exitCode := app.Execute([]string{"run", "start", "suite-1", "--profile-id", "prof-1", "--artifact-id", "art-1", "--server", "http://localhost:8080"}, stdout, stderr)
		assert.Equal(t, 0, exitCode)
		assert.Contains(t, stdout.String(), "run-123")
		assert.Contains(t, stdout.String(), "QUEUED")
	})
}
