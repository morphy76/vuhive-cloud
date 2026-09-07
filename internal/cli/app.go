package cli

import (
	"io"
	"net/http"

	"github.com/morphy76/vuhive-cloud/internal/version"
)

type App struct {
	store      CredentialStore
	httpClient *http.Client
}

func NewApp(store CredentialStore) *App {
	return NewAppWithClient(store, http.DefaultClient)
}

func NewAppWithClient(store CredentialStore, httpClient *http.Client) *App {
	if store == nil {
		store = NewFileCredentialStore("")
	}
	if httpClient == nil {
		httpClient = http.DefaultClient
	}
	return &App{
		store:      store,
		httpClient: httpClient,
	}
}

func (a *App) Execute(args []string, stdout, stderr io.Writer) int {
	if len(args) == 0 {
		a.printHelp(stdout)
		return 0
	}

	cmd := args[0]
	subArgs := args[1:]

	switch cmd {
	case "auth":
		authCmd := NewAuthCommand(a.store, a.httpClient)
		return authCmd.Execute(subArgs, stdout, stderr)

	case "suite":
		suiteCmd := NewSuiteCommand(a.store, a.httpClient)
		return suiteCmd.Execute(subArgs, stdout, stderr)

	case "run":
		runCmd := NewRunCommand(a.store, a.httpClient)
		return runCmd.Execute(subArgs, stdout, stderr)

	case "schedule":
		schedCmd := NewScheduleCommand(a.store, a.httpClient)
		return schedCmd.Execute(subArgs, stdout, stderr)

	case "version", "--version", "-v":
		fprintf(stdout, "vuhive CLI %s (commit: %s, built: %s)\n", version.Version, version.Commit, version.BuildTime)
		return 0

	case "help", "--help", "-h":
		a.printHelp(stdout)
		return 0

	default:
		fprintf(stderr, "Unknown command: %s\nRun 'vuhive --help' for usage.\n", cmd)
		return 1
	}
}

func (a *App) printHelp(w io.Writer) {
	fprintln(w, `vuhive — Developer CLI for the vuhive-cloud load testing orchestration platform

Usage:
  vuhive <command> [subcommand] [flags]

Commands:
  auth        Manage authentication and credentials
    login     Log in via browser OAuth2 Authorization Code flow (with PKCE & device fallback)
    status    Display current authentication status, active roles, and token expiration
    logout    Revoke credentials and clear local session

  suite       Manage load test scenarios and source packages
    upload    Upload and compile test suite source package (Requires role: vuhive-developer)

  run         Manage and inspect test runs
    start     Dispatch an ad-hoc test run (Requires role: vuhive-deployer)
    status    Inspect execution status and indexed performance KPIs
    logs      Stream or fetch execution logs
    report    Fetch deterministic JSON summary report

  schedule    Manage native Kubernetes CronJob schedules
    create    Create scheduled load test execution (Requires role: vuhive-deployer)
    list      List configured CronJob schedules (Requires role: vuhive-viewer)

  version     Print CLI version and build metadata
  help        Show help documentation

Global Options:
  -h, --help     Show help for command
  -v, --version  Show CLI version`)
}
