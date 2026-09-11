package docker_test

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gopkg.in/yaml.v3"
)

func TestCIWorkflowStructure(t *testing.T) {
	repoRoot := findRepoRoot(t)
	ciPath := filepath.Join(repoRoot, ".github", "workflows", "ci.yaml")

	contentBytes, err := os.ReadFile(ciPath)
	require.NoError(t, err, "ci.yaml must exist under .github/workflows/")

	var parsed map[string]interface{}
	err = yaml.Unmarshal(contentBytes, &parsed)
	require.NoError(t, err, "ci.yaml must be valid YAML")

	content := string(contentBytes)

	t.Run("Triggers and Concurrency", func(t *testing.T) {
		assert.Contains(t, content, "pull_request:", "must trigger on pull_request")
		assert.Contains(t, content, "push:", "must trigger on push")
		assert.Contains(t, content, "main", "must target main branch")
		assert.Contains(t, content, "concurrency:", "must specify concurrency group")
		assert.Contains(t, content, "cancel-in-progress: true", "must cancel in-progress runs")
	})

	t.Run("Frontend Web Validation Job", func(t *testing.T) {
		assert.Contains(t, content, "pnpm", "must configure pnpm")
		assert.Contains(t, content, "lint", "must execute web lint")
		assert.Contains(t, content, "test", "must execute web test")
		assert.Contains(t, content, "build", "must execute web build")
		assert.Contains(t, content, "frozen-lockfile", "must install with --frozen-lockfile")
	})

	t.Run("Go BFF Validation Job", func(t *testing.T) {
		assert.Contains(t, content, "golangci-lint", "must execute golangci-lint")
		assert.Contains(t, content, "./cmd/bff/...", "must target cmd/bff")
		assert.Contains(t, content, "./internal/bff/...", "must target internal/bff")
		assert.Contains(t, content, "-race", "must run tests with -race")
		assert.Contains(t, content, "go test", "must run go test")
	})
}

func TestDockerWorkflowStructure(t *testing.T) {
	repoRoot := findRepoRoot(t)
	dockerPath := filepath.Join(repoRoot, ".github", "workflows", "docker.yaml")

	contentBytes, err := os.ReadFile(dockerPath)
	require.NoError(t, err, "docker.yaml must exist under .github/workflows/")

	var parsed map[string]interface{}
	err = yaml.Unmarshal(contentBytes, &parsed)
	require.NoError(t, err, "docker.yaml must be valid YAML")

	content := string(contentBytes)

	t.Run("Triggers and Permissions", func(t *testing.T) {
		assert.Contains(t, content, "packages: write", "must grant packages: write for GHCR")
		assert.Contains(t, content, "id-token: write", "must grant id-token: write for Cosign OIDC")
		assert.Contains(t, content, "tags:", "must trigger on tags")
		assert.True(t, strings.Contains(content, "v*") || strings.Contains(content, "v*.*.*"), "must match semver tags")
	})

	t.Run("Multi-Arch and GHCR BFF Image Target", func(t *testing.T) {
		assert.Contains(t, content, "ghcr.io", "must publish to ghcr.io")
		assert.Contains(t, content, "bff", "must configure BFF image")
		assert.Contains(t, content, "deploy/docker/bff.Dockerfile", "must use bff.Dockerfile")
		assert.Contains(t, content, "linux/amd64", "must support linux/amd64")
		assert.Contains(t, content, "linux/arm64", "must support linux/arm64")
	})

	t.Run("Cosign Image Signing", func(t *testing.T) {
		assert.Contains(t, content, "cosign", "must integrate cosign for signing")
		assert.Contains(t, content, "cosign sign", "must execute cosign sign")
	})
}
