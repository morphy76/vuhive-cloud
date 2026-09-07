package cli

import (
	"archive/tar"
	"bytes"
	"compress/gzip"
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"mime/multipart"
	"net/http"
	"os"
	"path/filepath"
	"strings"

	"github.com/morphy76/vuhive-cloud/internal/domain/model"
)

type SuiteCommand struct {
	store      CredentialStore
	httpClient *http.Client
}

func NewSuiteCommand(store CredentialStore, httpClient *http.Client) *SuiteCommand {
	return &SuiteCommand{
		store:      store,
		httpClient: httpClient,
	}
}

func (s *SuiteCommand) Execute(args []string, stdout, stderr io.Writer) int {
	if len(args) == 0 {
		fmt.Fprintln(stderr, "Usage: vuhive suite <upload> [flags] <path>")
		return 1
	}

	switch args[0] {
	case "upload":
		return s.upload(args[1:], stdout, stderr)
	default:
		fmt.Fprintf(stderr, "Unknown suite subcommand: %s\n", args[0])
		return 1
	}
}

func (s *SuiteCommand) upload(args []string, stdout, stderr io.Writer) int {
	fs := flag.NewFlagSet("upload", flag.ContinueOnError)
	fs.SetOutput(stderr)

	suiteIDFlag := fs.String("suite-id", "", "Target test suite UUID (required)")
	platformFlag := fs.String("platform", "linux/amd64", "Target runner compilation platform (linux/amd64 or linux/arm64)")
	serverFlag := fs.String("server", "", "Control plane server URL")
	insecureFlag := fs.Bool("insecure", false, "Request insecure imports policy override")

	if err := fs.Parse(args); err != nil {
		return 1
	}

	positional := fs.Args()
	if len(positional) == 0 {
		fmt.Fprintln(stderr, "Error: missing source archive or directory path")
		return 1
	}
	sourcePath := positional[0]

	suiteID := strings.TrimSpace(*suiteIDFlag)
	if suiteID == "" {
		fmt.Fprintln(stderr, "Error: --suite-id is required")
		return 1
	}

	// Verify developer role guard locally before sending
	creds, _ := s.store.Load()
	if creds != nil && creds.AccessToken != "" {
		if claims, err := creds.Claims(); err == nil && claims != nil {
			if !claims.HasRole(model.RoleDeveloper) && !claims.HasRole(model.RoleAdmin) {
				fmt.Fprintf(stderr, "Error: role guard violation: user %q lacks required 'vuhive-developer' role\n", claims.Username())
				return 1
			}
		}
	}

	serverURL := *serverFlag
	if serverURL == "" && creds != nil {
		serverURL = creds.ServerURL
	}
	client := NewAPIClient(serverURL, s.store, s.httpClient)

	var tarData []byte
	fi, err := os.Stat(sourcePath)
	if err != nil {
		fmt.Fprintf(stderr, "Error accessing source path %q: %v\n", sourcePath, err)
		return 1
	}

	if fi.IsDir() {
		data, err := createTarGzFromDir(sourcePath)
		if err != nil {
			fmt.Fprintf(stderr, "Error packaging source directory: %v\n", err)
			return 1
		}
		tarData = data
	} else {
		data, err := os.ReadFile(sourcePath)
		if err != nil {
			fmt.Fprintf(stderr, "Error reading source archive: %v\n", err)
			return 1
		}
		tarData = data
	}

	body := &bytes.Buffer{}
	writer := multipart.NewWriter(body)

	part, err := writer.CreateFormFile("source", filepath.Base(sourcePath)+".tar.gz")
	if err != nil {
		fmt.Fprintf(stderr, "Error creating multipart file: %v\n", err)
		return 1
	}
	if _, err := part.Write(tarData); err != nil {
		fmt.Fprintf(stderr, "Error writing multipart data: %v\n", err)
		return 1
	}

	_ = writer.WriteField("platform", *platformFlag)
	if *insecureFlag {
		_ = writer.WriteField("allow_insecure_imports", "true")
	}
	_ = writer.Close()

	uploadPath := fmt.Sprintf("/api/v1/suites/%s/builds", suiteID)
	req, err := http.NewRequestWithContext(context.Background(), http.MethodPost, client.ServerURL()+uploadPath, body)
	if err != nil {
		fmt.Fprintf(stderr, "Error creating upload request: %v\n", err)
		return 1
	}
	req.Header.Set("Content-Type", writer.FormDataContentType())

	resp, err := client.Do(context.Background(), req)
	if err != nil {
		fmt.Fprintf(stderr, "Upload failed: %v\n", err)
		return 1
	}
	defer func() { _ = resp.Body.Close() }()

	respBody, _ := io.ReadAll(resp.Body)
	if resp.StatusCode != http.StatusAccepted && resp.StatusCode != http.StatusOK {
		fmt.Fprintf(stderr, "Build initiation failed (HTTP %d): %s\n", resp.StatusCode, string(respBody))
		return 1
	}

	var result struct {
		Message   string `json:"message"`
		Artifacts []struct {
			ID       string `json:"id"`
			Platform string `json:"platform"`
			Status   string `json:"status"`
		} `json:"artifacts"`
	}
	_ = json.Unmarshal(respBody, &result)

	fmt.Fprintln(stdout, "=== Source Uploaded & Compilation Initiated ===")
	fmt.Fprintf(stdout, "Message:   %s\n", result.Message)
	for _, a := range result.Artifacts {
		fmt.Fprintf(stdout, "Artifact:  %s (Platform: %s, Status: %s)\n", a.ID, a.Platform, a.Status)
	}

	return 0
}

func createTarGzFromDir(srcDir string) ([]byte, error) {
	var buf bytes.Buffer
	gw := gzip.NewWriter(&buf)
	tw := tar.NewWriter(gw)

	err := filepath.Walk(srcDir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		relPath, err := filepath.Rel(srcDir, path)
		if err != nil {
			return err
		}
		if relPath == "." {
			return nil
		}

		header, err := tar.FileInfoHeader(info, info.Name())
		if err != nil {
			return err
		}
		header.Name = filepath.ToSlash(relPath)

		if err := tw.WriteHeader(header); err != nil {
			return err
		}

		if info.Mode().IsRegular() {
			f, err := os.Open(path)
			if err != nil {
				return err
			}
			defer f.Close()
			if _, err := io.Copy(tw, f); err != nil {
				return err
			}
		}
		return nil
	})

	if err != nil {
		return nil, err
	}
	if err := tw.Close(); err != nil {
		return nil, err
	}
	if err := gw.Close(); err != nil {
		return nil, err
	}

	return buf.Bytes(), nil
}
