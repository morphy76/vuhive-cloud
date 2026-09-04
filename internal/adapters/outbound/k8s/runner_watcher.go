package k8s

import (
	"context"
	"errors"
	"strings"
	"time"

	"github.com/rs/zerolog"
	batchv1 "k8s.io/api/batch/v1"
	"k8s.io/client-go/informers"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/tools/cache"

	"github.com/morphy76/vuhive-cloud/internal/adapters/outbound/s3"
	"github.com/morphy76/vuhive-cloud/internal/application/ports/outbound"
	"github.com/morphy76/vuhive-cloud/internal/domain/model"
)

// RunnerJobWatcher watches Kubernetes Job lifecycle events via client-go Informer
// and reflects status updates (Running, Succeeded, Failed) onto TestRun aggregates.
type RunnerJobWatcher struct {
	client  kubernetes.Interface
	runRepo outbound.TestRunRepository
	cfg     Config
}

// NewRunnerJobWatcher constructs a new RunnerJobWatcher.
func NewRunnerJobWatcher(
	client kubernetes.Interface,
	runRepo outbound.TestRunRepository,
	cfg Config,
) *RunnerJobWatcher {
	return &RunnerJobWatcher{
		client:  client,
		runRepo: runRepo,
		cfg:     cfg,
	}
}

// Start launches the client-go SharedInformer and processes Job lifecycle events until context cancellation.
func (w *RunnerJobWatcher) Start(ctx context.Context) error {
	start := time.Now()
	namespace := strings.TrimSpace(w.cfg.RunnerNamespace)
	if namespace == "" {
		namespace = model.DefaultRunnerNamespace
	}

	log := zerolog.Ctx(ctx).With().
		Str("op", "RunnerJobWatcher.Start").
		Str("namespace", namespace).
		Logger()
	log.Info().Msg("starting runner job informer watcher")

	factory := informers.NewSharedInformerFactoryWithOptions(
		w.client,
		10*time.Minute,
		informers.WithNamespace(namespace),
	)

	jobInformer := factory.Batch().V1().Jobs().Informer()

	_, err := jobInformer.AddEventHandler(cache.ResourceEventHandlerFuncs{
		AddFunc: func(obj interface{}) {
			if job, ok := obj.(*batchv1.Job); ok {
				if err := w.SyncJob(ctx, job); err != nil {
					log.Error().Err(err).Str("job_name", job.Name).Msg("failed handling job add event")
				}
			}
		},
		UpdateFunc: func(_, newObj interface{}) {
			if job, ok := newObj.(*batchv1.Job); ok {
				if err := w.SyncJob(ctx, job); err != nil {
					log.Error().Err(err).Str("job_name", job.Name).Msg("failed handling job update event")
				}
			}
		},
	})
	if err != nil {
		log.Error().Err(err).Dur("duration_ms", time.Since(start)).Msg("failed registering informer event handler")
		return err
	}

	factory.Start(ctx.Done())

	synced := factory.WaitForCacheSync(ctx.Done())
	for typ, ok := range synced {
		if !ok {
			log.Warn().Str("informer_type", typ.String()).Msg("informer cache failed to sync")
		}
	}

	log.Info().Dur("duration_ms", time.Since(start)).Msg("informer caches synced; watcher running")

	<-ctx.Done()
	log.Info().Msg("stopping runner job informer watcher due to context cancellation")
	return nil
}

// SyncJob inspects a batch/v1 Job and transitions the corresponding TestRun aggregate.
func (w *RunnerJobWatcher) SyncJob(ctx context.Context, job *batchv1.Job) error {
	start := time.Now()
	if job == nil {
		return nil
	}

	runID := strings.TrimSpace(job.Labels["vuhive.io/run-id"])
	if runID == "" {
		return nil
	}

	log := zerolog.Ctx(ctx).With().
		Str("op", "RunnerJobWatcher.SyncJob").
		Str("job_name", job.Name).
		Str("run_id", runID).
		Logger()
	log.Debug().Msg("processing job status sync")

	run, err := w.runRepo.FindByID(ctx, runID)
	if err != nil {
		if errors.Is(err, model.ErrNotFound) {
			log.Debug().Msg("test run not found in repository for job; skipping")
			return nil
		}
		log.Error().Err(err).Dur("duration_ms", time.Since(start)).Msg("failed fetching test run")
		return err
	}

	if run.Status().IsTerminal() {
		log.Debug().Str("status", string(run.Status())).Msg("test run is already in terminal state; ignoring")
		return nil
	}

	now := time.Now().UTC()

	// 1. Check if Job succeeded
	if isJobSuccessful(job) {
		finishTime := now
		if job.Status.CompletionTime != nil {
			finishTime = job.Status.CompletionTime.Time.UTC()
		}

		if run.Status() == model.RunStatusQueued {
			startTime := finishTime
			if job.Status.StartTime != nil {
				startTime = job.Status.StartTime.Time.UTC()
			}
			_ = run.Start(job.Name, startTime)
		}

		reportKey, err := s3.KeySummaryReport(run.ID())
		if err != nil {
			log.Error().Err(err).Msg("failed calculating summary report key")
		}
		logsKey, err := s3.KeyExecutionLogs(run.ID())
		if err != nil {
			log.Error().Err(err).Msg("failed calculating execution logs key")
		}

		if err := run.Complete(model.RunMetrics{}, reportKey, logsKey, nil, true, finishTime); err != nil {
			log.Error().Err(err).Dur("duration_ms", time.Since(start)).Msg("failed transitioning run to COMPLETED")
			return err
		}

		if err := w.runRepo.Save(ctx, run); err != nil {
			log.Error().Err(err).Dur("duration_ms", time.Since(start)).Msg("failed saving completed run")
			return err
		}

		log.Info().
			Str("status", string(model.RunStatusCompleted)).
			Dur("duration_ms", time.Since(start)).
			Msg("test run transitioned to COMPLETED")
		return nil
	}

	// 2. Check if Job failed
	if isJobFailed(job) {
		finishTime := now
		if run.Status() == model.RunStatusQueued {
			startTime := finishTime
			if job.Status.StartTime != nil {
				startTime = job.Status.StartTime.Time.UTC()
			}
			_ = run.Start(job.Name, startTime)
		}

		logsKey, err := s3.KeyExecutionLogs(run.ID())
		if err != nil {
			log.Error().Err(err).Msg("failed calculating execution logs key")
		}

		exitCode := 1
		if err := run.Fail(exitCode, logsKey, finishTime); err != nil {
			log.Error().Err(err).Dur("duration_ms", time.Since(start)).Msg("failed transitioning run to FAILED")
			return err
		}

		if err := w.runRepo.Save(ctx, run); err != nil {
			log.Error().Err(err).Dur("duration_ms", time.Since(start)).Msg("failed saving failed run")
			return err
		}

		log.Info().
			Str("status", string(model.RunStatusFailed)).
			Dur("duration_ms", time.Since(start)).
			Msg("test run transitioned to FAILED")
		return nil
	}

	// 3. Check if Job is Active / Running
	if job.Status.Active > 0 || job.Status.StartTime != nil {
		if run.Status() == model.RunStatusQueued {
			startTime := now
			if job.Status.StartTime != nil {
				startTime = job.Status.StartTime.Time.UTC()
			}

			if err := run.Start(job.Name, startTime); err != nil {
				log.Error().Err(err).Dur("duration_ms", time.Since(start)).Msg("failed transitioning run to RUNNING")
				return err
			}

			if err := w.runRepo.Save(ctx, run); err != nil {
				log.Error().Err(err).Dur("duration_ms", time.Since(start)).Msg("failed saving running run")
				return err
			}

			log.Info().
				Str("status", string(model.RunStatusRunning)).
				Dur("duration_ms", time.Since(start)).
				Msg("test run transitioned to RUNNING")
			return nil
		}
	}

	return nil
}
