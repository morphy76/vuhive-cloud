package cli_test

import (
	"bytes"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/morphy76/vuhive-cloud/internal/cli"
	"github.com/morphy76/vuhive-cloud/internal/domain/model"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func createFakeJWT(username string, roles, groups []string, exp time.Time) string {
	header := base64.RawURLEncoding.EncodeToString([]byte(`{"alg":"none","typ":"JWT"}`))
	payloadData := map[string]interface{}{
		"sub":                "sub-" + username,
		"preferred_username": username,
		"email":              username + "@example.com",
		"realm_access": map[string]interface{}{
			"roles": roles,
		},
		"groups": groups,
		"exp":    exp.Unix(),
	}
	payloadJSON, _ := json.Marshal(payloadData)
	payload := base64.RawURLEncoding.EncodeToString(payloadJSON)
	return fmt.Sprintf("%s.%s.sig", header, payload)
}

func TestCredentialStore(t *testing.T) {
	tempDir, err := os.MkdirTemp("", "vuhive-cli-test-*")
	require.NoError(t, err)
	defer func() { _ = os.RemoveAll(tempDir) }()

	credPath := filepath.Join(tempDir, "credentials.json")
	store := cli.NewFileCredentialStore(credPath)

	t.Run("returns not authenticated when file does not exist", func(t *testing.T) {
		creds, err := store.Load()
		require.NoError(t, err)
		assert.Nil(t, creds)
	})

	t.Run("saves and loads valid credentials with 0600 permissions", func(t *testing.T) {
		jwtToken := createFakeJWT("alice", []string{model.RoleDeveloper}, []string{"/developers"}, time.Now().Add(time.Hour))
		newCreds := &cli.Credentials{
			AccessToken:  jwtToken,
			RefreshToken: "refresh-123",
			IDToken:      "id-123",
			IssuerURL:    "http://keycloak.local/realms/vuhive",
			ServerURL:    "http://localhost:8080",
			ExpiresAt:    time.Now().Add(time.Hour),
		}

		err := store.Save(newCreds)
		require.NoError(t, err)

		info, err := os.Stat(credPath)
		require.NoError(t, err)
		assert.Equal(t, os.FileMode(0600), info.Mode().Perm())

		loaded, err := store.Load()
		require.NoError(t, err)
		require.NotNil(t, loaded)
		assert.Equal(t, jwtToken, loaded.AccessToken)
		assert.Equal(t, "refresh-123", loaded.RefreshToken)

		claims, err := loaded.Claims()
		require.NoError(t, err)
		require.NotNil(t, claims)
		assert.Equal(t, "alice", claims.Username())
		assert.True(t, claims.HasRole(model.RoleDeveloper))
		assert.True(t, claims.InGroup("/developers"))
	})

	t.Run("clears credentials on logout", func(t *testing.T) {
		err := store.Clear()
		require.NoError(t, err)

		loaded, err := store.Load()
		require.NoError(t, err)
		assert.Nil(t, loaded)
	})
}

func TestCLI_AppRouting(t *testing.T) {
	tempDir, err := os.MkdirTemp("", "vuhive-app-test-*")
	require.NoError(t, err)
	defer func() { _ = os.RemoveAll(tempDir) }()

	credPath := filepath.Join(tempDir, "credentials.json")
	store := cli.NewFileCredentialStore(credPath)

	t.Run("displays help on empty args or --help", func(t *testing.T) {
		stdout := &bytes.Buffer{}
		stderr := &bytes.Buffer{}
		app := cli.NewApp(store)

		exitCode := app.Execute([]string{"--help"}, stdout, stderr)
		assert.Equal(t, 0, exitCode)
		assert.Contains(t, stdout.String(), "vuhive")
		assert.Contains(t, stdout.String(), "auth")
		assert.Contains(t, stdout.String(), "suite")
		assert.Contains(t, stdout.String(), "run")
		assert.Contains(t, stdout.String(), "schedule")
	})

	t.Run("auth status when not logged in", func(t *testing.T) {
		stdout := &bytes.Buffer{}
		stderr := &bytes.Buffer{}
		app := cli.NewApp(store)

		exitCode := app.Execute([]string{"auth", "status"}, stdout, stderr)
		assert.Equal(t, 0, exitCode)
		assert.Contains(t, stdout.String(), "Not authenticated")
	})

	t.Run("auth status when logged in displays roles", func(t *testing.T) {
		jwtToken := createFakeJWT("bob", []string{model.RoleDeployer}, []string{"/deployers"}, time.Now().Add(time.Hour))
		_ = store.Save(&cli.Credentials{
			AccessToken: jwtToken,
			ExpiresAt:   time.Now().Add(time.Hour),
			ServerURL:   "http://localhost:8080",
		})

		stdout := &bytes.Buffer{}
		stderr := &bytes.Buffer{}
		app := cli.NewApp(store)

		exitCode := app.Execute([]string{"auth", "status"}, stdout, stderr)
		assert.Equal(t, 0, exitCode)
		assert.Contains(t, stdout.String(), "bob")
		assert.Contains(t, stdout.String(), model.RoleDeployer)
		assert.Contains(t, stdout.String(), "/deployers")
	})
}
