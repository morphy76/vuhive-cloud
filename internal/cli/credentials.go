package cli

import (
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/morphy76/vuhive-cloud/internal/domain/model"
)

// DefaultCredentialsPath returns ~/.vuhive/credentials.json.
func DefaultCredentialsPath() string {
	home, err := os.UserHomeDir()
	if err != nil {
		return "credentials.json"
	}
	return filepath.Join(home, ".vuhive", "credentials.json")
}

// Credentials stores cached authentication tokens and connection endpoints.
type Credentials struct {
	AccessToken  string    `json:"access_token"`
	RefreshToken string    `json:"refresh_token,omitempty"`
	IDToken      string    `json:"id_token,omitempty"`
	TokenType    string    `json:"token_type,omitempty"`
	IssuerURL    string    `json:"issuer_url,omitempty"`
	ServerURL    string    `json:"server_url,omitempty"`
	ExpiresAt    time.Time `json:"expires_at"`
}

// Claims parses claims from the access token without full signature validation (for client-side display).
func (c *Credentials) Claims() (*model.Claims, error) {
	if c.AccessToken == "" {
		return nil, errors.New("empty access token")
	}

	parts := strings.Split(c.AccessToken, ".")
	if len(parts) != 3 {
		return nil, errors.New("malformed access token")
	}

	payloadBytes, err := base64.RawURLEncoding.DecodeString(parts[1])
	if err != nil {
		return nil, fmt.Errorf("failed decoding token payload: %w", err)
	}

	var raw struct {
		Sub               string   `json:"sub"`
		PreferredUsername string   `json:"preferred_username"`
		Email             string   `json:"email"`
		Exp               int64    `json:"exp"`
		RealmAccess       struct {
			Roles []string `json:"roles"`
		} `json:"realm_access"`
		Groups []string `json:"groups"`
	}

	if err := json.Unmarshal(payloadBytes, &raw); err != nil {
		return nil, fmt.Errorf("failed unmarshaling token claims: %w", err)
	}

	var exp time.Time
	if raw.Exp > 0 {
		exp = time.Unix(raw.Exp, 0).UTC()
	}

	return model.NewClaims(
		raw.Sub,
		raw.PreferredUsername,
		raw.Email,
		raw.RealmAccess.Roles,
		raw.Groups,
		exp,
	), nil
}

// CredentialStore defines the interface for persisting and loading user credentials.
type CredentialStore interface {
	Load() (*Credentials, error)
	Save(creds *Credentials) error
	Clear() error
}

// FileCredentialStore persists credentials to a secure local file (mode 0600).
type FileCredentialStore struct {
	filePath string
}

// NewFileCredentialStore creates a new FileCredentialStore.
func NewFileCredentialStore(path string) *FileCredentialStore {
	if path == "" {
		path = DefaultCredentialsPath()
	}
	return &FileCredentialStore{filePath: path}
}

// Load reads and parses credentials from file. Returns nil, nil if the file does not exist.
func (s *FileCredentialStore) Load() (*Credentials, error) {
	data, err := os.ReadFile(s.filePath)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, fmt.Errorf("failed reading credentials file: %w", err)
	}

	var creds Credentials
	if err := json.Unmarshal(data, &creds); err != nil {
		return nil, fmt.Errorf("failed parsing credentials file: %w", err)
	}

	return &creds, nil
}

// Save writes credentials to file with 0600 file permissions.
func (s *FileCredentialStore) Save(creds *Credentials) error {
	dir := filepath.Dir(s.filePath)
	if err := os.MkdirAll(dir, 0700); err != nil {
		return fmt.Errorf("failed creating directory %s: %w", dir, err)
	}

	data, err := json.MarshalIndent(creds, "", "  ")
	if err != nil {
		return fmt.Errorf("failed marshaling credentials: %w", err)
	}

	if err := os.WriteFile(s.filePath, data, 0600); err != nil {
		return fmt.Errorf("failed writing credentials file: %w", err)
	}

	return nil
}

// Clear removes the credentials file.
func (s *FileCredentialStore) Clear() error {
	if err := os.Remove(s.filePath); err != nil && !os.IsNotExist(err) {
		return fmt.Errorf("failed removing credentials file: %w", err)
	}
	return nil
}
