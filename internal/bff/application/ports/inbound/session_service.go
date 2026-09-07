package inbound

import (
	"context"
	"time"

	"github.com/morphy76/vuhive-cloud/internal/bff/domain/model"
)

// RotateTokensCommand encapsulates input parameters for rotating session tokens.
type RotateTokensCommand struct {
	SessionID    model.SessionID
	AccessToken  string
	RefreshToken string
	IDToken      string
	TTL          time.Duration
}

// SessionService defines the driving inbound port for session lifecycle operations.
type SessionService interface {
	CreateSession(ctx context.Context, cmd CreateSessionCommand) (*model.ClientSession, error)
	GetSession(ctx context.Context, id model.SessionID) (*model.ClientSession, error)
	RotateSessionTokens(ctx context.Context, cmd RotateTokensCommand) (*model.ClientSession, error)
	RevokeSession(ctx context.Context, id model.SessionID) error
	RevokeByKeycloakSID(ctx context.Context, sid string) error
}
