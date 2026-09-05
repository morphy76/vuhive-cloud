package model

import (
	"fmt"
	"time"
)

// SessionID represents the unique identifier for a client session.
type SessionID string

// ClientSession models an active client session interacting with the BFF.
type ClientSession struct {
	ID        SessionID
	UserID    string
	CreatedAt time.Time
	ExpiresAt time.Time
	Metadata  map[string]string
}

// NewClientSession constructs and validates a new ClientSession aggregate.
func NewClientSession(id, userID string, ttl time.Duration) (*ClientSession, error) {
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
	return &ClientSession{
		ID:        SessionID(id),
		UserID:    userID,
		CreatedAt: now,
		ExpiresAt: now.Add(ttl),
		Metadata:  make(map[string]string),
	}, nil
}

// IsExpired checks if the session has passed its expiration time.
func (s *ClientSession) IsExpired() bool {
	return time.Now().After(s.ExpiresAt)
}
