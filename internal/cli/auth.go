package cli

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/url"
	"os/exec"
	"runtime"
	"strings"
	"time"

	"github.com/zitadel/oidc/v3/pkg/client/rp"
	"github.com/zitadel/oidc/v3/pkg/oidc"
	"golang.org/x/oauth2"
)

type AuthCommand struct {
	store      CredentialStore
	httpClient *http.Client
}

func NewAuthCommand(store CredentialStore, httpClient *http.Client) *AuthCommand {
	if httpClient == nil {
		httpClient = &http.Client{Timeout: 30 * time.Second}
	}
	return &AuthCommand{
		store:      store,
		httpClient: httpClient,
	}
}

func (a *AuthCommand) Execute(args []string, stdout, stderr io.Writer) int {
	if len(args) == 0 {
		fmt.Fprintln(stderr, "Usage: vuhive auth <login|status|logout> [flags]")
		return 1
	}

	switch args[0] {
	case "login":
		return a.login(args[1:], stdout, stderr)
	case "status":
		return a.status(args[1:], stdout, stderr)
	case "logout":
		return a.logout(args[1:], stdout, stderr)
	default:
		fmt.Fprintf(stderr, "Unknown auth subcommand: %s\n", args[0])
		return 1
	}
}

func (a *AuthCommand) status(_ []string, stdout, stderr io.Writer) int {
	creds, err := a.store.Load()
	if err != nil {
		fmt.Fprintf(stderr, "Error reading credentials: %v\n", err)
		return 1
	}

	if creds == nil || creds.AccessToken == "" {
		fmt.Fprintln(stdout, "Not authenticated. Run 'vuhive auth login' to authenticate.")
		return 0
	}

	claims, err := creds.Claims()
	if err != nil {
		fmt.Fprintf(stderr, "Error parsing token claims: %v\n", err)
		return 1
	}

	fmt.Fprintln(stdout, "=== vuhive-cloud Authentication Status ===")
	fmt.Fprintf(stdout, "User:       %s (%s)\n", claims.Username(), claims.Subject())
	if claims.Email() != "" {
		fmt.Fprintf(stdout, "Email:      %s\n", claims.Email())
	}
	fmt.Fprintf(stdout, "Roles:      %s\n", strings.Join(claims.Roles(), ", "))
	if len(claims.Groups()) > 0 {
		fmt.Fprintf(stdout, "Groups:     %s\n", strings.Join(claims.Groups(), ", "))
	}
	if creds.ServerURL != "" {
		fmt.Fprintf(stdout, "Server URL: %s\n", creds.ServerURL)
	}

	if claims.IsExpired() {
		fmt.Fprintf(stdout, "Status:     EXPIRED (expired at %s). Run 'vuhive auth login' to refresh.\n", claims.ExpiresAt().Format(time.RFC3339))
	} else {
		fmt.Fprintf(stdout, "Status:     ACTIVE (expires at %s)\n", claims.ExpiresAt().Format(time.RFC3339))
	}

	return 0
}

func (a *AuthCommand) logout(_ []string, stdout, stderr io.Writer) int {
	creds, _ := a.store.Load()
	if creds != nil && creds.RefreshToken != "" && creds.IssuerURL != "" {
		revokeURL := strings.TrimRight(creds.IssuerURL, "/") + "/protocol/openid-connect/revoke"
		data := url.Values{}
		data.Set("token", creds.RefreshToken)
		data.Set("token_type_hint", "refresh_token")
		data.Set("client_id", "vuhive-cloud-cli")

		req, err := http.NewRequest(http.MethodPost, revokeURL, strings.NewReader(data.Encode()))
		if err == nil {
			req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
			resp, doErr := a.httpClient.Do(req)
			if doErr == nil {
				_ = resp.Body.Close()
			}
		}
	}

	if err := a.store.Clear(); err != nil {
		fmt.Fprintf(stderr, "Error clearing credentials: %v\n", err)
		return 1
	}

	fmt.Fprintln(stdout, "Successfully logged out and cleared credentials.")
	return 0
}

func (a *AuthCommand) login(args []string, stdout, stderr io.Writer) int {
	fs := flag.NewFlagSet("login", flag.ContinueOnError)
	fs.SetOutput(stderr)

	issuerFlag := fs.String("issuer", "http://localhost:8080/realms/vuhive", "Keycloak OIDC realm issuer URL")
	serverFlag := fs.String("server", "http://localhost:8080", "vuhive-cloud control plane server URL")
	clientIDFlag := fs.String("client-id", "vuhive-cloud-cli", "OIDC client identifier")
	noBrowserFlag := fs.Bool("no-browser", false, "Do not attempt to open browser automatically")
	deviceFlag := fs.Bool("device", false, "Use OAuth2 Device Authorization Flow instead of browser authorization code flow")

	if err := fs.Parse(args); err != nil {
		return 1
	}

	issuerURL := strings.TrimRight(*issuerFlag, "/")
	serverURL := strings.TrimRight(*serverFlag, "/")
	clientID := *clientIDFlag

	if *deviceFlag {
		return a.deviceFlow(issuerURL, serverURL, clientID, stdout, stderr)
	}

	// 1. Generate PKCE verifier and challenge
	verifierBytes := make([]byte, 32)
	_, _ = rand.Read(verifierBytes)
	codeVerifier := base64.RawURLEncoding.EncodeToString(verifierBytes)

	h := sha256.Sum256([]byte(codeVerifier))
	codeChallenge := base64.RawURLEncoding.EncodeToString(h[:])

	// 2. Generate random state
	stateBytes := make([]byte, 16)
	_, _ = rand.Read(stateBytes)
	expectedState := base64.RawURLEncoding.EncodeToString(stateBytes)

	// 3. Start local loopback HTTP listener
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		fmt.Fprintf(stderr, "Failed starting local loopback server: %v\nFalling back to Device Flow...\n", err)
		return a.deviceFlow(issuerURL, serverURL, clientID, stdout, stderr)
	}
	defer func() { _ = listener.Close() }()

	localPort := listener.Addr().(*net.TCPAddr).Port
	redirectURI := fmt.Sprintf("http://127.0.0.1:%d/callback", localPort)

	oauthCfg := &oauth2.Config{
		ClientID: clientID,
		Endpoint: oauth2.Endpoint{
			AuthURL:  issuerURL + "/protocol/openid-connect/auth",
			TokenURL: issuerURL + "/protocol/openid-connect/token",
		},
		RedirectURL: redirectURI,
		Scopes:      []string{"openid", "profile", "email"},
	}

	var rpOpts []rp.Option
	if a.httpClient != nil {
		rpOpts = append(rpOpts, rp.WithHTTPClient(a.httpClient))
	}
	rpOpts = append(rpOpts, rp.WithAuthStyle(oauth2.AuthStyleInParams))

	relyingParty, err := rp.NewRelyingPartyOAuth(oauthCfg, rpOpts...)
	if err != nil {
		fmt.Fprintf(stderr, "Failed initializing OIDC RelyingParty: %v\n", err)
		return 1
	}

	authURL := rp.AuthURL(expectedState, relyingParty, rp.WithCodeChallenge(codeChallenge))

	codeChan := make(chan string, 1)
	errChan := make(chan error, 1)

	server := &http.Server{
		Handler: http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if r.URL.Path != "/callback" {
				http.NotFound(w, r)
				return
			}

			q := r.URL.Query()
			state := q.Get("state")
			if state != expectedState {
				errChan <- errors.New("state mismatch; possible CSRF detected")
				w.WriteHeader(http.StatusBadRequest)
				_, _ = w.Write([]byte("<h1>Authentication Failed</h1><p>State verification failed.</p>"))
				return
			}

			if errParam := q.Get("error"); errParam != "" {
				desc := q.Get("error_description")
				errChan <- fmt.Errorf("oauth error %s: %s", errParam, desc)
				w.WriteHeader(http.StatusBadRequest)
				_, _ = fmt.Fprintf(w, "<h1>Authentication Failed</h1><p>%s: %s</p>", errParam, desc)
				return
			}

			code := q.Get("code")
			if code == "" {
				errChan <- errors.New("no authorization code returned")
				w.WriteHeader(http.StatusBadRequest)
				_, _ = w.Write([]byte("<h1>Authentication Failed</h1><p>No authorization code received.</p>"))
				return
			}

			w.Header().Set("Content-Type", "text/html")
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write([]byte(`<!DOCTYPE html><html><body style="font-family:sans-serif;text-align:center;padding:50px;">` +
				`<h2>Authentication Successful!</h2><p>You can now close this tab and return to your terminal.</p></body></html>`))

			codeChan <- code
		}),
	}

	go func() {
		_ = server.Serve(listener)
	}()

	fmt.Fprintln(stdout, "Opening browser for authentication...")
	fmt.Fprintf(stdout, "If browser does not open automatically, navigate to:\n%s\n\n", authURL)

	if !*noBrowserFlag {
		_ = openBrowser(authURL)
	}

	var code string
	select {
	case code = <-codeChan:
	case err := <-errChan:
		fmt.Fprintf(stderr, "Authentication failed: %v\n", err)
		return 1
	case <-time.After(2 * time.Minute):
		fmt.Fprintln(stderr, "Authentication timed out waiting for browser callback.")
		return 1
	}

	// 4. Exchange authorization code for tokens via zitadel RelyingParty
	tokens, err := rp.CodeExchange[*oidc.IDTokenClaims](context.Background(), code, relyingParty, rp.WithCodeVerifier(codeVerifier))
	if err != nil {
		fmt.Fprintf(stderr, "Token exchange failed: %v\n", err)
		return 1
	}

	creds := &Credentials{
		AccessToken:  tokens.AccessToken,
		RefreshToken: tokens.RefreshToken,
		IDToken:      tokens.IDToken,
		TokenType:    tokens.TokenType,
		IssuerURL:    issuerURL,
		ServerURL:    serverURL,
		ExpiresAt:    tokens.Expiry,
	}
	if creds.TokenType == "" {
		creds.TokenType = "Bearer"
	}

	if err := a.store.Save(creds); err != nil {
		fmt.Fprintf(stderr, "Failed saving credentials: %v\n", err)
		return 1
	}

	claims, err := creds.Claims()
	if err == nil && claims != nil {
		fmt.Fprintf(stdout, "Successfully authenticated as %s (%s)!\n", claims.Username(), claims.Email())
		fmt.Fprintf(stdout, "Assigned Roles: %s\n", strings.Join(claims.Roles(), ", "))
		if len(claims.Groups()) > 0 {
			fmt.Fprintf(stdout, "Groups:         %s\n", strings.Join(claims.Groups(), ", "))
		}
	} else {
		fmt.Fprintln(stdout, "Successfully authenticated!")
	}

	return 0
}

func (a *AuthCommand) deviceFlow(issuerURL, serverURL, clientID string, stdout, stderr io.Writer) int {
	deviceURL := issuerURL + "/protocol/openid-connect/auth/device"
	data := url.Values{}
	data.Set("client_id", clientID)
	data.Set("scope", "openid profile email")

	req, err := http.NewRequest(http.MethodPost, deviceURL, strings.NewReader(data.Encode()))
	if err != nil {
		fmt.Fprintf(stderr, "Failed creating device authorization request: %v\n", err)
		return 1
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	resp, err := a.httpClient.Do(req)
	if err != nil {
		fmt.Fprintf(stderr, "Failed requesting device code: %v\n", err)
		return 1
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		fmt.Fprintf(stderr, "Device flow request failed (HTTP %d): %s\n", resp.StatusCode, string(body))
		return 1
	}

	var devResp struct {
		DeviceCode              string `json:"device_code"`
		UserCode                string `json:"user_code"`
		VerificationURI         string `json:"verification_uri"`
		VerificationURIComplete string `json:"verification_uri_complete"`
		ExpiresIn               int    `json:"expires_in"`
		Interval                int    `json:"interval"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&devResp); err != nil {
		fmt.Fprintf(stderr, "Failed decoding device authorization response: %v\n", err)
		return 1
	}

	uri := devResp.VerificationURIComplete
	if uri == "" {
		uri = devResp.VerificationURI
	}

	fmt.Fprintln(stdout, "=== Device Authorization Flow ===")
	fmt.Fprintf(stdout, "Navigate to: %s\n", uri)
	fmt.Fprintf(stdout, "Enter User Code: %s\n\n", devResp.UserCode)
	fmt.Fprintln(stdout, "Waiting for verification in browser...")

	interval := time.Duration(devResp.Interval) * time.Second
	if interval <= 0 {
		interval = 5 * time.Second
	}
	deadline := time.Now().Add(time.Duration(devResp.ExpiresIn) * time.Second)

	tokenURL := issuerURL + "/protocol/openid-connect/token"

	for time.Now().Before(deadline) {
		time.Sleep(interval)

		pollData := url.Values{}
		pollData.Set("grant_type", "urn:ietf:params:oauth:grant-type:device_code")
		pollData.Set("client_id", clientID)
		pollData.Set("device_code", devResp.DeviceCode)

		pollReq, _ := http.NewRequest(http.MethodPost, tokenURL, strings.NewReader(pollData.Encode()))
		pollReq.Header.Set("Content-Type", "application/x-www-form-urlencoded")

		pollResp, pollErr := a.httpClient.Do(pollReq)
		if pollErr != nil {
			continue
		}

		if pollResp.StatusCode == http.StatusOK {
			var tokenResult struct {
				AccessToken  string `json:"access_token"`
				RefreshToken string `json:"refresh_token"`
				IDToken      string `json:"id_token"`
				ExpiresIn    int    `json:"expires_in"`
				TokenType    string `json:"token_type"`
			}
			_ = json.NewDecoder(pollResp.Body).Decode(&tokenResult)
			_ = pollResp.Body.Close()

			creds := &Credentials{
				AccessToken:  tokenResult.AccessToken,
				RefreshToken: tokenResult.RefreshToken,
				IDToken:      tokenResult.IDToken,
				TokenType:    tokenResult.TokenType,
				IssuerURL:    issuerURL,
				ServerURL:    serverURL,
				ExpiresAt:    time.Now().Add(time.Duration(tokenResult.ExpiresIn) * time.Second),
			}
			_ = a.store.Save(creds)
			fmt.Fprintln(stdout, "Successfully authenticated via Device Flow!")
			return 0
		}

		var errResp struct {
			Error string `json:"error"`
		}
		_ = json.NewDecoder(pollResp.Body).Decode(&errResp)
		_ = pollResp.Body.Close()

		if errResp.Error == "authorization_pending" {
			continue
		}
		if errResp.Error == "slow_down" {
			interval += 2 * time.Second
			continue
		}

		fmt.Fprintf(stderr, "Device authorization failed: %s\n", errResp.Error)
		return 1
	}

	fmt.Fprintln(stderr, "Device code expired before authorization was completed.")
	return 1
}

func openBrowser(url string) error {
	var cmd *exec.Cmd
	switch runtime.GOOS {
	case "darwin":
		cmd = exec.Command("open", url)
	case "windows":
		cmd = exec.Command("rundll32", "url.dll,FileProtocolHandler", url)
	default:
		cmd = exec.Command("xdg-open", url)
	}
	return cmd.Start()
}
