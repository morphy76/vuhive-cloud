package service_test

import (
	"archive/tar"
	"bytes"
	"compress/gzip"
	"io"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/morphy76/vuhive-cloud/internal/domain/model"
	"github.com/morphy76/vuhive-cloud/internal/domain/service"
)

func createTestTarGz(t *testing.T, files map[string]string) []byte {
	t.Helper()
	var buf bytes.Buffer
	gw := gzip.NewWriter(&buf)
	tw := tar.NewWriter(gw)

	for name, content := range files {
		data := []byte(content)
		hdr := &tar.Header{
			Name: name,
			Mode: 0644,
			Size: int64(len(data)),
		}
		require.NoError(t, tw.WriteHeader(hdr))
		_, err := tw.Write(data)
		require.NoError(t, err)
	}

	require.NoError(t, tw.Close())
	require.NoError(t, gw.Close())
	return buf.Bytes()
}

func validGoMod() string {
	return `module mytest

go 1.26

require github.com/morphy76/vuhive v1.1.5
`
}

func validScenarioCode() string {
	return `package scenario

import (
	"context"
	"net/http"
	"time"

	"github.com/morphy76/vuhive/pkg/vuhive"
)

func NewScenario() *vuhive.Scenario {
	client := &http.Client{Timeout: 5 * time.Second}
	return vuhive.NewScenario("User Checkout Flow").
		Step("Homepage", func(ctx context.Context) error {
			resp, err := client.Get("http://target/healthz")
			if err != nil {
				return err
			}
			return resp.Body.Close()
		})
}
`
}

func TestStaticAnalyzer_AnalyzeArchive(t *testing.T) {
	analyzer := service.NewStaticAnalyzer(service.StaticAnalyzerConfig{
		AllowInsecureOverride: false,
	})

	t.Run("fails when archive is corrupted or not gzip", func(t *testing.T) {
		_, err := analyzer.AnalyzeArchive(strings.NewReader("not-a-tarball"), service.StaticAnalysisOptions{})
		require.Error(t, err)
		assert.ErrorIs(t, err, model.ErrInvalidArchive)
	})

	t.Run("fails when go.mod is missing", func(t *testing.T) {
		archive := createTestTarGz(t, map[string]string{
			"scenario.go": validScenarioCode(),
		})
		_, err := analyzer.AnalyzeArchive(bytes.NewReader(archive), service.StaticAnalysisOptions{})
		require.Error(t, err)
		assert.ErrorIs(t, err, model.ErrMissingGoMod)
	})

	t.Run("fails when go.mod is malformed", func(t *testing.T) {
		archive := createTestTarGz(t, map[string]string{
			"go.mod":      "invalid go.mod content [[[",
			"scenario.go": validScenarioCode(),
		})
		_, err := analyzer.AnalyzeArchive(bytes.NewReader(archive), service.StaticAnalysisOptions{})
		require.Error(t, err)
		assert.ErrorIs(t, err, model.ErrValidation)
	})

	t.Run("fails when vuhive dependency is missing from go.mod", func(t *testing.T) {
		archive := createTestTarGz(t, map[string]string{
			"go.mod": `module mytest

go 1.26

require github.com/google/uuid v1.6.0
`,
			"scenario.go": validScenarioCode(),
		})
		_, err := analyzer.AnalyzeArchive(bytes.NewReader(archive), service.StaticAnalysisOptions{})
		require.Error(t, err)
		assert.ErrorIs(t, err, model.ErrMissingVuhiveDependency)
	})

	t.Run("fails when vuhive dependency is indirect in go.mod", func(t *testing.T) {
		archive := createTestTarGz(t, map[string]string{
			"go.mod": `module mytest

go 1.26

require github.com/morphy76/vuhive v1.1.5 // indirect
`,
			"scenario.go": validScenarioCode(),
		})
		_, err := analyzer.AnalyzeArchive(bytes.NewReader(archive), service.StaticAnalysisOptions{})
		require.Error(t, err)
		assert.ErrorIs(t, err, model.ErrMissingVuhiveDependency)
	})

	t.Run("fails when no Go files exist in archive", func(t *testing.T) {
		archive := createTestTarGz(t, map[string]string{
			"go.mod":    validGoMod(),
			"README.md": "# Readme",
		})
		_, err := analyzer.AnalyzeArchive(bytes.NewReader(archive), service.StaticAnalysisOptions{})
		require.Error(t, err)
		assert.ErrorIs(t, err, model.ErrValidation)
		assert.Contains(t, err.Error(), "no Go source files")
	})

	t.Run("fails when Go file has syntax errors", func(t *testing.T) {
		archive := createTestTarGz(t, map[string]string{
			"go.mod":      validGoMod(),
			"scenario.go": "package scenario\n\nfunc Broken( {",
		})
		_, err := analyzer.AnalyzeArchive(bytes.NewReader(archive), service.StaticAnalysisOptions{})
		require.Error(t, err)
		assert.ErrorIs(t, err, model.ErrValidation)
		assert.Contains(t, err.Error(), "syntax error")
	})

	t.Run("fails when package main is declared", func(t *testing.T) {
		archive := createTestTarGz(t, map[string]string{
			"go.mod": validGoMod(),
			"main.go": `package main

import "github.com/morphy76/vuhive"

func Scenario() *vuhive.Scenario { return nil }
`,
		})
		_, err := analyzer.AnalyzeArchive(bytes.NewReader(archive), service.StaticAnalysisOptions{})
		require.Error(t, err)
		assert.ErrorIs(t, err, model.ErrForbiddenPackageMain)
	})

	t.Run("fails when func main is declared", func(t *testing.T) {
		archive := createTestTarGz(t, map[string]string{
			"go.mod": validGoMod(),
			"scenario.go": `package scenario

import "fmt"

func main() { fmt.Println("bypass") }
`,
		})
		_, err := analyzer.AnalyzeArchive(bytes.NewReader(archive), service.StaticAnalysisOptions{})
		require.Error(t, err)
		assert.ErrorIs(t, err, model.ErrForbiddenPackageMain)
	})

	t.Run("fails when forbidden package os/exec is imported", func(t *testing.T) {
		archive := createTestTarGz(t, map[string]string{
			"go.mod": validGoMod(),
			"scenario.go": `package scenario

import (
	"os/exec"
	"github.com/morphy76/vuhive"
)

func NewScenario() *vuhive.Scenario {
	_ = exec.Command("sh")
	return vuhive.NewScenario("Malicious")
}
`,
		})
		_, err := analyzer.AnalyzeArchive(bytes.NewReader(archive), service.StaticAnalysisOptions{})
		require.Error(t, err)
		assert.ErrorIs(t, err, model.ErrForbiddenImport)
		assert.Contains(t, err.Error(), "os/exec")
	})

	t.Run("fails when forbidden package syscall is imported", func(t *testing.T) {
		archive := createTestTarGz(t, map[string]string{
			"go.mod": validGoMod(),
			"scenario.go": `package scenario

import (
	"syscall"
	"github.com/morphy76/vuhive"
)

func NewScenario() *vuhive.Scenario {
	_ = syscall.Getpid()
	return vuhive.NewScenario("Malicious")
}
`,
		})
		_, err := analyzer.AnalyzeArchive(bytes.NewReader(archive), service.StaticAnalysisOptions{})
		require.Error(t, err)
		assert.ErrorIs(t, err, model.ErrForbiddenImport)
		assert.Contains(t, err.Error(), "syscall")
	})

	t.Run("fails when forbidden package unsafe is imported", func(t *testing.T) {
		archive := createTestTarGz(t, map[string]string{
			"go.mod": validGoMod(),
			"scenario.go": `package scenario

import (
	"unsafe"
	"github.com/morphy76/vuhive"
)

func NewScenario() *vuhive.Scenario {
	_ = unsafe.Sizeof(0)
	return vuhive.NewScenario("Malicious")
}
`,
		})
		_, err := analyzer.AnalyzeArchive(bytes.NewReader(archive), service.StaticAnalysisOptions{})
		require.Error(t, err)
		assert.ErrorIs(t, err, model.ErrForbiddenImport)
		assert.Contains(t, err.Error(), "unsafe")
	})

	t.Run("fails when vuhive.Scenario contract is missing", func(t *testing.T) {
		archive := createTestTarGz(t, map[string]string{
			"go.mod": validGoMod(),
			"scenario.go": `package scenario

import "fmt"

func Helper() {
	fmt.Println("No scenario contract declared here")
}
`,
		})
		_, err := analyzer.AnalyzeArchive(bytes.NewReader(archive), service.StaticAnalysisOptions{})
		require.Error(t, err)
		assert.ErrorIs(t, err, model.ErrMissingScenarioContract)
	})

	t.Run("succeeds with valid scenario function NewScenario", func(t *testing.T) {
		archive := createTestTarGz(t, map[string]string{
			"go.mod":      validGoMod(),
			"scenario.go": validScenarioCode(),
		})
		res, err := analyzer.AnalyzeArchive(bytes.NewReader(archive), service.StaticAnalysisOptions{})
		require.NoError(t, err)
		assert.Equal(t, "mytest", res.ModuleName)
		assert.Equal(t, "scenario", res.PackageName)
		assert.Equal(t, "NewScenario", res.EntrypointName)
		assert.False(t, res.IsDangerous)
		assert.Empty(t, res.SuspiciousImports)
	})

	t.Run("succeeds with function Scenario", func(t *testing.T) {
		archive := createTestTarGz(t, map[string]string{
			"go.mod": validGoMod(),
			"scenario.go": `package scenario

import "github.com/morphy76/vuhive"

func Scenario() *vuhive.Scenario {
	return vuhive.NewScenario("Checkout")
}
`,
		})
		res, err := analyzer.AnalyzeArchive(bytes.NewReader(archive), service.StaticAnalysisOptions{})
		require.NoError(t, err)
		assert.Equal(t, "Scenario", res.EntrypointName)
	})

	t.Run("succeeds with function returning error", func(t *testing.T) {
		archive := createTestTarGz(t, map[string]string{
			"go.mod": validGoMod(),
			"scenario.go": `package scenario

import "github.com/morphy76/vuhive"

func InitScenario() (*vuhive.Scenario, error) {
	return vuhive.NewScenario("Checkout"), nil
}
`,
		})
		res, err := analyzer.AnalyzeArchive(bytes.NewReader(archive), service.StaticAnalysisOptions{})
		require.NoError(t, err)
		assert.Equal(t, "InitScenario", res.EntrypointName)
		assert.True(t, res.ReturnsError)
	})

	t.Run("succeeds with global variable Scenario", func(t *testing.T) {
		archive := createTestTarGz(t, map[string]string{
			"go.mod": validGoMod(),
			"scenario.go": `package scenario

import "github.com/morphy76/vuhive"

var Scenario = vuhive.NewScenario("Checkout")
`,
		})
		res, err := analyzer.AnalyzeArchive(bytes.NewReader(archive), service.StaticAnalysisOptions{})
		require.NoError(t, err)
		assert.Equal(t, "Scenario", res.EntrypointName)
		assert.Equal(t, service.EntrypointKindVariable, res.EntrypointKind)
	})

	t.Run("succeeds with Register function", func(t *testing.T) {
		archive := createTestTarGz(t, map[string]string{
			"go.mod": validGoMod(),
			"scenario.go": `package scenario

import "github.com/morphy76/vuhive"

func Register(engine *vuhive.Engine) {
	_ = engine.Run(vuhive.NewScenario("Checkout"))
}
`,
		})
		res, err := analyzer.AnalyzeArchive(bytes.NewReader(archive), service.StaticAnalysisOptions{})
		require.NoError(t, err)
		assert.Equal(t, "Register", res.EntrypointName)
		assert.Equal(t, service.EntrypointKindRegister, res.EntrypointKind)
	})
}

func TestStaticAnalyzer_ImportOverridesAndDangerousFlag(t *testing.T) {
	codeWithSyscall := `package scenario

import (
	"syscall"
	"github.com/morphy76/vuhive"
)

func NewScenario() *vuhive.Scenario {
	_ = syscall.Getpid()
	return vuhive.NewScenario("Custom")
}
`

	t.Run("override rejected when cluster policy disables override", func(t *testing.T) {
		analyzer := service.NewStaticAnalyzer(service.StaticAnalyzerConfig{
			AllowInsecureOverride: false,
		})
		archive := createTestTarGz(t, map[string]string{
			"go.mod":      validGoMod(),
			"scenario.go": codeWithSyscall,
		})
		_, err := analyzer.AnalyzeArchive(bytes.NewReader(archive), service.StaticAnalysisOptions{
			AllowInsecureImports: true,
		})
		require.Error(t, err)
		assert.ErrorIs(t, err, model.ErrInsecureOverrideForbidden)
	})

	t.Run("override accepted when cluster policy allows override", func(t *testing.T) {
		analyzer := service.NewStaticAnalyzer(service.StaticAnalyzerConfig{
			AllowInsecureOverride: true,
		})
		archive := createTestTarGz(t, map[string]string{
			"go.mod":      validGoMod(),
			"scenario.go": codeWithSyscall,
		})
		res, err := analyzer.AnalyzeArchive(bytes.NewReader(archive), service.StaticAnalysisOptions{
			AllowInsecureImports: true,
		})
		require.NoError(t, err)
		assert.True(t, res.IsDangerous)
		assert.Contains(t, res.SuspiciousImports, "syscall")
	})

	t.Run("deployer whitelist allows specific package without marking dangerous", func(t *testing.T) {
		analyzer := service.NewStaticAnalyzer(service.StaticAnalyzerConfig{
			AllowInsecureOverride: false,
			WhitelistedPackages:   []string{"syscall"},
		})
		archive := createTestTarGz(t, map[string]string{
			"go.mod":      validGoMod(),
			"scenario.go": codeWithSyscall,
		})
		res, err := analyzer.AnalyzeArchive(bytes.NewReader(archive), service.StaticAnalysisOptions{})
		require.NoError(t, err)
		assert.False(t, res.IsDangerous)
		assert.Empty(t, res.SuspiciousImports)
	})
}

func TestStaticAnalyzer_PrepareSourceArchive(t *testing.T) {
	analyzer := service.NewStaticAnalyzer(service.StaticAnalyzerConfig{})

	t.Run("successfully reorganizes files into scenario/ and injects main.go", func(t *testing.T) {
		archive := createTestTarGz(t, map[string]string{
			"go.mod":      validGoMod(),
			"scenario.go": validScenarioCode(),
		})

		preparedBytes, res, err := analyzer.PrepareSourceArchive(bytes.NewReader(archive), service.StaticAnalysisOptions{})
		require.NoError(t, err)
		require.NotNil(t, res)
		assert.Equal(t, "mytest", res.ModuleName)

		// Inspect prepared archive entries
		gr, err := gzip.NewReader(bytes.NewReader(preparedBytes))
		require.NoError(t, err)
		tr := tar.NewReader(gr)

		entries := make(map[string]string)
		for {
			hdr, err := tr.Next()
			if err == io.EOF {
				break
			}
			require.NoError(t, err)
			var content bytes.Buffer
			_, err = io.Copy(&content, tr)
			require.NoError(t, err)
			entries[hdr.Name] = content.String()
		}

		assert.Contains(t, entries, "go.mod")
		assert.Contains(t, entries, "main.go")
		assert.Contains(t, entries, "scenario/scenario.go")

		mainContent := entries["main.go"]
		assert.Contains(t, mainContent, "package main")
		assert.Contains(t, mainContent, "mytest/scenario")
		assert.Contains(t, mainContent, "scenario.NewScenario()")
		assert.Contains(t, mainContent, `"summary-export"`)
		assert.Contains(t, mainContent, `"github.com/morphy76/vuhive/pkg/vuhive"`)
		assert.NotContains(t, mainContent, "\t\"github.com/morphy76/vuhive\"\n")
	})
}
