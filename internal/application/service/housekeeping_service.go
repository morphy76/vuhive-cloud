package service

import (
	"context"
	"fmt"
	"time"

	"github.com/morphy76/vuhive-cloud/internal/application/ports/inbound"
	"github.com/morphy76/vuhive-cloud/internal/application/ports/outbound"
	"github.com/morphy76/vuhive-cloud/internal/domain/model"
	"github.com/rs/zerolog"
)

// HousekeepingService implements inbound.HousekeepingUseCase orchestrating retention policies and cleanup.
type HousekeepingService struct {
	suiteRepo    outbound.TestSuiteRepository
	artifactRepo outbound.ArtifactRepository
	runRepo      outbound.TestRunRepository
	storage      outbound.StoragePort
	globalPolicy model.RetentionPolicy
}

// NewHousekeepingService constructs a new HousekeepingService.
func NewHousekeepingService(
	suiteRepo outbound.TestSuiteRepository,
	artifactRepo outbound.ArtifactRepository,
	runRepo outbound.TestRunRepository,
	storage outbound.StoragePort,
	policy model.RetentionPolicy,
) *HousekeepingService {
	if err := policy.Validate(); err != nil {
		policy = model.DefaultRetentionPolicy()
	}
	return &HousekeepingService{
		suiteRepo:    suiteRepo,
		artifactRepo: artifactRepo,
		runRepo:      runRepo,
		storage:      storage,
		globalPolicy: policy,
	}
}

// GetGlobalPolicy returns the active global retention policy.
func (s *HousekeepingService) GetGlobalPolicy(_ context.Context) model.RetentionPolicy {
	return s.globalPolicy
}

// ConfigureBucketLifecycle configures native S3 lifecycle rules on the underlying storage bucket.
func (s *HousekeepingService) ConfigureBucketLifecycle(ctx context.Context) error {
	start := time.Now()
	log := zerolog.Ctx(ctx).With().Str("op", "HousekeepingService.ConfigureBucketLifecycle").Logger()
	log.Debug().Msg("starting configuring bucket lifecycle rules")

	if s.storage == nil {
		log.Warn().Msg("storage adapter is nil; skipping bucket lifecycle configuration")
		return nil
	}

	rules := []outbound.LifecycleRule{
		{
			ID:             "vuhive-runs-logs-expiration",
			Prefix:         "runs/",
			ExpirationDays: s.globalPolicy.LogsTTLDays,
			Enabled:        true,
		},
		{
			ID:             "vuhive-suites-sources-expiration",
			Prefix:         "suites/",
			ExpirationDays: s.globalPolicy.SourcesTTLDays,
			Enabled:        true,
		},
	}

	if err := s.storage.PutBucketLifecycleConfiguration(ctx, rules); err != nil {
		log.Error().Err(err).Dur("duration_ms", time.Since(start)).Msg("failed configuring bucket lifecycle rules")
		return fmt.Errorf("failed to configure bucket lifecycle rules: %w", err)
	}

	log.Info().Dur("duration_ms", time.Since(start)).Msg("completed configuring bucket lifecycle rules")
	return nil
}

// RunHousekeeping executes a single housekeeping evaluation cycle.
func (s *HousekeepingService) RunHousekeeping(ctx context.Context, cmd inbound.HousekeepingCommand) (*inbound.HousekeepingResult, error) {
	start := time.Now()
	log := zerolog.Ctx(ctx).With().
		Str("op", "HousekeepingService.RunHousekeeping").
		Bool("dry_run", cmd.DryRun).
		Logger()
	log.Debug().Msg("starting housekeeping cycle")

	result := &inbound.HousekeepingResult{
		Errors: make([]string, 0),
	}

	// Determine effective retention policy
	effectivePolicy := s.globalPolicy
	if cmd.SuiteID != nil && *cmd.SuiteID != "" && s.suiteRepo != nil {
		suite, err := s.suiteRepo.FindByID(ctx, *cmd.SuiteID)
		if err == nil && suite != nil && suite.RetentionPolicy() != nil {
			effectivePolicy = effectivePolicy.Merge(*suite.RetentionPolicy())
		}
	}

	now := time.Now().UTC()

	// 1. Purge execution logs past LogsTTLDays
	if s.runRepo != nil {
		cutoffLogs := now.Add(-effectivePolicy.LogsDuration())
		runsWithLogs, err := s.runRepo.ListExpiredRunsForLogs(ctx, cutoffLogs, cmd.SuiteID, 500)
		if err != nil {
			errMsg := fmt.Sprintf("failed listing expired runs for logs: %v", err)
			log.Error().Err(err).Msg(errMsg)
			result.Errors = append(result.Errors, errMsg)
		} else {
			for _, run := range runsWithLogs {
				result.PurgedLogsCount++
				if !cmd.DryRun {
					if s.storage != nil && run.S3LogsKey() != "" {
						if err := s.storage.Delete(ctx, run.S3LogsKey()); err != nil {
							log.Warn().Err(err).Str("key", run.S3LogsKey()).Msg("failed deleting s3 logs object")
						}
					}
					if err := s.runRepo.ClearLogsKey(ctx, run.ID()); err != nil {
						errMsg := fmt.Sprintf("failed clearing logs key for run %s: %v", run.ID(), err)
						log.Error().Err(err).Msg(errMsg)
						result.Errors = append(result.Errors, errMsg)
					}
				}
			}
		}
	}

	// 2. Purge reports and archive runs past ReportsTTLDays
	if s.runRepo != nil {
		cutoffReports := now.Add(-effectivePolicy.ReportsDuration())
		runsWithReports, err := s.runRepo.ListExpiredRunsForReports(ctx, cutoffReports, cmd.SuiteID, 500)
		if err != nil {
			errMsg := fmt.Sprintf("failed listing expired runs for reports: %v", err)
			log.Error().Err(err).Msg(errMsg)
			result.Errors = append(result.Errors, errMsg)
		} else {
			for _, run := range runsWithReports {
				result.PurgedReportsCount++
				result.ArchivedRunsCount++
				if !cmd.DryRun {
					if s.storage != nil && run.S3ReportKey() != "" {
						if err := s.storage.Delete(ctx, run.S3ReportKey()); err != nil {
							log.Warn().Err(err).Str("key", run.S3ReportKey()).Msg("failed deleting s3 report object")
						}
					}
					if err := s.runRepo.ArchiveRun(ctx, run.ID()); err != nil {
						errMsg := fmt.Sprintf("failed archiving run %s: %v", run.ID(), err)
						log.Error().Err(err).Msg(errMsg)
						result.Errors = append(result.Errors, errMsg)
					}
				}
			}
		}
	}

	// 3. Prune historical test run records past RunsTTLDays
	if s.runRepo != nil && !effectivePolicy.ArchiveOnly {
		cutoffRuns := now.Add(-effectivePolicy.RunsDuration())
		runsToPrune, err := s.runRepo.ListExpiredRunsForPrune(ctx, cutoffRuns, cmd.SuiteID, 500)
		if err != nil {
			errMsg := fmt.Sprintf("failed listing expired runs for prune: %v", err)
			log.Error().Err(err).Msg(errMsg)
			result.Errors = append(result.Errors, errMsg)
		} else if len(runsToPrune) > 0 {
			var ids []string
			for _, r := range runsToPrune {
				ids = append(ids, r.ID())
			}
			result.PrunedRunsCount += int64(len(ids))
			if !cmd.DryRun {
				if _, err := s.runRepo.DeleteBatch(ctx, ids); err != nil {
					errMsg := fmt.Sprintf("failed deleting historical runs batch: %v", err)
					log.Error().Err(err).Msg(errMsg)
					result.Errors = append(result.Errors, errMsg)
				}
			}
		}
	}

	// 4. Clean up orphaned compilation artifacts resulting from failed or cancelled jobs
	if s.artifactRepo != nil {
		cutoffOrphans := now.Add(-24 * time.Hour)
		orphans, err := s.artifactRepo.ListOrphanedArtifacts(ctx, cutoffOrphans, 200)
		if err != nil {
			errMsg := fmt.Sprintf("failed listing orphaned artifacts: %v", err)
			log.Error().Err(err).Msg(errMsg)
			result.Errors = append(result.Errors, errMsg)
		} else {
			for _, art := range orphans {
				result.PurgedArtifactsCount++
				if !cmd.DryRun {
					if s.storage != nil {
						if art.BuildLogsS3Key() != "" {
							_ = s.storage.Delete(ctx, art.BuildLogsS3Key())
						}
						if art.S3BinaryKey() != "" {
							_ = s.storage.Delete(ctx, art.S3BinaryKey())
						}
					}
					if err := s.artifactRepo.Delete(ctx, art.ID()); err != nil {
						errMsg := fmt.Sprintf("failed deleting orphaned artifact %s: %v", art.ID(), err)
						log.Error().Err(err).Msg(errMsg)
						result.Errors = append(result.Errors, errMsg)
					}
				}
			}
		}
	}

	// 5. Purge expired binaries past BinariesTTLDays
	if s.artifactRepo != nil {
		cutoffBinaries := now.Add(-effectivePolicy.BinariesDuration())
		expiredArtifacts, err := s.artifactRepo.ListExpiredArtifacts(ctx, cutoffBinaries, cmd.SuiteID, 200)
		if err != nil {
			errMsg := fmt.Sprintf("failed listing expired artifacts: %v", err)
			log.Error().Err(err).Msg(errMsg)
			result.Errors = append(result.Errors, errMsg)
		} else {
			for _, art := range expiredArtifacts {
				result.PurgedBinariesCount++
				if !cmd.DryRun {
					if s.storage != nil && art.S3BinaryKey() != "" {
						_ = s.storage.Delete(ctx, art.S3BinaryKey())
					}
					if err := s.artifactRepo.ClearBinaryKeys(ctx, art.ID()); err != nil {
						errMsg := fmt.Sprintf("failed clearing binary keys for artifact %s: %v", art.ID(), err)
						log.Error().Err(err).Msg(errMsg)
						result.Errors = append(result.Errors, errMsg)
					}
				}
			}
		}
	}

	result.DurationMs = time.Since(start).Milliseconds()
	log.Info().
		Int64("purged_logs", result.PurgedLogsCount).
		Int64("purged_reports", result.PurgedReportsCount).
		Int64("pruned_runs", result.PrunedRunsCount).
		Int64("archived_runs", result.ArchivedRunsCount).
		Int64("purged_artifacts", result.PurgedArtifactsCount).
		Int64("purged_binaries", result.PurgedBinariesCount).
		Dur("duration_ms", time.Since(start)).
		Msg("completed housekeeping cycle")

	return result, nil
}

// StartBackgroundWorker runs the housekeeping cycle periodically until the context is cancelled.
func (s *HousekeepingService) StartBackgroundWorker(ctx context.Context, interval time.Duration) error {
	if interval <= 0 {
		interval = 6 * time.Hour
	}

	log := zerolog.Ctx(ctx).With().
		Str("component", "HousekeepingBackgroundWorker").
		Dur("interval", interval).
		Logger()
	log.Info().Msg("starting housekeeping background worker")

	ticker := time.NewTicker(interval)
	defer ticker.Stop()

	// Run an initial housekeeping sweep shortly after startup (non-blocking)
	go func() {
		select {
		case <-time.After(30 * time.Second):
			if _, err := s.RunHousekeeping(ctx, inbound.HousekeepingCommand{}); err != nil {
				log.Error().Err(err).Msg("initial background housekeeping sweep encountered errors")
			}
		case <-ctx.Done():
			return
		}
	}()

	for {
		select {
		case <-ctx.Done():
			log.Info().Msg("shutting down housekeeping background worker")
			return ctx.Err()
		case <-ticker.C:
			if _, err := s.RunHousekeeping(ctx, inbound.HousekeepingCommand{}); err != nil {
				log.Error().Err(err).Msg("periodic housekeeping run encountered errors")
			}
		}
	}
}

// Static compile-time interface assertion
var _ inbound.HousekeepingUseCase = (*HousekeepingService)(nil)
