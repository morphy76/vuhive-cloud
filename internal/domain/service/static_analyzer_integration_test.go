//go:build integration

package service_test

import (
	"archive/tar"
	"bytes"
	"compress/gzip"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"testing"

	"github.com/morphy76/vuhive-cloud/internal/domain/service"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestStaticAnalyzer_GeneratedDriver_CompilesAndRuns(t *testing.T) {
	analyzer := service.NewStaticAnalyzer(service.StaticAnalyzerConfig{})
	archive := createTestTarGz(t, map[string]string{
		"go.mod": validGoMod(),
		"scenario.go": `package scenario

import (
	"github.com/morphy76/vuhive/pkg/vuhive"
)

func NewScenario() *vuhive.Scenario {
	return &vuhive.Scenario{
		RunVU: func(ctx vuhive.VUContext) error {
			return nil
		},
	}
}
`,
		"vuhive.yaml": `version: "1.0"
default_scenario: "smoke"
scenarios:
  smoke:
    type: "constant_vus"
    vus: 1
    run_period: "1s"
`,
	})

	preparedBytes, res, err := analyzer.PrepareSourceArchive(bytes.NewReader(archive), service.StaticAnalysisOptions{
		SuiteName: "Integration Smoke Test",
	})
	require.NoError(t, err)
	require.NotNil(t, res)

	tmpDir := t.TempDir()
	gr, err := gzip.NewReader(bytes.NewReader(preparedBytes))
	require.NoError(t, err)
	tr := tar.NewReader(gr)

	for {
		hdr, err := tr.Next()
		if err == io.EOF {
			break
		}
		require.NoError(t, err)

		targetPath := filepath.Join(tmpDir, hdr.Name)
		if hdr.Typeflag == tar.TypeDir {
			require.NoError(t, os.MkdirAll(targetPath, 0755))
			continue
		}
		require.NoError(t, os.MkdirAll(filepath.Dir(targetPath), 0755))
		f, err := os.OpenFile(targetPath, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, hdr.FileInfo().Mode())
		require.NoError(t, err)
		_, err = io.Copy(f, tr)
		require.NoError(t, err)
		require.NoError(t, f.Close())
	}

	// Run go mod tidy in tmpDir
	tidyCmd := exec.Command("go", "mod", "tidy")
	tidyCmd.Dir = tmpDir
	tidyOut, err := tidyCmd.CombinedOutput()
	require.NoError(t, err, "go mod tidy failed: %s", string(tidyOut))

	// Run go build in tmpDir
	runnerBin := filepath.Join(tmpDir, "runner")
	buildCmd := exec.Command("go", "build", "-o", runnerBin, ".")
	buildCmd.Dir = tmpDir
	buildOut, err := buildCmd.CombinedOutput()
	require.NoError(t, err, "go build failed: %s", string(buildOut))

	// Execute runner with --summary-export
	summaryPath := filepath.Join(tmpDir, "summary.json")
	runCmd := exec.Command(runnerBin, "--summary-export="+summaryPath, "--config="+filepath.Join(tmpDir, "vuhive.yaml"))
	runCmd.Dir = tmpDir
	runOut, err := runCmd.CombinedOutput()
	require.NoError(t, err, "runner execution failed: %s", string(runOut))

	// Verify summary.json was created and has passed status
	summaryBytes, err := os.ReadFile(summaryPath)
	require.NoError(t, err)
	assert.Contains(t, string(summaryBytes), `"suite_name": "Integration Smoke Test"`)
	assert.Contains(t, string(summaryBytes), `"scenario": "smoke"`)
}
