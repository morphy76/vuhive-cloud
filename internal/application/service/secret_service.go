package service

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/morphy76/vuhive-cloud/internal/application/ports/inbound"
	"github.com/morphy76/vuhive-cloud/internal/application/ports/outbound"
	"github.com/morphy76/vuhive-cloud/internal/domain/model"
	"github.com/rs/zerolog"
)

// SecretService implements inbound.SecretsUseCase orchestrating suite-scoped secret operations.
type SecretService struct {
	suiteRepo  outbound.TestSuiteRepository
	secretRepo outbound.SecretRepository
	encryptor  outbound.Encryptor
}

var _ inbound.SecretsUseCase = (*SecretService)(nil)

// NewSecretService constructs a new SecretService instance.
func NewSecretService(
	suiteRepo outbound.TestSuiteRepository,
	secretRepo outbound.SecretRepository,
	encryptor outbound.Encryptor,
) *SecretService {
	return &SecretService{
		suiteRepo:  suiteRepo,
		secretRepo: secretRepo,
		encryptor:  encryptor,
	}
}

// CreateSecret encrypts a plaintext value and persists a new suite-scoped secret.
func (s *SecretService) CreateSecret(ctx context.Context, cmd inbound.CreateSecretCommand) (*model.Secret, error) {
	if s == nil || s.encryptor == nil {
		return nil, model.ErrSecretsDisabled
	}

	start := time.Now()
	trimmedSuiteID := strings.TrimSpace(cmd.SuiteID)
	trimmedKey := strings.TrimSpace(cmd.Key)
	trimmedValue := strings.TrimSpace(cmd.Value)

	log := zerolog.Ctx(ctx).With().
		Str("op", "SecretService.CreateSecret").
		Str("suite_id", trimmedSuiteID).
		Str("key", trimmedKey).
		Logger()
	log.Debug().Msg("starting secret creation")

	if trimmedSuiteID == "" {
		return nil, fmt.Errorf("%w: suite ID cannot be empty", model.ErrValidation)
	}
	if err := model.ValidateSecretKey(trimmedKey); err != nil {
		return nil, err
	}
	if trimmedValue == "" {
		return nil, fmt.Errorf("%w: secret value cannot be empty", model.ErrValidation)
	}

	if s.suiteRepo != nil {
		if _, err := s.suiteRepo.FindByID(ctx, trimmedSuiteID); err != nil {
			log.Error().Err(err).Dur("duration_ms", time.Since(start)).Msg("failed finding parent test suite")
			return nil, err
		}
	}

	encrypted, err := s.encryptor.Encrypt([]byte(trimmedValue))
	if err != nil {
		log.Error().Err(err).Dur("duration_ms", time.Since(start)).Msg("failed encrypting secret value")
		return nil, fmt.Errorf("failed to encrypt secret value: %w", err)
	}

	secret, err := model.NewSecret(trimmedSuiteID, trimmedKey, encrypted)
	if err != nil {
		log.Error().Err(err).Dur("duration_ms", time.Since(start)).Msg("failed constructing secret domain model")
		return nil, err
	}

	if s.secretRepo != nil {
		if err := s.secretRepo.Save(ctx, secret); err != nil {
			log.Error().Err(err).Dur("duration_ms", time.Since(start)).Msg("failed saving secret to repository")
			return nil, err
		}
	}

	log.Info().
		Str("secret_id", secret.ID()).
		Dur("duration_ms", time.Since(start)).
		Msg("completed secret creation")
	return secret, nil
}

// ListSecrets retrieves all secrets attached to a given test suite.
// Note: encrypted values are returned; use GetDecryptedSecrets for plaintext.
func (s *SecretService) ListSecrets(ctx context.Context, suiteID string) ([]*model.Secret, error) {
	if s == nil || s.encryptor == nil {
		return nil, model.ErrSecretsDisabled
	}

	start := time.Now()
	trimmedSuiteID := strings.TrimSpace(suiteID)
	if trimmedSuiteID == "" {
		return nil, fmt.Errorf("%w: suite ID cannot be empty", model.ErrValidation)
	}

	log := zerolog.Ctx(ctx).With().
		Str("op", "SecretService.ListSecrets").
		Str("suite_id", trimmedSuiteID).
		Logger()
	log.Debug().Msg("starting secrets listing")

	if s.suiteRepo != nil {
		if _, err := s.suiteRepo.FindByID(ctx, trimmedSuiteID); err != nil {
			log.Error().Err(err).Dur("duration_ms", time.Since(start)).Msg("parent suite not found")
			return nil, err
		}
	}

	if s.secretRepo == nil {
		return []*model.Secret{}, nil
	}

	secrets, err := s.secretRepo.ListBySuiteID(ctx, trimmedSuiteID)
	if err != nil {
		log.Error().Err(err).Dur("duration_ms", time.Since(start)).Msg("failed listing secrets")
		return nil, err
	}

	log.Info().Int("count", len(secrets)).Dur("duration_ms", time.Since(start)).Msg("completed secrets listing")
	return secrets, nil
}

// UpdateSecret re-encrypts a new plaintext value and updates an existing suite-scoped secret.
func (s *SecretService) UpdateSecret(ctx context.Context, cmd inbound.UpdateSecretCommand) (*model.Secret, error) {
	if s == nil || s.encryptor == nil {
		return nil, model.ErrSecretsDisabled
	}

	start := time.Now()
	trimmedSuiteID := strings.TrimSpace(cmd.SuiteID)
	trimmedSecretID := strings.TrimSpace(cmd.SecretID)
	trimmedValue := strings.TrimSpace(cmd.Value)

	if trimmedSuiteID == "" || trimmedSecretID == "" {
		return nil, fmt.Errorf("%w: suite ID and secret ID cannot be empty", model.ErrValidation)
	}

	log := zerolog.Ctx(ctx).With().
		Str("op", "SecretService.UpdateSecret").
		Str("suite_id", trimmedSuiteID).
		Str("secret_id", trimmedSecretID).
		Logger()
	log.Debug().Msg("starting secret update")

	if s.suiteRepo != nil {
		if _, err := s.suiteRepo.FindByID(ctx, trimmedSuiteID); err != nil {
			log.Error().Err(err).Dur("duration_ms", time.Since(start)).Msg("failed finding parent test suite")
			return nil, err
		}
	}

	if s.secretRepo == nil {
		return nil, model.ErrNotFound
	}

	secret, err := s.secretRepo.FindByID(ctx, trimmedSecretID)
	if err != nil {
		log.Error().Err(err).Dur("duration_ms", time.Since(start)).Msg("failed finding secret for update")
		return nil, err
	}

	if secret.SuiteID() != trimmedSuiteID {
		log.Warn().
			Str("actual_suite_id", secret.SuiteID()).
			Dur("duration_ms", time.Since(start)).
			Msg("secret does not belong to specified suite")
		return nil, model.ErrNotFound
	}

	if trimmedValue == "" {
		return nil, fmt.Errorf("%w: secret value cannot be empty", model.ErrValidation)
	}

	encrypted, err := s.encryptor.Encrypt([]byte(trimmedValue))
	if err != nil {
		log.Error().Err(err).Dur("duration_ms", time.Since(start)).Msg("failed encrypting new secret value")
		return nil, fmt.Errorf("failed to encrypt secret value: %w", err)
	}

	if err := secret.UpdateValue(encrypted); err != nil {
		return nil, err
	}

	if err := s.secretRepo.Save(ctx, secret); err != nil {
		log.Error().Err(err).Dur("duration_ms", time.Since(start)).Msg("failed saving updated secret to repository")
		return nil, err
	}

	log.Info().
		Str("secret_id", secret.ID()).
		Dur("duration_ms", time.Since(start)).
		Msg("completed secret update")
	return secret, nil
}

// DeleteSecret removes a suite-scoped secret by ID.
func (s *SecretService) DeleteSecret(ctx context.Context, suiteID, secretID string) error {
	if s == nil || s.encryptor == nil {
		return model.ErrSecretsDisabled
	}

	start := time.Now()
	trimmedSuiteID := strings.TrimSpace(suiteID)
	trimmedSecretID := strings.TrimSpace(secretID)

	if trimmedSuiteID == "" || trimmedSecretID == "" {
		return fmt.Errorf("%w: suite ID and secret ID cannot be empty", model.ErrValidation)
	}

	log := zerolog.Ctx(ctx).With().
		Str("op", "SecretService.DeleteSecret").
		Str("suite_id", trimmedSuiteID).
		Str("secret_id", trimmedSecretID).
		Logger()
	log.Debug().Msg("starting secret deletion")

	if s.secretRepo == nil {
		return model.ErrNotFound
	}

	secret, err := s.secretRepo.FindByID(ctx, trimmedSecretID)
	if err != nil {
		log.Error().Err(err).Dur("duration_ms", time.Since(start)).Msg("failed finding secret for deletion")
		return err
	}

	if secret.SuiteID() != trimmedSuiteID {
		log.Warn().
			Str("actual_suite_id", secret.SuiteID()).
			Dur("duration_ms", time.Since(start)).
			Msg("secret does not belong to specified suite")
		return model.ErrNotFound
	}

	if err := s.secretRepo.Delete(ctx, trimmedSecretID); err != nil {
		log.Error().Err(err).Dur("duration_ms", time.Since(start)).Msg("failed deleting secret from repository")
		return err
	}

	log.Info().Dur("duration_ms", time.Since(start)).Msg("completed secret deletion")
	return nil
}

// GetDecryptedSecrets fetches and decrypts the specified secret keys for a given suite.
// Returns a map of key → plaintext value. Returns ErrMissingSecret if any requested key is not found.
func (s *SecretService) GetDecryptedSecrets(ctx context.Context, suiteID string, keys []string) (map[string]string, error) {
	if s == nil || s.encryptor == nil {
		return nil, model.ErrSecretsDisabled
	}

	start := time.Now()
	trimmedSuiteID := strings.TrimSpace(suiteID)

	log := zerolog.Ctx(ctx).With().
		Str("op", "SecretService.GetDecryptedSecrets").
		Str("suite_id", trimmedSuiteID).
		Int("requested_keys", len(keys)).
		Logger()
	log.Debug().Msg("starting decrypted secrets retrieval")

	if s.secretRepo == nil {
		return nil, fmt.Errorf("%w: secret repository not available", model.ErrValidation)
	}

	allSecrets, err := s.secretRepo.ListBySuiteID(ctx, trimmedSuiteID)
	if err != nil {
		log.Error().Err(err).Dur("duration_ms", time.Since(start)).Msg("failed listing secrets for decryption")
		return nil, err
	}

	secretMap := make(map[string]*model.Secret, len(allSecrets))
	for _, sec := range allSecrets {
		secretMap[sec.Key()] = sec
	}

	// Validate all requested keys exist
	var missing []string
	for _, key := range keys {
		if _, ok := secretMap[key]; !ok {
			missing = append(missing, key)
		}
	}
	if len(missing) > 0 {
		log.Error().
			Strs("missing_keys", missing).
			Dur("duration_ms", time.Since(start)).
			Msg("requested secrets not found")
		return nil, fmt.Errorf("%w: %s", model.ErrMissingSecret, strings.Join(missing, ", "))
	}

	// Decrypt requested keys
	result := make(map[string]string, len(keys))
	for _, key := range keys {
		sec := secretMap[key]
		plaintext, err := s.encryptor.Decrypt(sec.EncryptedValue())
		if err != nil {
			log.Error().Err(err).Str("key", key).Dur("duration_ms", time.Since(start)).Msg("failed decrypting secret")
			return nil, fmt.Errorf("failed to decrypt secret %q: %w", key, err)
		}
		result[key] = string(plaintext)
	}

	log.Info().
		Int("decrypted_count", len(result)).
		Dur("duration_ms", time.Since(start)).
		Msg("completed decrypted secrets retrieval")
	return result, nil
}
