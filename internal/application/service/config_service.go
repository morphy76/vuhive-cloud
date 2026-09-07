package service

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/morphy76/vuhive-cloud/internal/application/ports/inbound"
	"github.com/morphy76/vuhive-cloud/internal/application/ports/outbound"
	"github.com/morphy76/vuhive-cloud/internal/domain/model"
	"github.com/rs/zerolog"
)

// ConfigService implements inbound.ConfigsUseCase orchestrating TestSuite Configuration operations.
type ConfigService struct {
	suiteRepo  outbound.TestSuiteRepository
	configRepo outbound.ConfigurationRepository
	storage    outbound.StoragePort
}

var _ inbound.ConfigsUseCase = (*ConfigService)(nil)

// NewConfigService constructs a new ConfigService instance.
func NewConfigService(
	suiteRepo outbound.TestSuiteRepository,
	configRepo outbound.ConfigurationRepository,
	storage outbound.StoragePort,
) *ConfigService {
	return &ConfigService{
		suiteRepo:  suiteRepo,
		configRepo: configRepo,
		storage:    storage,
	}
}

// CreateConfig uploads a scenario configuration YAML file to object storage and registers the entity.
func (s *ConfigService) CreateConfig(ctx context.Context, cmd inbound.CreateConfigCommand) (*model.Configuration, error) {
	start := time.Now()
	trimmedSuiteID := strings.TrimSpace(cmd.SuiteID)
	trimmedName := strings.TrimSpace(cmd.Name)
	trimmedYAML := strings.TrimSpace(cmd.ContentYAML)

	log := zerolog.Ctx(ctx).With().
		Str("op", "ConfigService.CreateConfig").
		Str("suite_id", trimmedSuiteID).
		Str("name", trimmedName).
		Logger()
	log.Debug().Msg("starting test scenario configuration creation")

	if trimmedSuiteID == "" {
		return nil, fmt.Errorf("%w: suite ID cannot be empty", model.ErrValidation)
	}
	if s.suiteRepo != nil {
		if _, err := s.suiteRepo.FindByID(ctx, trimmedSuiteID); err != nil {
			log.Error().Err(err).Dur("duration_ms", time.Since(start)).Msg("failed finding parent test suite")
			return nil, err
		}
	}

	if trimmedName == "" {
		return nil, model.ErrEmptyName
	}
	if trimmedYAML == "" {
		return nil, fmt.Errorf("%w: configuration YAML cannot be empty", model.ErrValidation)
	}

	configID := uuid.NewString()
	s3Key := fmt.Sprintf("suites/%s/configs/%s.yaml", trimmedSuiteID, configID)

	if s.storage != nil {
		reader := strings.NewReader(cmd.ContentYAML)
		if err := s.storage.Upload(ctx, s3Key, reader, int64(len(cmd.ContentYAML)), "application/x-yaml"); err != nil {
			log.Error().Err(err).Dur("duration_ms", time.Since(start)).Msg("failed uploading configuration to storage")
			return nil, fmt.Errorf("failed to upload configuration to storage: %w", err)
		}
	}

	config, err := model.NewConfigurationWithID(
		configID,
		trimmedSuiteID,
		trimmedName,
		cmd.ContentYAML,
		s3Key,
		cmd.IsDefault,
		time.Now().UTC(),
	)
	if err != nil {
		log.Error().Err(err).Dur("duration_ms", time.Since(start)).Msg("failed constructing configuration domain model")
		return nil, err
	}

	if s.configRepo != nil {
		if err := s.configRepo.Save(ctx, config); err != nil {
			log.Error().Err(err).Dur("duration_ms", time.Since(start)).Msg("failed saving configuration to repository")
			return nil, err
		}
	}

	log.Info().
		Str("config_id", config.ID()).
		Dur("duration_ms", time.Since(start)).
		Msg("completed test scenario configuration creation")
	return config, nil
}

// GetConfig retrieves an attached configuration by suite ID and config ID.
func (s *ConfigService) GetConfig(ctx context.Context, suiteID, configID string) (*model.Configuration, error) {
	start := time.Now()
	trimmedSuiteID := strings.TrimSpace(suiteID)
	trimmedConfigID := strings.TrimSpace(configID)

	if trimmedSuiteID == "" || trimmedConfigID == "" {
		return nil, fmt.Errorf("%w: suite ID and config ID cannot be empty", model.ErrValidation)
	}

	log := zerolog.Ctx(ctx).With().
		Str("op", "ConfigService.GetConfig").
		Str("suite_id", trimmedSuiteID).
		Str("config_id", trimmedConfigID).
		Logger()
	log.Debug().Msg("starting test scenario configuration retrieval")

	if s.configRepo == nil {
		return nil, model.ErrNotFound
	}

	config, err := s.configRepo.FindByID(ctx, trimmedConfigID)
	if err != nil {
		log.Error().Err(err).Dur("duration_ms", time.Since(start)).Msg("failed retrieving configuration")
		return nil, err
	}

	if config.SuiteID() != trimmedSuiteID {
		log.Warn().
			Str("actual_suite_id", config.SuiteID()).
			Dur("duration_ms", time.Since(start)).
			Msg("configuration does not belong to specified suite")
		return nil, model.ErrNotFound
	}

	log.Info().Dur("duration_ms", time.Since(start)).Msg("completed test scenario configuration retrieval")
	return config, nil
}

// ListConfigs retrieves all configurations attached to a given test suite.
func (s *ConfigService) ListConfigs(ctx context.Context, suiteID string) ([]*model.Configuration, error) {
	start := time.Now()
	trimmedSuiteID := strings.TrimSpace(suiteID)
	if trimmedSuiteID == "" {
		return nil, fmt.Errorf("%w: suite ID cannot be empty", model.ErrValidation)
	}

	log := zerolog.Ctx(ctx).With().
		Str("op", "ConfigService.ListConfigs").
		Str("suite_id", trimmedSuiteID).
		Logger()
	log.Debug().Msg("starting test scenario configurations listing")

	if s.suiteRepo != nil {
		if _, err := s.suiteRepo.FindByID(ctx, trimmedSuiteID); err != nil {
			log.Error().Err(err).Dur("duration_ms", time.Since(start)).Msg("parent suite not found")
			return nil, err
		}
	}

	if s.configRepo == nil {
		return []*model.Configuration{}, nil
	}

	configs, err := s.configRepo.ListBySuiteID(ctx, trimmedSuiteID)
	if err != nil {
		log.Error().Err(err).Dur("duration_ms", time.Since(start)).Msg("failed listing configurations")
		return nil, err
	}

	log.Info().Int("count", len(configs)).Dur("duration_ms", time.Since(start)).Msg("completed test scenario configurations listing")
	return configs, nil
}

// DeleteConfig deletes an attached configuration from storage and repository.
func (s *ConfigService) DeleteConfig(ctx context.Context, suiteID, configID string) error {
	start := time.Now()
	trimmedSuiteID := strings.TrimSpace(suiteID)
	trimmedConfigID := strings.TrimSpace(configID)

	if trimmedSuiteID == "" || trimmedConfigID == "" {
		return fmt.Errorf("%w: suite ID and config ID cannot be empty", model.ErrValidation)
	}

	log := zerolog.Ctx(ctx).With().
		Str("op", "ConfigService.DeleteConfig").
		Str("suite_id", trimmedSuiteID).
		Str("config_id", trimmedConfigID).
		Logger()
	log.Debug().Msg("starting test scenario configuration deletion")

	if s.configRepo == nil {
		return model.ErrNotFound
	}

	config, err := s.configRepo.FindByID(ctx, trimmedConfigID)
	if err != nil {
		log.Error().Err(err).Dur("duration_ms", time.Since(start)).Msg("failed finding configuration for deletion")
		return err
	}

	if config.SuiteID() != trimmedSuiteID {
		log.Warn().
			Str("actual_suite_id", config.SuiteID()).
			Dur("duration_ms", time.Since(start)).
			Msg("configuration does not belong to specified suite")
		return model.ErrNotFound
	}

	if s.storage != nil && config.S3ConfigKey() != "" {
		if err := s.storage.Delete(ctx, config.S3ConfigKey()); err != nil {
			log.Warn().Err(err).Str("key", config.S3ConfigKey()).Msg("failed deleting configuration file from storage")
		}
	}

	if err := s.configRepo.Delete(ctx, trimmedConfigID); err != nil {
		log.Error().Err(err).Dur("duration_ms", time.Since(start)).Msg("failed deleting configuration record from repository")
		return err
	}

	log.Info().Dur("duration_ms", time.Since(start)).Msg("completed test scenario configuration deletion")
	return nil
}
