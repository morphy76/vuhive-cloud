package cli

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"net/http"
	"strings"

	"github.com/morphy76/vuhive-cloud/internal/domain/model"
)

type ScheduleCommand struct {
	store      CredentialStore
	httpClient *http.Client
}

func NewScheduleCommand(store CredentialStore, httpClient *http.Client) *ScheduleCommand {
	return &ScheduleCommand{
		store:      store,
		httpClient: httpClient,
	}
}

func (s *ScheduleCommand) Execute(args []string, stdout, stderr io.Writer) int {
	if len(args) == 0 {
		fmt.Fprintln(stderr, "Usage: vuhive schedule <create|list> [flags]")
		return 1
	}

	switch args[0] {
	case "create":
		return s.create(args[1:], stdout, stderr)
	case "list":
		return s.list(args[1:], stdout, stderr)
	default:
		fmt.Fprintf(stderr, "Unknown schedule subcommand: %s\n", args[0])
		return 1
	}
}

func (s *ScheduleCommand) create(args []string, stdout, stderr io.Writer) int {
	fs := flag.NewFlagSet("create", flag.ContinueOnError)
	fs.SetOutput(stderr)

	nameFlag := fs.String("name", "", "Schedule name (required)")
	suiteIDFlag := fs.String("suite-id", "", "Target test suite UUID (required)")
	profileIDFlag := fs.String("profile-id", "", "Runner profile UUID (required)")
	artifactIDFlag := fs.String("artifact-id", "", "Compiled artifact UUID (required)")
	cronFlag := fs.String("cron", "", "Standard 5-field cron expression (required, e.g. '0 2 * * *')")
	serverFlag := fs.String("server", "", "Control plane server URL")

	if err := fs.Parse(args); err != nil {
		return 1
	}

	name := strings.TrimSpace(*nameFlag)
	suiteID := strings.TrimSpace(*suiteIDFlag)
	profileID := strings.TrimSpace(*profileIDFlag)
	artifactID := strings.TrimSpace(*artifactIDFlag)
	cronExpr := strings.TrimSpace(*cronFlag)

	if name == "" || suiteID == "" || profileID == "" || artifactID == "" || cronExpr == "" {
		fmt.Fprintln(stderr, "Error: --name, --suite-id, --profile-id, --artifact-id, and --cron are all required")
		return 1
	}

	// Deployer role guard check
	creds, _ := s.store.Load()
	if creds != nil && creds.AccessToken != "" {
		if claims, err := creds.Claims(); err == nil && claims != nil {
			if !claims.HasRole(model.RoleDeployer) && !claims.HasRole(model.RoleAdmin) {
				fmt.Fprintf(stderr, "Error: role guard violation: user %q lacks required 'vuhive-deployer' role\n", claims.Username())
				return 1
			}
		}
	}

	serverURL := *serverFlag
	if serverURL == "" && creds != nil {
		serverURL = creds.ServerURL
	}
	client := NewAPIClient(serverURL, s.store, s.httpClient)

	payload := map[string]interface{}{
		"name":              name,
		"suite_id":          suiteID,
		"artifact_id":       artifactID,
		"runner_profile_id": profileID,
		"cron_expression":   cronExpr,
	}

	resp, err := client.PostJSON(context.Background(), "/api/v1/schedules", payload)
	if err != nil {
		fmt.Fprintf(stderr, "Failed creating schedule: %v\n", err)
		return 1
	}
	defer func() { _ = resp.Body.Close() }()

	body, _ := io.ReadAll(resp.Body)
	if resp.StatusCode != http.StatusCreated {
		fmt.Fprintf(stderr, "Create schedule failed (HTTP %d): %s\n", resp.StatusCode, string(body))
		return 1
	}

	var schedResp struct {
		ID             string `json:"id"`
		Name           string `json:"name"`
		CronExpression string `json:"cron_expression"`
		Status         string `json:"status"`
	}
	_ = json.Unmarshal(body, &schedResp)

	fmt.Fprintln(stdout, "=== CronJob Schedule Created ===")
	fmt.Fprintf(stdout, "Schedule ID: %s\n", schedResp.ID)
	fmt.Fprintf(stdout, "Name:        %s\n", schedResp.Name)
	fmt.Fprintf(stdout, "Cron:        %s\n", schedResp.CronExpression)
	fmt.Fprintf(stdout, "Status:      %s\n", schedResp.Status)

	return 0
}

func (s *ScheduleCommand) list(args []string, stdout, stderr io.Writer) int {
	fs := flag.NewFlagSet("list", flag.ContinueOnError)
	fs.SetOutput(stderr)
	serverFlag := fs.String("server", "", "Control plane server URL")
	if err := fs.Parse(args); err != nil {
		return 1
	}

	creds, _ := s.store.Load()
	serverURL := *serverFlag
	if serverURL == "" && creds != nil {
		serverURL = creds.ServerURL
	}
	client := NewAPIClient(serverURL, s.store, s.httpClient)

	resp, err := client.Get(context.Background(), "/api/v1/schedules")
	if err != nil {
		fmt.Fprintf(stderr, "Failed fetching schedules: %v\n", err)
		return 1
	}
	defer func() { _ = resp.Body.Close() }()

	body, _ := io.ReadAll(resp.Body)
	if resp.StatusCode != http.StatusOK {
		fmt.Fprintf(stderr, "List schedules failed (HTTP %d): %s\n", resp.StatusCode, string(body))
		return 1
	}

	var listResp struct {
		Items []struct {
			ID             string `json:"id"`
			Name           string `json:"name"`
			CronExpression string `json:"cron_expression"`
			Status         string `json:"status"`
		} `json:"items"`
	}
	_ = json.Unmarshal(body, &listResp)

	fmt.Fprintln(stdout, "=== Schedules ===")
	for _, item := range listResp.Items {
		fmt.Fprintf(stdout, "- [%s] %s (%s) — %s\n", item.Status, item.Name, item.CronExpression, item.ID)
	}

	return 0
}
