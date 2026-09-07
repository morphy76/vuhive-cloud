package cli

import (
	"context"
	"encoding/json"
	"flag"
	"io"
	"net/http"
	"strings"

	"github.com/morphy76/vuhive-cloud/internal/domain/model"
)

type RunCommand struct {
	store      CredentialStore
	httpClient *http.Client
}

func NewRunCommand(store CredentialStore, httpClient *http.Client) *RunCommand {
	return &RunCommand{
		store:      store,
		httpClient: httpClient,
	}
}

func (r *RunCommand) Execute(args []string, stdout, stderr io.Writer) int {
	if len(args) == 0 {
		fprintln(stderr, "Usage: vuhive run <start|status|logs|report> [flags]")
		return 1
	}

	switch args[0] {
	case "start":
		return r.start(args[1:], stdout, stderr)
	case "status":
		return r.status(args[1:], stdout, stderr)
	case "logs":
		return r.logs(args[1:], stdout, stderr)
	case "report":
		return r.report(args[1:], stdout, stderr)
	default:
		fprintf(stderr, "Unknown run subcommand: %s\n", args[0])
		return 1
	}
}

func splitFlagsAndPositional(args []string) ([]string, []string) {
	var flags []string
	var positional []string
	for i := 0; i < len(args); i++ {
		arg := args[i]
		if strings.HasPrefix(arg, "-") {
			flags = append(flags, arg)
			if !strings.Contains(arg, "=") && i+1 < len(args) && !strings.HasPrefix(args[i+1], "-") {
				i++
				flags = append(flags, args[i])
			}
		} else {
			positional = append(positional, arg)
		}
	}
	return flags, positional
}

func (r *RunCommand) start(args []string, stdout, stderr io.Writer) int {
	// Deployer role guard check
	creds, _ := r.store.Load()
	if creds != nil && creds.AccessToken != "" {
		if claims, err := creds.Claims(); err == nil && claims != nil {
			if !claims.HasRole(model.RoleDeployer) && !claims.HasRole(model.RoleAdmin) {
				fprintf(stderr, "Error: role guard violation: user %q lacks required 'vuhive-deployer' role\n", claims.Username())
				return 1
			}
		}
	}

	flags, positional := splitFlagsAndPositional(args)

	fs := flag.NewFlagSet("start", flag.ContinueOnError)
	fs.SetOutput(stderr)

	profileIDFlag := fs.String("profile-id", "", "Runner profile UUID (required)")
	artifactIDFlag := fs.String("artifact-id", "", "Compiled artifact UUID (required)")
	serverFlag := fs.String("server", "", "Control plane server URL")

	if err := fs.Parse(flags); err != nil {
		return 1
	}

	if len(positional) == 0 {
		fprintln(stderr, "Error: missing suite ID argument. Usage: vuhive run start <suite-id> --profile-id <id> --artifact-id <id>")
		return 1
	}
	suiteID := strings.TrimSpace(positional[0])

	profileID := strings.TrimSpace(*profileIDFlag)
	artifactID := strings.TrimSpace(*artifactIDFlag)
	if profileID == "" || artifactID == "" {
		fprintln(stderr, "Error: --profile-id and --artifact-id are required")
		return 1
	}

	serverURL := *serverFlag
	if serverURL == "" && creds != nil {
		serverURL = creds.ServerURL
	}
	client := NewAPIClient(serverURL, r.store, r.httpClient)

	payload := map[string]interface{}{
		"suite_id":          suiteID,
		"artifact_id":       artifactID,
		"runner_profile_id": profileID,
	}

	resp, err := client.PostJSON(context.Background(), "/api/v1/runs", payload)
	if err != nil {
		fprintf(stderr, "Failed triggering run: %v\n", err)
		return 1
	}
	defer func() { _ = resp.Body.Close() }()

	body, _ := io.ReadAll(resp.Body)
	if resp.StatusCode != http.StatusCreated {
		fprintf(stderr, "Run trigger failed (HTTP %d): %s\n", resp.StatusCode, string(body))
		return 1
	}

	var runResp struct {
		ID        string `json:"id"`
		SuiteID   string `json:"suite_id"`
		Status    string `json:"status"`
		CreatedAt string `json:"created_at"`
	}
	_ = json.Unmarshal(body, &runResp)

	fprintln(stdout, "=== Test Run Triggered ===")
	fprintf(stdout, "Run ID:     %s\n", runResp.ID)
	fprintf(stdout, "Suite ID:   %s\n", runResp.SuiteID)
	fprintf(stdout, "Status:     %s\n", runResp.Status)
	fprintf(stdout, "Created At: %s\n", runResp.CreatedAt)

	return 0
}

func (r *RunCommand) status(args []string, stdout, stderr io.Writer) int {
	flags, positional := splitFlagsAndPositional(args)
	fs := flag.NewFlagSet("status", flag.ContinueOnError)
	fs.SetOutput(stderr)
	serverFlag := fs.String("server", "", "Control plane server URL")
	if err := fs.Parse(flags); err != nil {
		return 1
	}

	if len(positional) == 0 {
		fprintln(stderr, "Error: missing run ID. Usage: vuhive run status <run-id>")
		return 1
	}
	runID := strings.TrimSpace(positional[0])

	creds, _ := r.store.Load()
	serverURL := *serverFlag
	if serverURL == "" && creds != nil {
		serverURL = creds.ServerURL
	}
	client := NewAPIClient(serverURL, r.store, r.httpClient)

	resp, err := client.Get(context.Background(), "/api/v1/runs/"+runID)
	if err != nil {
		fprintf(stderr, "Failed fetching run status: %v\n", err)
		return 1
	}
	defer func() { _ = resp.Body.Close() }()

	body, _ := io.ReadAll(resp.Body)
	if resp.StatusCode != http.StatusOK {
		fprintf(stderr, "Get run failed (HTTP %d): %s\n", resp.StatusCode, string(body))
		return 1
	}

	var runResp struct {
		ID        string `json:"id"`
		SuiteID   string `json:"suite_id"`
		Status    string `json:"status"`
		ExitCode  *int   `json:"exit_code"`
		SLAPassed *bool  `json:"sla_passed"`
	}
	_ = json.Unmarshal(body, &runResp)

	fprintln(stdout, "=== Test Run Details ===")
	fprintf(stdout, "Run ID:     %s\n", runResp.ID)
	fprintf(stdout, "Suite ID:   %s\n", runResp.SuiteID)
	fprintf(stdout, "Status:     %s\n", runResp.Status)
	if runResp.ExitCode != nil {
		fprintf(stdout, "Exit Code:  %d\n", *runResp.ExitCode)
	}
	if runResp.SLAPassed != nil {
		fprintf(stdout, "SLA Passed: %v\n", *runResp.SLAPassed)
	}

	return 0
}

func (r *RunCommand) logs(args []string, stdout, stderr io.Writer) int {
	flags, positional := splitFlagsAndPositional(args)
	fs := flag.NewFlagSet("logs", flag.ContinueOnError)
	fs.SetOutput(stderr)
	serverFlag := fs.String("server", "", "Control plane server URL")
	if err := fs.Parse(flags); err != nil {
		return 1
	}

	if len(positional) == 0 {
		fprintln(stderr, "Error: missing run ID. Usage: vuhive run logs <run-id>")
		return 1
	}
	runID := strings.TrimSpace(positional[0])

	creds, _ := r.store.Load()
	serverURL := *serverFlag
	if serverURL == "" && creds != nil {
		serverURL = creds.ServerURL
	}
	client := NewAPIClient(serverURL, r.store, r.httpClient)

	resp, err := client.Get(context.Background(), "/api/v1/runs/"+runID+"/logs")
	if err != nil {
		fprintf(stderr, "Failed fetching run logs: %v\n", err)
		return 1
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		fprintf(stderr, "Failed fetching logs (HTTP %d): %s\n", resp.StatusCode, string(body))
		return 1
	}

	_, _ = io.Copy(stdout, resp.Body)
	return 0
}

func (r *RunCommand) report(args []string, stdout, stderr io.Writer) int {
	flags, positional := splitFlagsAndPositional(args)
	fs := flag.NewFlagSet("report", flag.ContinueOnError)
	fs.SetOutput(stderr)
	serverFlag := fs.String("server", "", "Control plane server URL")
	if err := fs.Parse(flags); err != nil {
		return 1
	}

	if len(positional) == 0 {
		fprintln(stderr, "Error: missing run ID. Usage: vuhive run report <run-id>")
		return 1
	}
	runID := strings.TrimSpace(positional[0])

	creds, _ := r.store.Load()
	serverURL := *serverFlag
	if serverURL == "" && creds != nil {
		serverURL = creds.ServerURL
	}
	client := NewAPIClient(serverURL, r.store, r.httpClient)

	resp, err := client.Get(context.Background(), "/api/v1/runs/"+runID+"/report")
	if err != nil {
		fprintf(stderr, "Failed fetching report: %v\n", err)
		return 1
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		fprintf(stderr, "Failed fetching report (HTTP %d): %s\n", resp.StatusCode, string(body))
		return 1
	}

	_, _ = io.Copy(stdout, resp.Body)
	return 0
}
