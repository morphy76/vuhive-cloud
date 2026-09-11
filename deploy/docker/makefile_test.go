package docker_test

import (
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestMakefileBFFAndWebTargets(t *testing.T) {
	repoRoot := findRepoRoot(t)
	makefilePath := filepath.Join(repoRoot, "Makefile")

	contentBytes, err := os.ReadFile(makefilePath)
	require.NoError(t, err, "Makefile must exist at repository root")
	content := string(contentBytes)

	t.Run("Single Monorepo Version File (VERSION.vuhive)", func(t *testing.T) {
		versionPath := filepath.Join(repoRoot, "VERSION.vuhive")
		versionBytes, err := os.ReadFile(versionPath)
		require.NoError(t, err, "VERSION.vuhive must exist at repository root as the single version file")

		versionStr := strings.TrimSpace(string(versionBytes))
		assert.Regexp(t, regexp.MustCompile(`^\d+\.\d+\.\d+$`), versionStr,
			"VERSION.vuhive must follow Semantic Versioning (e.g. 0.0.1)")

		versionBffPath := filepath.Join(repoRoot, "VERSION.bff")
		_, err = os.Stat(versionBffPath)
		assert.True(t, os.IsNotExist(err), "VERSION.bff must NOT exist; monorepo uses single version file")
	})

	requiredTargets := []string{
		"build-bff",
		"test-bff",
		"lint-bff",
		"web-install",
		"web-build",
		"web-test",
		"web-lint",
		"docker-build-bff",
	}

	for _, target := range requiredTargets {
		t.Run("Target Declared and Phony: "+target, func(t *testing.T) {
			targetRegex := regexp.MustCompile(`(?m)^` + regexp.QuoteMeta(target) + `:.*##\s+.+$`)
			assert.Regexp(t, targetRegex, content, "target %s must be declared with a ## help comment", target)

			phonyRegex := regexp.MustCompile(`(?m)^\.PHONY:\s+.*` + regexp.QuoteMeta(target) + `(\s+.*)?$`)
			assert.Regexp(t, phonyRegex, content, "target %s must be marked as .PHONY", target)
		})
	}

	t.Run("Make Help Output Coverage", func(t *testing.T) {
		cmd := exec.Command("make", "-C", repoRoot, "help")
		outputBytes, err := cmd.CombinedOutput()
		require.NoError(t, err, "make help must execute cleanly: %s", string(outputBytes))
		helpOutput := string(outputBytes)

		for _, target := range requiredTargets {
			assert.Contains(t, helpOutput, target, "make help output must include %s", target)
		}
	})
}
