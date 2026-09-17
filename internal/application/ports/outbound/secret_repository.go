package outbound

import (
	"context"

	"github.com/morphy76/vuhive-cloud/internal/domain/model"
)

// SecretRepository defines the driven persistence port for Secret entities scoped to test suites.
type SecretRepository interface {
	Save(ctx context.Context, secret *model.Secret) error
	FindByID(ctx context.Context, id string) (*model.Secret, error)
	FindBySuiteIDAndKey(ctx context.Context, suiteID, key string) (*model.Secret, error)
	ListBySuiteID(ctx context.Context, suiteID string) ([]*model.Secret, error)
	Delete(ctx context.Context, id string) error
}
