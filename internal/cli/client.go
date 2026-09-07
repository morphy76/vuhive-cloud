package cli

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"
)

// APIClient interacts with the vuhive-cloud control plane REST API.
type APIClient struct {
	serverURL  string
	store      CredentialStore
	httpClient *http.Client
}

// NewAPIClient creates an APIClient.
func NewAPIClient(serverURL string, store CredentialStore, httpClient *http.Client) *APIClient {
	if serverURL == "" {
		serverURL = "http://localhost:8080"
	}
	serverURL = strings.TrimRight(serverURL, "/")
	if httpClient == nil {
		httpClient = &http.Client{Timeout: 60 * time.Second}
	}
	return &APIClient{
		serverURL:  serverURL,
		store:      store,
		httpClient: httpClient,
	}
}

// ServerURL returns the configured base server URL.
func (c *APIClient) ServerURL() string {
	return c.serverURL
}

// Do executes an HTTP request with Authorization bearer header if credentials exist.
func (c *APIClient) Do(ctx context.Context, req *http.Request) (*http.Response, error) {
	if c.store != nil {
		if creds, err := c.store.Load(); err == nil && creds != nil && creds.AccessToken != "" {
			req.Header.Set("Authorization", "Bearer "+creds.AccessToken)
		}
	}

	resp, err := c.httpClient.Do(req.WithContext(ctx))
	if err != nil {
		return nil, fmt.Errorf("failed connecting to control plane at %s: %w", c.serverURL, err)
	}

	if resp.StatusCode == http.StatusUnauthorized {
		return resp, errors.New("authentication required (401): please run 'vuhive auth login'")
	}
	if resp.StatusCode == http.StatusForbidden {
		return resp, errors.New("forbidden (403): your user does not have permission to execute this operation")
	}

	return resp, nil
}

// Get performs a GET request against the relative path.
func (c *APIClient) Get(ctx context.Context, path string) (*http.Response, error) {
	url := c.serverURL + path
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return nil, err
	}
	return c.Do(ctx, req)
}

// PostJSON performs a POST request with JSON payload.
func (c *APIClient) PostJSON(ctx context.Context, path string, payload interface{}) (*http.Response, error) {
	url := c.serverURL + path
	var body io.Reader
	if payload != nil {
		data, err := json.Marshal(payload)
		if err != nil {
			return nil, fmt.Errorf("failed marshaling request payload: %w", err)
		}
		body = bytes.NewReader(data)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, url, body)
	if err != nil {
		return nil, err
	}
	if payload != nil {
		req.Header.Set("Content-Type", "application/json")
	}

	return c.Do(ctx, req)
}
