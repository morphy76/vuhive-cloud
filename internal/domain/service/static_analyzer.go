package service

import (
	"archive/tar"
	"archive/zip"
	"bufio"
	"bytes"
	"compress/gzip"
	"fmt"
	"go/ast"
	"go/parser"
	"go/token"
	"io"
	"path"
	"strings"

	"github.com/morphy76/vuhive-cloud/internal/domain/model"
)

// EntrypointKind denotes how the vuhive.Scenario is declared or initiated in the scenario package.
type EntrypointKind string

const (
	EntrypointKindFunction EntrypointKind = "function"
	EntrypointKindVariable EntrypointKind = "variable"
	EntrypointKindRegister EntrypointKind = "register"
)

var defaultForbiddenPrefixes = []string{
	"os/exec",
	"syscall",
	"unsafe",
	"plugin",
	"runtime/cgo",
	"golang.org/x/sys",
	"net/http/pprof",
	"debug/",
}

// StaticAnalyzerConfig configures cluster-wide static analysis rules and exemptions.
type StaticAnalyzerConfig struct {
	AllowInsecureOverride bool
	WhitelistedPackages   []string
}

// StaticAnalysisOptions contains request-level analysis options.
type StaticAnalysisOptions struct {
	AllowInsecureImports bool
}

// AnalysisResult contains metadata extracted and validated from the uploaded Go package.
type AnalysisResult struct {
	ModuleName        string
	PackageName       string
	EntrypointName    string
	EntrypointKind    EntrypointKind
	ReturnsError      bool
	IsDangerous       bool
	SuspiciousImports []string
	Files             map[string][]byte
}

// StaticAnalyzer performs pre-build static verification, framework contract checking, and driver injection.
type StaticAnalyzer struct {
	cfg StaticAnalyzerConfig
}

// NewStaticAnalyzer creates a new StaticAnalyzer.
func NewStaticAnalyzer(cfg StaticAnalyzerConfig) *StaticAnalyzer {
	return &StaticAnalyzer{cfg: cfg}
}

// AnalyzeArchive parses the uploaded .tar.gz archive, verifies go.mod and AST compliance, and returns analysis metadata.
func (a *StaticAnalyzer) AnalyzeArchive(r io.Reader, opts StaticAnalysisOptions) (*AnalysisResult, error) {
	if r == nil {
		return nil, fmt.Errorf("%w: source archive cannot be nil", model.ErrValidation)
	}

	files, err := extractArchive(r)
	if err != nil {
		return nil, err
	}

	// 1. Direct Dependency Verification (go.mod)
	goModData, hasGoMod := files["go.mod"]
	if !hasGoMod {
		return nil, model.ErrMissingGoMod
	}

	moduleName, hasDirectVuhive, hasIndirectVuhive, err := parseGoMod(goModData)
	if err != nil {
		return nil, fmt.Errorf("%w: %v", model.ErrValidation, err)
	}

	if hasIndirectVuhive && !hasDirectVuhive {
		return nil, model.ErrMissingVuhiveDependency
	}
	if !hasDirectVuhive {
		return nil, model.ErrMissingVuhiveDependency
	}

	// 2. Go AST Static Inspection
	goFiles := make(map[string][]byte)
	for name, data := range files {
		if strings.HasSuffix(name, ".go") && !strings.HasSuffix(name, "_test.go") {
			goFiles[name] = data
		}
	}

	if len(goFiles) == 0 {
		return nil, fmt.Errorf("%w: no Go source files found in uploaded archive", model.ErrValidation)
	}

	var suspiciousImports []string
	fset := token.NewFileSet()
	var (
		detectedPackageName string
		detectedEntrypoint  string
		detectedKind        EntrypointKind
		detectedErrReturn   bool
	)

	for filename, content := range goFiles {
		astFile, err := parser.ParseFile(fset, filename, content, parser.AllErrors)
		if err != nil {
			return nil, fmt.Errorf("%w: syntax error in %s: %v", model.ErrValidation, filename, err)
		}

		pkgName := astFile.Name.Name
		if pkgName == "main" {
			return nil, model.ErrForbiddenPackageMain
		}
		if pkgName != "scenario" {
			return nil, fmt.Errorf("%w: package is '%s', must be 'scenario'", model.ErrForbiddenPackageMain, pkgName)
		}
		detectedPackageName = pkgName

		// Enforce no func main()
		for _, decl := range astFile.Decls {
			if fn, ok := decl.(*ast.FuncDecl); ok {
				if fn.Name.Name == "main" {
					return nil, model.ErrForbiddenPackageMain
				}
			}
		}

		// Inspect imports
		for _, imp := range astFile.Imports {
			importPath := strings.Trim(imp.Path.Value, `"`)
			if a.isForbiddenImport(importPath) {
				if !containsString(suspiciousImports, importPath) {
					suspiciousImports = append(suspiciousImports, importPath)
				}
			}
		}

		// Inspect scenario contracts
		name, kind, returnsErr := a.findScenarioContract(astFile)
		if name != "" && detectedEntrypoint == "" {
			detectedEntrypoint = name
			detectedKind = kind
			detectedErrReturn = returnsErr
		}
	}

	// Handle suspicious imports & overrides
	isDangerous := false
	if len(suspiciousImports) > 0 {
		if opts.AllowInsecureImports {
			if !a.cfg.AllowInsecureOverride {
				return nil, fmt.Errorf("%w: imported disallowed packages %v", model.ErrInsecureOverrideForbidden, suspiciousImports)
			}
			isDangerous = true
		} else {
			return nil, fmt.Errorf("%w: %s", model.ErrForbiddenImport, strings.Join(suspiciousImports, ", "))
		}
	}

	// Verify scenario contract was detected
	if detectedEntrypoint == "" {
		return nil, model.ErrMissingScenarioContract
	}

	return &AnalysisResult{
		ModuleName:        moduleName,
		PackageName:       detectedPackageName,
		EntrypointName:    detectedEntrypoint,
		EntrypointKind:    detectedKind,
		ReturnsError:      detectedErrReturn,
		IsDangerous:       isDangerous,
		SuspiciousImports: suspiciousImports,
		Files:             files,
	}, nil
}

// GenerateMainDriver constructs the platform-managed main.go driver for the scenario.
func (a *StaticAnalyzer) GenerateMainDriver(result *AnalysisResult) ([]byte, error) {
	if result == nil {
		return nil, fmt.Errorf("%w: analysis result cannot be nil", model.ErrValidation)
	}

	var invocation string
	switch result.EntrypointKind {
	case EntrypointKindRegister:
		invocation = fmt.Sprintf("\tscenario.%s(engine)", result.EntrypointName)
	case EntrypointKindVariable:
		invocation = fmt.Sprintf(`	sc := scenario.%s
	if err := engine.Run(sc); err != nil {
		fmt.Fprintf(os.Stderr, "scenario execution error: %%v\n", err)
		os.Exit(1)
	}`, result.EntrypointName)
	case EntrypointKindFunction:
		if result.ReturnsError {
			invocation = fmt.Sprintf(`	sc, err := scenario.%s()
	if err != nil {
		fmt.Fprintf(os.Stderr, "scenario initialization error: %%v\n", err)
		os.Exit(1)
	}
	if err := engine.Run(sc); err != nil {
		fmt.Fprintf(os.Stderr, "scenario execution error: %%v\n", err)
		os.Exit(1)
	}`, result.EntrypointName)
		} else {
			invocation = fmt.Sprintf(`	sc := scenario.%s()
	if err := engine.Run(sc); err != nil {
		fmt.Fprintf(os.Stderr, "scenario execution error: %%v\n", err)
		os.Exit(1)
	}`, result.EntrypointName)
		}
	default:
		return nil, fmt.Errorf("%w: unknown entrypoint kind: %s", model.ErrValidation, result.EntrypointKind)
	}

	content := fmt.Sprintf(`// Code generated by vuhive-cloud. DO NOT EDIT.
package main

import (
	"context"
	"flag"
	"fmt"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/morphy76/vuhive/pkg/vuhive"
	"%s/scenario"
)

func main() {
	summaryExport := flag.String("summary-export", "", "Path to export summary report")
	configPath := flag.String("config", "", "Path to configuration YAML")
	flag.Parse()

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	cfg := vuhive.EngineConfig{
		DefaultDuration: 30 * time.Second,
		DefaultVUs:      10,
	}
	_ = configPath
	_ = summaryExport
	_ = ctx

	engine := vuhive.NewEngine(cfg)

%s
}
`, result.ModuleName, invocation)

	return []byte(content), nil
}

// PrepareSourceArchive validates the archive, ensures files are placed in scenario/, injects main.go, and returns the new .tar.gz bytes.
func (a *StaticAnalyzer) PrepareSourceArchive(r io.Reader, opts StaticAnalysisOptions) ([]byte, *AnalysisResult, error) {
	res, err := a.AnalyzeArchive(r, opts)
	if err != nil {
		return nil, nil, err
	}

	mainContent, err := a.GenerateMainDriver(res)
	if err != nil {
		return nil, nil, err
	}

	var buf bytes.Buffer
	gw := gzip.NewWriter(&buf)
	tw := tar.NewWriter(gw)

	// Write go.mod
	if goModData, ok := res.Files["go.mod"]; ok {
		hdr := &tar.Header{
			Name: "go.mod",
			Mode: 0644,
			Size: int64(len(goModData)),
		}
		if err := tw.WriteHeader(hdr); err != nil {
			return nil, nil, err
		}
		if _, err := tw.Write(goModData); err != nil {
			return nil, nil, err
		}
	}

	// Write injected main.go
	mainHdr := &tar.Header{
		Name: "main.go",
		Mode: 0644,
		Size: int64(len(mainContent)),
	}
	if err := tw.WriteHeader(mainHdr); err != nil {
		return nil, nil, err
	}
	if _, err := tw.Write(mainContent); err != nil {
		return nil, nil, err
	}

	// Write all scenario Go files into scenario/
	for name, data := range res.Files {
		if name == "go.mod" || name == "main.go" {
			continue
		}
		var targetPath string
		if strings.HasSuffix(name, ".go") {
			base := path.Base(name)
			targetPath = "scenario/" + base
		} else {
			targetPath = name
		}

		hdr := &tar.Header{
			Name: targetPath,
			Mode: 0644,
			Size: int64(len(data)),
		}
		if err := tw.WriteHeader(hdr); err != nil {
			return nil, nil, err
		}
		if _, err := tw.Write(data); err != nil {
			return nil, nil, err
		}
	}

	if err := tw.Close(); err != nil {
		return nil, nil, err
	}
	if err := gw.Close(); err != nil {
		return nil, nil, err
	}

	return buf.Bytes(), res, nil
}

func (a *StaticAnalyzer) isForbiddenImport(importPath string) bool {
	for _, wl := range a.cfg.WhitelistedPackages {
		if importPath == wl || strings.HasPrefix(importPath, wl+"/") {
			return false
		}
	}
	for _, prefix := range defaultForbiddenPrefixes {
		if importPath == prefix || strings.HasPrefix(importPath, prefix+"/") {
			return true
		}
	}
	return false
}

func (a *StaticAnalyzer) findScenarioContract(file *ast.File) (name string, kind EntrypointKind, returnsErr bool) {
	// 1. Look for exported functions returning *vuhive.Scenario
	for _, decl := range file.Decls {
		fn, ok := decl.(*ast.FuncDecl)
		if !ok || !fn.Name.IsExported() {
			continue
		}

		// Register(engine *vuhive.Engine)
		if fn.Type.Params != nil && len(fn.Type.Params.List) == 1 {
			paramType := typeString(fn.Type.Params.List[0].Type)
			if strings.Contains(paramType, "vuhive.Engine") {
				return fn.Name.Name, EntrypointKindRegister, false
			}
		}

		// Returns *vuhive.Scenario or (*vuhive.Scenario, error)
		if fn.Type.Results != nil && len(fn.Type.Results.List) > 0 {
			firstRet := typeString(fn.Type.Results.List[0].Type)
			if strings.Contains(firstRet, "vuhive.Scenario") {
				hasErr := len(fn.Type.Results.List) > 1 && typeString(fn.Type.Results.List[1].Type) == "error"
				return fn.Name.Name, EntrypointKindFunction, hasErr
			}
		}
	}

	// 2. Look for package-level variables
	for _, decl := range file.Decls {
		gen, ok := decl.(*ast.GenDecl)
		if !ok || gen.Tok != token.VAR {
			continue
		}
		for _, spec := range gen.Specs {
			valSpec, ok := spec.(*ast.ValueSpec)
			if !ok {
				continue
			}
			for i, ident := range valSpec.Names {
				if !ident.IsExported() {
					continue
				}
				if valSpec.Type != nil && strings.Contains(typeString(valSpec.Type), "vuhive.Scenario") {
					return ident.Name, EntrypointKindVariable, false
				}
				if i < len(valSpec.Values) {
					if call, ok := valSpec.Values[i].(*ast.CallExpr); ok {
						callStr := typeString(call.Fun)
						if strings.Contains(callStr, "vuhive.NewScenario") {
							return ident.Name, EntrypointKindVariable, false
						}
					}
				}
			}
		}
	}

	// 3. Fallback: exported function containing vuhive.NewScenario call
	for _, decl := range file.Decls {
		fn, ok := decl.(*ast.FuncDecl)
		if !ok || !fn.Name.IsExported() || fn.Body == nil {
			continue
		}
		callsNewScenario := false
		ast.Inspect(fn.Body, func(n ast.Node) bool {
			if call, ok := n.(*ast.CallExpr); ok {
				if strings.Contains(typeString(call.Fun), "vuhive.NewScenario") {
					callsNewScenario = true
					return false
				}
			}
			return true
		})
		if callsNewScenario {
			return fn.Name.Name, EntrypointKindFunction, false
		}
	}

	return "", "", false
}

func typeString(expr ast.Expr) string {
	switch t := expr.(type) {
	case *ast.Ident:
		return t.Name
	case *ast.StarExpr:
		return "*" + typeString(t.X)
	case *ast.SelectorExpr:
		return typeString(t.X) + "." + t.Sel.Name
	default:
		return ""
	}
}

func extractArchive(r io.Reader) (map[string][]byte, error) {
	data, err := io.ReadAll(r)
	if err != nil {
		return nil, fmt.Errorf("%w: failed reading source archive: %v", model.ErrInvalidArchive, err)
	}
	if len(data) < 4 {
		return nil, fmt.Errorf("%w: archive too small", model.ErrInvalidArchive)
	}

	// Sniff magic bytes
	// Gzip: 0x1f 0x8b
	if data[0] == 0x1f && data[1] == 0x8b {
		return extractTarGz(bytes.NewReader(data))
	}

	// Zip: 0x50 0x4b (PK\x03\x04 or PK\x05\x06)
	if data[0] == 0x50 && data[1] == 0x4b {
		return extractZip(bytes.NewReader(data), int64(len(data)))
	}

	// Fallback attempts
	if files, err := extractTarGz(bytes.NewReader(data)); err == nil {
		return files, nil
	}
	if files, err := extractZip(bytes.NewReader(data), int64(len(data))); err == nil {
		return files, nil
	}

	return nil, fmt.Errorf("%w: unsupported archive format, must be .tar.gz or .zip", model.ErrInvalidArchive)
}

func extractZip(ra io.ReaderAt, size int64) (map[string][]byte, error) {
	zr, err := zip.NewReader(ra, size)
	if err != nil {
		return nil, fmt.Errorf("%w: invalid zip archive: %v", model.ErrInvalidArchive, err)
	}

	files := make(map[string][]byte)
	for _, f := range zr.File {
		if f.FileInfo().IsDir() {
			continue
		}

		rc, err := f.Open()
		if err != nil {
			return nil, fmt.Errorf("%w: error opening zip entry %s: %v", model.ErrInvalidArchive, f.Name, err)
		}

		var buf bytes.Buffer
		if _, err := io.Copy(&buf, rc); err != nil {
			_ = rc.Close()
			return nil, fmt.Errorf("%w: error reading zip entry %s: %v", model.ErrInvalidArchive, f.Name, err)
		}
		_ = rc.Close()

		cleanName := strings.TrimPrefix(path.Clean(f.Name), "/")
		files[cleanName] = buf.Bytes()
	}

	return files, nil
}

func extractTarGz(r io.Reader) (map[string][]byte, error) {
	gr, err := gzip.NewReader(r)
	if err != nil {
		return nil, fmt.Errorf("%w: invalid gzip header: %v", model.ErrInvalidArchive, err)
	}
	defer func() {
		_ = gr.Close()
	}()

	tr := tar.NewReader(gr)
	files := make(map[string][]byte)

	for {
		hdr, err := tr.Next()
		if err == io.EOF {
			break
		}
		if err != nil {
			return nil, fmt.Errorf("%w: tar read error: %v", model.ErrInvalidArchive, err)
		}

		if hdr.Typeflag == tar.TypeDir {
			continue
		}

		var buf bytes.Buffer
		if _, err := io.Copy(&buf, tr); err != nil {
			return nil, fmt.Errorf("%w: error reading entry %s: %v", model.ErrInvalidArchive, hdr.Name, err)
		}

		cleanName := strings.TrimPrefix(path.Clean(hdr.Name), "/")
		files[cleanName] = buf.Bytes()
	}

	return files, nil
}

func containsString(slice []string, s string) bool {
	for _, item := range slice {
		if item == s {
			return true
		}
	}
	return false
}

func parseGoMod(data []byte) (moduleName string, hasDirectVuhive bool, hasIndirectVuhive bool, err error) {
	scanner := bufio.NewScanner(bytes.NewReader(data))
	inRequireBlock := false

	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" || strings.HasPrefix(line, "//") {
			continue
		}

		if strings.HasPrefix(line, "module ") {
			parts := strings.Fields(line)
			if len(parts) >= 2 {
				moduleName = parts[1]
			}
			continue
		}

		if line == "require (" {
			inRequireBlock = true
			continue
		}
		if inRequireBlock && line == ")" {
			inRequireBlock = false
			continue
		}

		if strings.HasPrefix(line, "require ") && !inRequireBlock {
			rest := strings.TrimPrefix(line, "require ")
			rest = strings.TrimSpace(rest)
			checkVuhive(rest, &hasDirectVuhive, &hasIndirectVuhive)
			continue
		}

		if inRequireBlock {
			checkVuhive(line, &hasDirectVuhive, &hasIndirectVuhive)
		}
	}

	if err := scanner.Err(); err != nil {
		return "", false, false, err
	}
	if moduleName == "" {
		return "", false, false, fmt.Errorf("go.mod missing module declaration")
	}
	return moduleName, hasDirectVuhive, hasIndirectVuhive, nil
}

func checkVuhive(line string, direct, indirect *bool) {
	fields := strings.Fields(line)
	if len(fields) >= 2 && fields[0] == "github.com/morphy76/vuhive" {
		if strings.Contains(line, "// indirect") {
			*indirect = true
		} else {
			*direct = true
		}
	}
}

