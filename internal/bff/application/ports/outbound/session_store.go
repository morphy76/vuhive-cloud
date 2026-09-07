package outbound

import (
	"context"
	"time"

	"github.com/morphy76/vuhive-cloud/internal/bff/domain/model"
)

// SessionStore defines the driven outbound port for persisting, querying, and managing client sessions.
type SessionStore interface {
	Create(ctx context.Context, session *model.ClientSession) error
	Get(ctx context.Context, id model.SessionID) (*model.ClientSession, error)
	Update(ctx context.Context, session *model.ClientSession) error
	Delete(ctx context.Context, id model.SessionID) error
	DeleteByKeycloakSID(ctx context.Context, sid string) error
	DeleteByUserID(ctx context.Context, userID string) error
	DeleteExpired(ctx context.Context, before time.Time) (int64, error)
}
