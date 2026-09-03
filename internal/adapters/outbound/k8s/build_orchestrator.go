package k8s

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"regexp"
	"strings"
	"time"

	"github.com/rs/zerolog"
	batchv1 "k8s.io/api/batch/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"

	"github.com/morphy76/vuhive-cloud/internal/application/ports/outbound"
	"github.com/morphy76/vuhive-cloud/internal/domain/model"
)

var sha256Regex = regexp.MustCompile(`VUHIVE_ARTIFACT_SHA256=([a-fA-F0-9]{64})`)

// BuildOrchestrator manages the dispatching and lifecycle tracking of Kubernetes compilation Jobs.
type BuildOrchestrator struct {
	client    kubernetes.Interface
	generator *BuildJobGenerator
	cfg       Config
}

// NewBuildOrchestrator constructs a new BuildOrchestrator.
func NewBuildOrchestrator(client kubernetes.Interface, cfg Config) *BuildOrchestrator {
	return &BuildOrchestrator{
		client:    client,
		generator: NewBuildJobGenerator(cfg),
		cfg:       cfg,
	}
}

// DispatchBuildJob manifests and submits a compilation Job into Kubernetes.
func (o *BuildOrchestrator) DispatchBuildJob(ctx context.Context, opts outbound.BuildJobOptions) (string, error) {
	start := time.Now()
	log := zerolog.Ctx(ctx).With().
		Str("op", "BuildOrchestrator.DispatchBuildJob").
		Str("suite_id", opts.SuiteID).
		Str("artifact_id", opts.ArtifactID).
		Str("platform", string(opts.Platform)).
		Logger()
	log.Debug().Msg("starting build job dispatch")

	jobManifest, err := o.generator.GenerateBuildJob(opts)
	if err != nil {
		log.Error().Err(err).Dur("duration_ms", time.Since(start)).Msg("failed generating build job manifest")
		return "", err
	}

	createdJob, err := o.client.BatchV1().Jobs(o.cfg.Namespace).Create(ctx, jobManifest, metav1.CreateOptions{})
	if err != nil {
		mapped := MapK8sError(err)
		log.Error().Err(mapped).Dur("duration_ms", time.Since(start)).Msg("failed creating build job in kubernetes")
		return "", mapped
	}

	log.Info().
		Str("job_name", createdJob.Name).
		Str("namespace", o.cfg.Namespace).
		Dur("duration_ms", time.Since(start)).
		Msg("completed build job dispatch")

	return createdJob.Name, nil
}

// StreamJobLogs returns an io.ReadCloser streaming logs from the build Job's pod.
func (o *BuildOrchestrator) StreamJobLogs(ctx context.Context, jobName string) (io.ReadCloser, error) {
	start := time.Now()
	log := zerolog.Ctx(ctx).With().
		Str("op", "BuildOrchestrator.StreamJobLogs").
		Str("job_name", jobName).
		Str("namespace", o.cfg.Namespace).
		Logger()
	log.Debug().Msg("starting log stream retrieval")

	podList, err := o.client.CoreV1().Pods(o.cfg.Namespace).List(ctx, metav1.ListOptions{
		LabelSelector: fmt.Sprintf("job-name=%s", jobName),
	})
	if err != nil {
		mapped := MapK8sError(err)
		log.Error().Err(mapped).Dur("duration_ms", time.Since(start)).Msg("failed listing pods for job")
		return nil, mapped
	}

	if len(podList.Items) == 0 {
		err := fmt.Errorf("%w: no pod found for job %s", model.ErrNotFound, jobName)
		log.Error().Err(err).Dur("duration_ms", time.Since(start)).Msg("job pod not found")
		return nil, err
	}

	podName := podList.Items[0].Name
	req := o.client.CoreV1().Pods(o.cfg.Namespace).GetLogs(podName, &corev1.PodLogOptions{})
	stream, err := req.Stream(ctx)
	if err != nil {
		mapped := MapK8sError(err)
		log.Error().Err(mapped).Dur("duration_ms", time.Since(start)).Msg("failed opening pod log stream")
		return nil, mapped
	}

	log.Info().
		Str("pod_name", podName).
		Dur("duration_ms", time.Since(start)).
		Msg("completed log stream retrieval")

	return stream, nil
}

// WaitForJob waits for the build job to reach a terminal state (Succeeded or Failed).
func (o *BuildOrchestrator) WaitForJob(ctx context.Context, jobName string) (*outbound.BuildJobExecution, error) {
	start := time.Now()
	log := zerolog.Ctx(ctx).With().
		Str("op", "BuildOrchestrator.WaitForJob").
		Str("job_name", jobName).
		Str("namespace", o.cfg.Namespace).
		Logger()
	log.Debug().Msg("waiting for build job completion")

	ticker := time.NewTicker(o.cfg.PollInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			mapped := MapK8sError(ctx.Err())
			log.Error().Err(mapped).Dur("duration_ms", time.Since(start)).Msg("context expired while waiting for job")
			return nil, mapped
		case <-ticker.C:
			job, err := o.client.BatchV1().Jobs(o.cfg.Namespace).Get(ctx, jobName, metav1.GetOptions{})
			if err != nil {
				mapped := MapK8sError(err)
				log.Error().Err(mapped).Dur("duration_ms", time.Since(start)).Msg("failed fetching job status")
				return nil, mapped
			}

			if isJobSuccessful(job) {
				logsReader, logContent := o.fetchLogsSafely(ctx, jobName)
				checksum := ExtractSHA256Checksum(logContent)

				log.Info().
					Str("job_name", jobName).
					Str("checksum", checksum).
					Dur("duration_ms", time.Since(start)).
					Msg("build job succeeded")

				return &outbound.BuildJobExecution{
					JobName:        jobName,
					ExitCode:       0,
					SHA256Checksum: checksum,
					Logs:           logsReader,
				}, nil
			}

			if isJobFailed(job) {
				logsReader, _ := o.fetchLogsSafely(ctx, jobName)
				log.Error().
					Str("job_name", jobName).
					Dur("duration_ms", time.Since(start)).
					Msg("build job failed")

				return &outbound.BuildJobExecution{
					JobName:  jobName,
					ExitCode: 1,
					Logs:     logsReader,
				}, model.ErrBuildFailed
			}
		}
	}
}

// DeleteJob removes a build Job and its child pods from the cluster.
func (o *BuildOrchestrator) DeleteJob(ctx context.Context, jobName string) error {
	start := time.Now()
	log := zerolog.Ctx(ctx).With().
		Str("op", "BuildOrchestrator.DeleteJob").
		Str("job_name", jobName).
		Str("namespace", o.cfg.Namespace).
		Logger()
	log.Debug().Msg("starting build job deletion")

	propagation := metav1.DeletePropagationBackground
	err := o.client.BatchV1().Jobs(o.cfg.Namespace).Delete(ctx, jobName, metav1.DeleteOptions{
		PropagationPolicy: &propagation,
	})
	if err != nil {
		mapped := MapK8sError(err)
		log.Error().Err(mapped).Dur("duration_ms", time.Since(start)).Msg("failed deleting build job")
		return mapped
	}

	log.Info().Dur("duration_ms", time.Since(start)).Msg("completed build job deletion")
	return nil
}

func (o *BuildOrchestrator) fetchLogsSafely(ctx context.Context, jobName string) (io.ReadCloser, string) {
	stream, err := o.StreamJobLogs(ctx, jobName)
	if err != nil {
		return io.NopCloser(strings.NewReader("")), ""
	}
	defer stream.Close()

	buf := new(bytes.Buffer)
	_, _ = io.Copy(buf, stream)
	content := buf.String()

	return io.NopCloser(bytes.NewReader(buf.Bytes())), content
}

func isJobSuccessful(job *batchv1.Job) bool {
	if job.Status.Succeeded > 0 {
		return true
	}
	for _, cond := range job.Status.Conditions {
		if cond.Type == batchv1.JobComplete && cond.Status == corev1.ConditionTrue {
			return true
		}
	}
	return false
}

func isJobFailed(job *batchv1.Job) bool {
	if job.Status.Failed > 0 {
		return true
	}
	for _, cond := range job.Status.Conditions {
		if cond.Type == batchv1.JobFailed && cond.Status == corev1.ConditionTrue {
			return true
		}
	}
	return false
}

// ExtractSHA256Checksum extracts the VUHIVE_ARTIFACT_SHA256 string from build logs if present.
func ExtractSHA256Checksum(logs string) string {
	matches := sha256Regex.FindStringSubmatch(logs)
	if len(matches) > 1 {
		return matches[1]
	}
	return ""
}

// Compile-time static interface verification
var _ outbound.BuildOrchestratorPort = (*BuildOrchestrator)(nil)
