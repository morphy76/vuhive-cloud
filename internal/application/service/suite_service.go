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

// SuiteService implements inbound.SuitesUseCase orchestrating TestSuite operations.
type SuiteService struct {
	suiteRepo outbound.TestSuiteRepository
}

var _ inbound.SuitesUseCase = (*SuiteService)(nil)

// NewSuiteService constructs a new SuiteService instance.
func NewSuiteService(suiteRepo outbound.TestSuiteRepository) *SuiteService {
	return &SuiteService{
		suiteRepo: suiteRepo,
	}
}

// CreateSuite creates and persists a new TestSuite aggregate in DRAFT state.
func (s *SuiteService) CreateSuite(ctx context.Context, name, description string) (*model.TestSuite, error) {
	start := time.Now()
	trimmedName := strings.TrimSpace(name)
	trimmedDesc := strings.TrimSpace(description)

	log := zerolog.Ctx(ctx).With().
		Str("op", "SuiteService.CreateSuite").
		Str("name", trimmedName).
		Logger()
	log.Debug().Msg("starting test suite creation")

	suite, err := model.NewTestSuite(trimmedName, trimmedDesc)
	if err != nil {
		log.Error().Err(err).Dur("duration_ms", time.Since(start)).Msg("failed creating test suite domain model")
		return nil, err
	}

	if err := s.suiteRepo.Save(ctx, suite); err != nil {
		log.Error().Err(err).Dur("duration_ms", time.Since(start)).Msg("failed saving test suite to repository")
		return nil, err
	}

	log.Info().
		Str("suite_id", suite.ID()).
		Dur("duration_ms", time.Since(start)).
		Msg("completed test suite creation")
	return suite, nil
}

// GetSuite retrieves a TestSuite by its unique identifier.
func (s *SuiteService) GetSuite(ctx context.Context, id string) (*model.TestSuite, error) {
	start := time.Now()
	trimmedID := strings.TrimSpace(id)
	if trimmedID == "" {
		return nil, fmt.Errorf("%w: suite ID cannot be empty", model.ErrValidation)
	}

	log := zerolog.Ctx(ctx).With().
		Str("op", "SuiteService.GetSuite").
		Str("suite_id", trimmedID).
		Logger()
	log.Debug().Msg("starting test suite retrieval")

	suite, err := s.suiteRepo.FindByID(ctx, trimmedID)
	if err != nil {
		log.Error().Err(err).Dur("duration_ms", time.Since(start)).Msg("failed retrieving test suite")
		return nil, err
	}

	log.Info().Dur("duration_ms", time.Since(start)).Msg("completed test suite retrieval")
	return suite, nil
}

// ListSuites returns all registered TestSuites ordered by creation timestamp.
func (s *SuiteService) ListSuites(ctx context.Context) ([]*model.TestSuite, error) {
	start := time.Now()
	log := zerolog.Ctx(ctx).With().Str("op", "SuiteService.ListSuites").Logger()
	log.Debug().Msg("starting test suites listing")

	suites, err := s.suiteRepo.List(ctx)
	if err != nil {
		log.Error().Err(err).Dur("duration_ms", time.Since(start)).Msg("failed listing test suites")
		return nil, err
	}

	log.Info().Int("count", len(suites)).Dur("duration_ms", time.Since(start)).Msg("completed test suites listing")
	return suites, nil
}

// UpdateSuite updates mutable fields (name, description, state) of an existing TestSuite.
func (s *SuiteService) UpdateSuite(ctx context.Context, id string, cmd inbound.UpdateSuiteCommand) (*model.TestSuite, error) {
	start := time.Now()
	trimmedID := strings.TrimSpace(id)
	if trimmedID == "" {
		return nil, fmt.Errorf("%w: suite ID cannot be empty", model.ErrValidation)
	}

	log := zerolog.Ctx(ctx).With().
		Str("op", "SuiteService.UpdateSuite").
		Str("suite_id", trimmedID).
		Logger()
	log.Debug().Msg("starting test suite update")

	suite, err := s.suiteRepo.FindByID(ctx, trimmedID)
	if err != nil {
		log.Error().Err(err).Dur("duration_ms", time.Since(start)).Msg("failed finding test suite for update")
		return nil, err
	}

	if cmd.State != nil {
		if !cmd.State.IsValid() {
			log.Error().Str("state", string(*cmd.State)).Dur("duration_ms", time.Since(start)).Msg("invalid target state")
			return nil, model.ErrInvalidStateTransition
		}
		if err := suite.TransitionState(*cmd.State); err != nil {
			log.Error().Err(err).Dur("duration_ms", time.Since(start)).Msg("failed transitioning suite state")
			return nil, err
		}
	}

	if err := suite.UpdateDetails(cmd.Name, cmd.Description); err != nil {
		log.Error().Err(err).Dur("duration_ms", time.Since(start)).Msg("failed updating suite details")
		return nil, err
	}

	if err := s.suiteRepo.Save(ctx, suite); err != nil {
		log.Error().Err(err).Dur("duration_ms", time.Since(start)).Msg("failed saving updated test suite")
		return nil, err
	}

	log.Info().
		Str("suite_id", suite.ID()).
		Str("state", string(suite.State())).
		Dur("duration_ms", time.Since(start)).
		Msg("completed test suite update")
	return suite, nil
}

// DeleteSuite deletes a TestSuite by its identifier.
func (s *SuiteService) DeleteSuite(ctx context.Context, id string) error {
	start := time.Now()
	trimmedID := strings.TrimSpace(id)
	if trimmedID == "" {
		return fmt.Errorf("%w: suite ID cannot be empty", model.ErrValidation)
	}

	log := zerolog.Ctx(ctx).With().
		Str("op", "SuiteService.DeleteSuite").
		Str("suite_id", trimmedID).
		Logger()
	log.Debug().Msg("starting test suite deletion")

	if err := s.suiteRepo.Delete(ctx, trimmedID); err != nil {
		log.Error().Err(err).Dur("duration_ms", time.Since(start)).Msg("failed deleting test suite")
		return err
	}

	log.Info().Dur("duration_ms", time.Since(start)).Msg("completed test suite deletion")
	return nil
}

// ArchiveSuite transitions an active or draft TestSuite to ARCHIVED state.
func (s *SuiteService) ArchiveSuite(ctx context.Context, id string) error {
	start := time.Now()
	trimmedID := strings.TrimSpace(id)
	if trimmedID == "" {
		return fmt.Errorf("%w: suite ID cannot be empty", model.ErrValidation)
	}

	log := zerolog.Ctx(ctx).With().
		Str("op", "SuiteService.ArchiveSuite").
		Str("suite_id", trimmedID).
		Logger()
	log.Debug().Msg("starting test suite archiving")

	suite, err := s.suiteRepo.FindByID(ctx, trimmedID)
	if err != nil {
		log.Error().Err(err).Dur("duration_ms", time.Since(start)).Msg("failed finding test suite to archive")
		return err
	}

	if err := suite.Archive(); err != nil {
		log.Error().Err(err).Dur("duration_ms", time.Since(start)).Msg("failed archiving test suite")
		return err
	}

	if err := s.suiteRepo.Save(ctx, suite); err != nil {
		log.Error().Err(err).Dur("duration_ms", time.Since(start)).Msg("failed saving archived test suite")
		return err
	}

	log.Info().Dur("duration_ms", time.Since(start)).Msg("completed test suite archiving")
	return nil
}
