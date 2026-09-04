package k8s

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/rs/zerolog"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"

	"github.com/morphy76/vuhive-cloud/internal/application/ports/outbound"
	"github.com/morphy76/vuhive-cloud/internal/domain/model"
)

// ScheduleOrchestrator manages native Kubernetes CronJobs.
type ScheduleOrchestrator struct {
	client    kubernetes.Interface
	generator *CronJobGenerator
	cfg       Config
}

// NewScheduleOrchestrator constructs a new ScheduleOrchestrator.
func NewScheduleOrchestrator(client kubernetes.Interface, cfg Config) *ScheduleOrchestrator {
	return &ScheduleOrchestrator{
		client:    client,
		generator: NewCronJobGenerator(cfg),
		cfg:       cfg,
	}
}

// CreateCronJob manifests and submits a native batch/v1 CronJob into Kubernetes.
func (o *ScheduleOrchestrator) CreateCronJob(
	ctx context.Context,
	schedule *model.Schedule,
	profile *model.RunnerProfile,
	opts outbound.RunnerJobOptions,
) (string, error) {
	start := time.Now()
	schedID := ""
	suiteID := ""
	if schedule != nil {
		schedID = schedule.ID()
		suiteID = schedule.SuiteID()
	}

	log := zerolog.Ctx(ctx).With().
		Str("op", "ScheduleOrchestrator.CreateCronJob").
		Str("schedule_id", schedID).
		Str("suite_id", suiteID).
		Logger()
	log.Debug().Msg("starting native cronjob creation")

	cronJobManifest, err := o.generator.GenerateCronJob(schedule, profile, opts)
	if err != nil {
		log.Error().Err(err).Dur("duration_ms", time.Since(start)).Msg("failed generating cronjob manifest")
		return "", err
	}

	namespace := cronJobManifest.Namespace
	createdCronJob, err := o.client.BatchV1().CronJobs(namespace).Create(ctx, cronJobManifest, metav1.CreateOptions{})
	if err != nil {
		mapped := MapK8sError(err)
		log.Error().Err(mapped).Dur("duration_ms", time.Since(start)).Msg("failed creating cronjob in kubernetes")
		return "", mapped
	}

	log.Info().
		Str("cronjob_name", createdCronJob.Name).
		Str("namespace", namespace).
		Dur("duration_ms", time.Since(start)).
		Msg("completed native cronjob creation")

	return createdCronJob.Name, nil
}

// UpdateCronJob updates the schedule expression and settings of an existing native CronJob.
func (o *ScheduleOrchestrator) UpdateCronJob(ctx context.Context, schedule *model.Schedule) error {
	start := time.Now()
	if schedule == nil {
		return fmt.Errorf("%w: schedule cannot be nil", model.ErrValidation)
	}

	namespace := strings.TrimSpace(o.cfg.RunnerNamespace)
	if namespace == "" {
		namespace = model.DefaultRunnerNamespace
	}

	cronJobName := schedule.K8sCronJobName()

	log := zerolog.Ctx(ctx).With().
		Str("op", "ScheduleOrchestrator.UpdateCronJob").
		Str("schedule_id", schedule.ID()).
		Str("cronjob_name", cronJobName).
		Str("namespace", namespace).
		Logger()
	log.Debug().Msg("starting native cronjob update")

	cronJob, err := o.client.BatchV1().CronJobs(namespace).Get(ctx, cronJobName, metav1.GetOptions{})
	if err != nil {
		mapped := MapK8sError(err)
		log.Error().Err(mapped).Dur("duration_ms", time.Since(start)).Msg("failed fetching cronjob for update")
		return mapped
	}

	cronJob.Spec.Schedule = schedule.CronExpression()
	suspend := !schedule.IsActive()
	cronJob.Spec.Suspend = &suspend

	_, err = o.client.BatchV1().CronJobs(namespace).Update(ctx, cronJob, metav1.UpdateOptions{})
	if err != nil {
		mapped := MapK8sError(err)
		log.Error().Err(mapped).Dur("duration_ms", time.Since(start)).Msg("failed updating cronjob in kubernetes")
		return mapped
	}

	log.Info().Dur("duration_ms", time.Since(start)).Msg("completed native cronjob update")
	return nil
}

// DeleteCronJob removes a native CronJob from Kubernetes.
func (o *ScheduleOrchestrator) DeleteCronJob(ctx context.Context, k8sCronJobName, namespace string) error {
	start := time.Now()
	trimmedName := strings.TrimSpace(k8sCronJobName)
	trimmedNamespace := strings.TrimSpace(namespace)
	if trimmedNamespace == "" {
		trimmedNamespace = strings.TrimSpace(o.cfg.RunnerNamespace)
	}
	if trimmedNamespace == "" {
		trimmedNamespace = model.DefaultRunnerNamespace
	}

	log := zerolog.Ctx(ctx).With().
		Str("op", "ScheduleOrchestrator.DeleteCronJob").
		Str("cronjob_name", trimmedName).
		Str("namespace", trimmedNamespace).
		Logger()
	log.Debug().Msg("starting native cronjob deletion")

	propagation := metav1.DeletePropagationBackground
	err := o.client.BatchV1().CronJobs(trimmedNamespace).Delete(ctx, trimmedName, metav1.DeleteOptions{
		PropagationPolicy: &propagation,
	})
	if err != nil {
		mapped := MapK8sError(err)
		if mapped == model.ErrNotFound {
			log.Debug().Dur("duration_ms", time.Since(start)).Msg("cronjob already not found in kubernetes during delete")
			return nil
		}
		log.Error().Err(mapped).Dur("duration_ms", time.Since(start)).Msg("failed deleting cronjob from kubernetes")
		return mapped
	}

	log.Info().Dur("duration_ms", time.Since(start)).Msg("completed native cronjob deletion")
	return nil
}

// Compile-time static interface verification
var _ outbound.ScheduleOrchestratorPort = (*ScheduleOrchestrator)(nil)
