package model

import (
	"fmt"
	"time"
)

// SessionID represents the unique identifier for a client session.
type SessionID string

// ClientSession models an active client session interacting with the BFF.
type ClientSession struct {
	ID           SessionID
	UserID       string
	KeycloakSID  string
	AccessToken  string
	RefreshToken string
	IDToken      string
	Roles        []string
	CreatedAt    time.Time
	UpdatedAt    time.Time
	ExpiresAt    time.Time
	Metadata     map[string]string
}

// SessionOption configures optional fields on a ClientSession during creation.
type SessionOption func(*ClientSession)

// WithKeycloakSID sets the Keycloak session ID (sid claim) on the session.
func WithKeycloakSID(sid string) SessionOption {
	return func(s *ClientSession) {
		s.KeycloakSID = sid
	}
}

// WithTokens sets the OAuth2 / OIDC tokens on the session.
func WithTokens(accessToken, refreshToken, idToken string) SessionOption {
	return func(s *ClientSession) {
		s.AccessToken = accessToken
		s.RefreshToken = refreshToken
		s.IDToken = idToken
	}
}

// WithRoles sets the user roles associated with the session.
func WithRoles(roles []string) SessionOption {
	return func(s *ClientSession) {
		if roles != nil {
			s.Roles = make([]string, len(roles))
			copy(s.Roles, roles)
		}
	}
}

// WithMetadata sets contextual metadata attributes on the session.
func WithMetadata(metadata map[string]string) SessionOption {
	return func(s *ClientSession) {
		if metadata != nil {
			s.Metadata = make(map[string]string, len(metadata))
			for k, v := range metadata {
				s.Metadata[k] = v
			}
		}
	}
}

// NewClientSession constructs and validates a new ClientSession aggregate.
func NewClientSession(id, userID string, ttl time.Duration, opts ...SessionOption) (*ClientSession, error) {
	if id == "" {
		return nil, fmt.Errorf("%w: session ID cannot be empty", ErrInvalidParameter)
	}
	if userID == "" {
		return nil, fmt.Errorf("%w: user ID cannot be empty", ErrInvalidParameter)
	}
	if ttl <= 0 {
		return nil, fmt.Errorf("%w: ttl must be greater than zero", ErrInvalidParameter)
	}

	now := time.Now()
	session := &ClientSession{
		ID:        SessionID(id),
		UserID:    userID,
		Roles:     make([]string, 0),
		CreatedAt: now,
		UpdatedAt: now,
		ExpiresAt: now.Add(ttl),
		Metadata:  make(map[string]string),
	}

	for _, opt := range opts {
		opt(session)
	}

	return session, nil
}

// IsExpired checks if the session has passed its expiration time.
func (s *ClientSession) IsExpired() bool {
	return time.Now().After(s.ExpiresAt)
}

// RotateTokens rotates the OAuth2 access and refresh tokens and extends the session TTL.
func (s *ClientSession) RotateTokens(accessToken, refreshToken, idToken string, ttl time.Duration) error {
	if s.IsExpired() {
		return fmt.Errorf("%w: cannot rotate tokens on expired session", ErrSessionExpired)
	}
	if accessToken == "" {
		return fmt.Errorf("%w: access token cannot be empty", ErrInvalidParameter)
	}
	if refreshToken == "" {
		return fmt.Errorf("%w: refresh token cannot be empty", ErrInvalidParameter)
	}
	if ttl <= 0 {
		return fmt.Errorf("%w: ttl must be greater than zero", ErrInvalidParameter)
	}

	now := time.Now()
	s.AccessToken = accessToken
	s.RefreshToken = refreshToken
	if idToken != "" {
		s.IDToken = idToken
	}
	s.UpdatedAt = now
	s.ExpiresAt = now.Add(ttl)
	return nil
}

// Touch extends the session expiration by the specified TTL without rotating tokens.
func (s *ClientSession) Touch(ttl time.Duration) error {
	if s.IsExpired() {
		return fmt.Errorf("%w: cannot touch expired session", ErrSessionExpired)
	}
	if ttl <= 0 {
		return fmt.Errorf("%w: ttl must be greater than zero", ErrInvalidParameter)
	}

	now := time.Now()
	s.UpdatedAt = now
	s.ExpiresAt = now.Add(ttl)
	return nil
}

// Revoke terminates the session by expiring it immediately.
func (s *ClientSession) Revoke() {
	now := time.Now()
	s.UpdatedAt = now
	s.ExpiresAt = now.Add(-1 * time.Second)
}

// Clone returns a deep copy of the ClientSession aggregate.
func (s *ClientSession) Clone() *ClientSession {
	if s == nil {
		return nil
	}
	cpy := *s
	if s.Roles != nil {
		cpy.Roles = make([]string, len(s.Roles))
		copy(cpy.Roles, s.Roles)
	}
	if s.Metadata != nil {
		cpy.Metadata = make(map[string]string, len(s.Metadata))
		for k, v := range s.Metadata {
			cpy.Metadata[k] = v
		}
	}
	return &cpy
}
