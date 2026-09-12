package service_test

import (
	"archive/tar"
	"archive/zip"
	"bytes"
	"compress/gzip"
	"encoding/hex"
	"io"
	"os/exec"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/morphy76/vuhive-cloud/internal/domain/model"
	"github.com/morphy76/vuhive-cloud/internal/domain/service"
)

// validTarBz2Hex is a pre-compressed valid .tar.bz2 containing go.mod and scenario.go.
const validTarBz2Hex = "425a68393141592653594082f66e00010b5f88ccb07175ff9219c10e107feffffa00080808300199a62d0688801a20c86869a000000347a401a9929e1269a36a00683268000d000004a10a78a644c0c9a87a9900681a01a0d0d34f2d388dc28b3511116020215d7ec3130ccd4932217950406019262282a0aaa8aa59b76d4ad4d8fae55118b43e725af8842ee88d2fa41e8d4c05adbeba4d23e5305a8f011f37e0920eac1f3d9da110220c0b6b5674eb9576345f326d8fa91c6db8529acbc58f3610ece2f85be7202671410d7adc218135436c7a1ddea32e905a11d0eb9174947298aa9190d54395156c5639c1cfa54d193ab39602c843e031f28235b5045084a938b013d9f460bd88d10a024e9153410f64cc66f3908080c797d4c12c324eeb725b21b4c761ae3a7cdc208a1b50981ca535648a608883f18b64257ce46502a05c930688e4fd2062cc257c8404aa83c722e08aa82da7e603f54c2b4c0c5670d0d99b38c63748c737651e8373acc556530d08b0879a1e0099f8f74e42c42cbc98d4412612a10cb9d687196ddea9342529bf271ce86efed8493715145f4308df4cfbe478192ede2d59a5cb6db8b2040ff1772453850904082f66e0"

func createTestZip(t *testing.T, files map[string]string) []byte {
	t.Helper()
	var buf bytes.Buffer
	zw := zip.NewWriter(&buf)

	for name, content := range files {
		w, err := zw.Create(name)
		require.NoError(t, err)
		_, err = w.Write([]byte(content))
		require.NoError(t, err)
	}

	require.NoError(t, zw.Close())
	return buf.Bytes()
}

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

func createTestTarBz2(t *testing.T, files map[string]string) []byte {
	t.Helper()
	var buf bytes.Buffer
	tw := tar.NewWriter(&buf)

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

	// If bzip2 CLI is present, dynamically compress the tar archive
	if bzipPath, err := exec.LookPath("bzip2"); err == nil {
		cmd := exec.Command(bzipPath, "-c")
		cmd.Stdin = &buf
		var out bytes.Buffer
		cmd.Stdout = &out
		if err := cmd.Run(); err == nil {
			return out.Bytes()
		}
	}

	// Fallback to static valid .tar.bz2 fixture
	raw, err := hex.DecodeString(validTarBz2Hex)
	require.NoError(t, err)
	return raw
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

	t.Run("succeeds with valid zip archive containing scenario package", func(t *testing.T) {
		zipArchive := createTestZip(t, map[string]string{
			"go.mod":      validGoMod(),
			"scenario.go": validScenarioCode(),
		})
		res, err := analyzer.AnalyzeArchive(bytes.NewReader(zipArchive), service.StaticAnalysisOptions{})
		require.NoError(t, err)
		assert.Equal(t, "mytest", res.ModuleName)
		assert.Equal(t, "scenario", res.PackageName)
		assert.Equal(t, "NewScenario", res.EntrypointName)
		assert.False(t, res.IsDangerous)
		assert.Empty(t, res.SuspiciousImports)
	})

	t.Run("succeeds with valid tar.bz2 archive containing scenario package", func(t *testing.T) {
		bz2Archive := createTestTarBz2(t, map[string]string{
			"go.mod":      validGoMod(),
			"scenario.go": validScenarioCode(),
		})
		res, err := analyzer.AnalyzeArchive(bytes.NewReader(bz2Archive), service.StaticAnalysisOptions{})
		require.NoError(t, err)
		assert.Equal(t, "mytest", res.ModuleName)
		assert.Equal(t, "scenario", res.PackageName)
		assert.Equal(t, "NewScenario", res.EntrypointName)
		assert.False(t, res.IsDangerous)
		assert.Empty(t, res.SuspiciousImports)
	})

	t.Run("fails when bzip2 archive has corrupted bzip2 data", func(t *testing.T) {
		corrupted := []byte{'B', 'Z', 'h', '9', 0xff, 0x00, 0x12, 0x34}
		_, err := analyzer.AnalyzeArchive(bytes.NewReader(corrupted), service.StaticAnalysisOptions{})
		require.Error(t, err)
		assert.ErrorIs(t, err, model.ErrInvalidArchive)
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

	t.Run("successfully repackages zip archive into tar.gz with injected main.go", func(t *testing.T) {
		zipArchive := createTestZip(t, map[string]string{
			"go.mod":      validGoMod(),
			"scenario.go": validScenarioCode(),
		})

		preparedBytes, res, err := analyzer.PrepareSourceArchive(bytes.NewReader(zipArchive), service.StaticAnalysisOptions{})
		require.NoError(t, err)
		require.NotNil(t, res)
		assert.Equal(t, "mytest", res.ModuleName)

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
		assert.Contains(t, entries["main.go"], "scenario.NewScenario()")
	})

	t.Run("successfully repackages tar.bz2 archive into tar.gz with injected main.go", func(t *testing.T) {
		bz2Archive := createTestTarBz2(t, map[string]string{
			"go.mod":      validGoMod(),
			"scenario.go": validScenarioCode(),
		})

		preparedBytes, res, err := analyzer.PrepareSourceArchive(bytes.NewReader(bz2Archive), service.StaticAnalysisOptions{})
		require.NoError(t, err)
		require.NotNil(t, res)
		assert.Equal(t, "mytest", res.ModuleName)

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
		assert.Contains(t, entries["main.go"], "scenario.NewScenario()")
	})
}
