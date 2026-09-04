package k8s

import (
	"context"
	"strings"
	"time"

	"github.com/rs/zerolog"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"

	"github.com/morphy76/vuhive-cloud/internal/application/ports/outbound"
	"github.com/morphy76/vuhive-cloud/internal/domain/model"
)

// RunnerOrchestrator manages the creation and lifecycle of Kubernetes runner Jobs.
type RunnerOrchestrator struct {
	client    kubernetes.Interface
	generator *RunnerJobGenerator
	cfg       Config
}

// NewRunnerOrchestrator constructs a new RunnerOrchestrator with the given kubernetes client and configuration.
func NewRunnerOrchestrator(client kubernetes.Interface, cfg Config) *RunnerOrchestrator {
	return &RunnerOrchestrator{
		client:    client,
		generator: NewRunnerJobGenerator(cfg),
		cfg:       cfg,
	}
}

// DispatchJob manifests and submits a runner batch/v1 Job into Kubernetes.
func (o *RunnerOrchestrator) DispatchJob(
	ctx context.Context,
	run *model.TestRun,
	profile *model.RunnerProfile,
	opts outbound.RunnerJobOptions,
) (string, error) {
	start := time.Now()
	runID := ""
	suiteID := ""
	if run != nil {
		runID = run.ID()
		suiteID = run.SuiteID()
	}

	log := zerolog.Ctx(ctx).With().
		Str("op", "RunnerOrchestrator.DispatchJob").
		Str("run_id", runID).
		Str("suite_id", suiteID).
		Logger()
	log.Debug().Msg("starting runner job dispatch")

	jobManifest, err := o.generator.GenerateJob(run, profile, opts)
	if err != nil {
		log.Error().Err(err).Dur("duration_ms", time.Since(start)).Msg("failed generating runner job manifest")
		return "", err
	}

	namespace := jobManifest.Namespace
	createdJob, err := o.client.BatchV1().Jobs(namespace).Create(ctx, jobManifest, metav1.CreateOptions{})
	if err != nil {
		mapped := MapK8sError(err)
		log.Error().Err(mapped).Dur("duration_ms", time.Since(start)).Msg("failed creating runner job in kubernetes")
		return "", mapped
	}

	log.Info().
		Str("job_name", createdJob.Name).
		Str("namespace", namespace).
		Dur("duration_ms", time.Since(start)).
		Msg("completed runner job dispatch")

	return createdJob.Name, nil
}

// AbortJob terminates and deletes an in-flight runner Job on Kubernetes.
func (o *RunnerOrchestrator) AbortJob(ctx context.Context, k8sJobName, namespace string) error {
	start := time.Now()
	trimmedJobName := strings.TrimSpace(k8sJobName)
	trimmedNamespace := strings.TrimSpace(namespace)
	if trimmedNamespace == "" {
		trimmedNamespace = strings.TrimSpace(o.cfg.RunnerNamespace)
	}
	if trimmedNamespace == "" {
		trimmedNamespace = model.DefaultRunnerNamespace
	}

	log := zerolog.Ctx(ctx).With().
		Str("op", "RunnerOrchestrator.AbortJob").
		Str("job_name", trimmedJobName).
		Str("namespace", trimmedNamespace).
		Logger()
	log.Debug().Msg("starting runner job abort")

	propagation := metav1.DeletePropagationBackground
	err := o.client.BatchV1().Jobs(trimmedNamespace).Delete(ctx, trimmedJobName, metav1.DeleteOptions{
		PropagationPolicy: &propagation,
	})
	if err != nil {
		mapped := MapK8sError(err)
		log.Error().Err(mapped).Dur("duration_ms", time.Since(start)).Msg("failed aborting runner job in kubernetes")
		return mapped
	}

	log.Info().Dur("duration_ms", time.Since(start)).Msg("completed runner job abort")
	return nil
}

// Compile-time static interface verification
var _ outbound.RunnerOrchestratorPort = (*RunnerOrchestrator)(nil)
