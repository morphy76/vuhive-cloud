package docker_test

import (
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func findRepoRoot(t *testing.T) string {
	t.Helper()
	dir, err := filepath.Abs(".")
	require.NoError(t, err)

	for {
		if _, err := os.Stat(filepath.Join(dir, "go.mod")); err == nil {
			return dir
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			t.Fatal("could not locate repository root containing go.mod")
		}
		dir = parent
	}
}

func TestBFFDockerfileStructure(t *testing.T) {
	repoRoot := findRepoRoot(t)
	dockerfilePath := filepath.Join(repoRoot, "deploy", "docker", "bff.Dockerfile")

	contentBytes, err := os.ReadFile(dockerfilePath)
	require.NoError(t, err, "bff.Dockerfile must exist")
	content := string(contentBytes)

	t.Run("Stage 1 - Node 22 Frontend Builder with pnpm", func(t *testing.T) {
		assert.Regexp(t, regexp.MustCompile(`(?m)^FROM\s+node:22-alpine(\s+AS\s+frontend-builder|\s+AS\s+frontend)?`), content,
			"must use node:22-alpine base image for frontend build stage")
		assert.Contains(t, content, "corepack", "must use corepack or pnpm setup for dependency management")
		assert.Contains(t, content, "pnpm", "must execute pnpm commands")
		assert.Regexp(t, regexp.MustCompile(`pnpm\s+.*build`), content, "must execute pnpm build to generate static assets")
	})

	t.Run("Stage 2 - Golang 1.26 Builder embedding frontend assets", func(t *testing.T) {
		assert.Regexp(t, regexp.MustCompile(`(?m)^FROM\s+golang:1.26-alpine\s+AS\s+builder`), content,
			"must use golang:1.26-alpine for Go compilation stage")
		assert.Regexp(t, regexp.MustCompile(`COPY\s+--from=(frontend-builder|frontend)\s+.*dist.*web/dist`), content,
			"must copy compiled frontend dist assets from frontend stage into web/dist")
		assert.Contains(t, content, "CGO_ENABLED=0", "must compile statically without cgo")
		assert.Contains(t, content, "internal/version", "must inject version metadata via ldflags")
		assert.Contains(t, content, "./cmd/bff", "must compile cmd/bff")
	})

	t.Run("Stage 3 - Minimal hardened runtime container", func(t *testing.T) {
		assert.Regexp(t, regexp.MustCompile(`(?m)^FROM\s+alpine:3.20`), content,
			"must use alpine:3.20 for minimal runtime")
		assert.Contains(t, content, "10001", "must run as non-root user (UID 10001)")
		assert.Contains(t, content, "USER 10001:10001", "must specify USER 10001:10001")
		assert.Regexp(t, regexp.MustCompile(`(?m)^EXPOSE\s+8080`), content,
			"must expose HTTP port 8080")
		assert.Contains(t, content, "ENV PORT=8080", "must default runtime PORT to 8080")
		assert.Contains(t, content, `ENTRYPOINT ["/usr/local/bin/bff"]`, "must set entrypoint to bff binary")
	})
}

func TestDockerignorePresent(t *testing.T) {
	repoRoot := findRepoRoot(t)
	dockerignorePath := filepath.Join(repoRoot, ".dockerignore")

	contentBytes, err := os.ReadFile(dockerignorePath)
	require.NoError(t, err, ".dockerignore must exist at repository root")
	content := string(contentBytes)

	assert.True(t, strings.Contains(content, ".git"), ".dockerignore must exclude .git")
	assert.True(t, strings.Contains(content, "node_modules"), ".dockerignore must exclude node_modules")
	assert.True(t, strings.Contains(content, "bin/"), ".dockerignore must exclude compiled binaries")
}
