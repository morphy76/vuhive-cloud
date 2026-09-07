package outbound

import (
	"context"

	"github.com/morphy76/vuhive-cloud/internal/domain/model"
)

// TokenVerifierPort defines the driven interface for verifying OIDC JWT access tokens.
type TokenVerifierPort interface {
	VerifyToken(ctx context.Context, tokenString string) (*model.Claims, error)
}
