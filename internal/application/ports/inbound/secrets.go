package inbound

import (
	"context"

	"github.com/morphy76/vuhive-cloud/internal/domain/model"
)

// CreateSecretCommand encapsulates input parameters for creating a new suite-scoped secret.
type CreateSecretCommand struct {
	SuiteID string
	Key     string
	Value   string // Plaintext value; will be encrypted before persistence.
}

// UpdateSecretCommand encapsulates input parameters for updating an existing suite-scoped secret.
type UpdateSecretCommand struct {
	SuiteID  string
	SecretID string
	Value    string // Plaintext value; will be encrypted before persistence.
}

// SecretsUseCase defines driving use cases for managing suite-scoped secrets.
type SecretsUseCase interface {
	CreateSecret(ctx context.Context, cmd CreateSecretCommand) (*model.Secret, error)
	ListSecrets(ctx context.Context, suiteID string) ([]*model.Secret, error)
	UpdateSecret(ctx context.Context, cmd UpdateSecretCommand) (*model.Secret, error)
	DeleteSecret(ctx context.Context, suiteID, secretID string) error
	GetDecryptedSecrets(ctx context.Context, suiteID string, keys []string) (map[string]string, error)
}
