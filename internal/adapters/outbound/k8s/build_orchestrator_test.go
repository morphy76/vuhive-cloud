package k8s_test

import (
	"context"
	"io"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	batchv1 "k8s.io/api/batch/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/kubernetes/fake"

	"github.com/morphy76/vuhive-cloud/internal/adapters/outbound/k8s"
	"github.com/morphy76/vuhive-cloud/internal/application/ports/outbound"
	"github.com/morphy76/vuhive-cloud/internal/domain/model"
)

func TestBuildOrchestrator_DispatchBuildJob(t *testing.T) {
	cfg := k8s.DefaultConfig()
	client := fake.NewSimpleClientset()
	orchestrator := k8s.NewBuildOrchestrator(client, cfg)

	ctx := context.Background()
	opts := outbound.BuildJobOptions{
		SuiteID:         "11111111-1111-1111-1111-111111111111",
		ArtifactID:      "22222222-2222-2222-2222-222222222222",
		Platform:        model.PlatformLinuxAmd64,
		SourceURL:       "https://s3.example.com/source.tar.gz",
		BinaryUploadURL: "https://s3.example.com/binary",
	}

	jobName, err := orchestrator.DispatchBuildJob(ctx, opts)
	require.NoError(t, err)
	assert.NotEmpty(t, jobName)

	// Verify job was created in the fake clientset
	job, err := client.BatchV1().Jobs(cfg.Namespace).Get(ctx, jobName, metav1.GetOptions{})
	require.NoError(t, err)
	assert.Equal(t, jobName, job.Name)
	assert.Equal(t, cfg.Namespace, job.Namespace)
}

func TestBuildOrchestrator_WaitForJob_Success(t *testing.T) {
	cfg := k8s.DefaultConfig()
	cfg.PollInterval = 10 * time.Millisecond

	jobName := "vuhive-build-success"
	expectedChecksum := "e3b0c44298fc1c149afbf4c8996fb92427ae41e4649b934ca495991b7852b855"

	// Create job in fake client
	job := &batchv1.Job{
		ObjectMeta: metav1.ObjectMeta{
			Name:      jobName,
			Namespace: cfg.Namespace,
		},
		Status: batchv1.JobStatus{
			Succeeded: 1,
			Conditions: []batchv1.JobCondition{
				{
					Type:   batchv1.JobComplete,
					Status: corev1.ConditionTrue,
				},
			},
		},
	}

	// Create associated Pod in fake client
	pod := &corev1.Pod{
		ObjectMeta: metav1.ObjectMeta{
			Name:      jobName + "-pod",
			Namespace: cfg.Namespace,
			Labels: map[string]string{
				"job-name": jobName,
			},
		},
		Status: corev1.PodStatus{
			Phase: corev1.PodSucceeded,
		},
	}

	client := fake.NewSimpleClientset(job, pod)
	orchestrator := k8s.NewBuildOrchestrator(client, cfg)

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	exec, err := orchestrator.WaitForJob(ctx, jobName)
	require.NoError(t, err)
	assert.Equal(t, jobName, exec.JobName)
	assert.Equal(t, 0, exec.ExitCode)

	// Note: in fake client, logs may be empty or contain fake output
	if exec.Logs != nil {
		defer exec.Logs.Close()
		_, _ = io.ReadAll(exec.Logs)
	}

	// ParseChecksumHelper check
	parsed := k8s.ExtractSHA256Checksum("some logs...\nVUHIVE_ARTIFACT_SHA256=" + expectedChecksum + "\ndone")
	assert.Equal(t, expectedChecksum, parsed)
}

func TestBuildOrchestrator_WaitForJob_Failure(t *testing.T) {
	cfg := k8s.DefaultConfig()
	cfg.PollInterval = 10 * time.Millisecond

	jobName := "vuhive-build-failure"

	job := &batchv1.Job{
		ObjectMeta: metav1.ObjectMeta{
			Name:      jobName,
			Namespace: cfg.Namespace,
		},
		Status: batchv1.JobStatus{
			Failed: 1,
			Conditions: []batchv1.JobCondition{
				{
					Type:    batchv1.JobFailed,
					Status:  corev1.ConditionTrue,
					Reason:  "BackoffLimitExceeded",
					Message: "Job has reached the specified backoff limit",
				},
			},
		},
	}

	pod := &corev1.Pod{
		ObjectMeta: metav1.ObjectMeta{
			Name:      jobName + "-pod",
			Namespace: cfg.Namespace,
			Labels: map[string]string{
				"job-name": jobName,
			},
		},
		Status: corev1.PodStatus{
			Phase: corev1.PodFailed,
		},
	}

	client := fake.NewSimpleClientset(job, pod)
	orchestrator := k8s.NewBuildOrchestrator(client, cfg)

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	exec, err := orchestrator.WaitForJob(ctx, jobName)
	assert.ErrorIs(t, err, model.ErrBuildFailed)
	require.NotNil(t, exec)
	assert.Equal(t, jobName, exec.JobName)
	assert.Equal(t, 1, exec.ExitCode)
	if exec.Logs != nil {
		_ = exec.Logs.Close()
	}
}

func TestBuildOrchestrator_DeleteJob(t *testing.T) {
	cfg := k8s.DefaultConfig()
	jobName := "vuhive-build-delete"

	job := &batchv1.Job{
		ObjectMeta: metav1.ObjectMeta{
			Name:      jobName,
			Namespace: cfg.Namespace,
		},
	}

	client := fake.NewSimpleClientset(job)
	orchestrator := k8s.NewBuildOrchestrator(client, cfg)

	ctx := context.Background()
	err := orchestrator.DeleteJob(ctx, jobName)
	require.NoError(t, err)

	// Verify job is deleted
	_, err = client.BatchV1().Jobs(cfg.Namespace).Get(ctx, jobName, metav1.GetOptions{})
	assert.Error(t, err)
}

func TestBuildOrchestrator_WaitForJob_ContextTimeout(t *testing.T) {
	cfg := k8s.DefaultConfig()
	cfg.PollInterval = 50 * time.Millisecond

	jobName := "vuhive-build-running"

	// Running job without complete or failed status
	job := &batchv1.Job{
		ObjectMeta: metav1.ObjectMeta{
			Name:      jobName,
			Namespace: cfg.Namespace,
		},
		Status: batchv1.JobStatus{
			Active: 1,
		},
	}

	client := fake.NewSimpleClientset([]runtime.Object{job}...)
	orchestrator := k8s.NewBuildOrchestrator(client, cfg)

	ctx, cancel := context.WithTimeout(context.Background(), 100*time.Millisecond)
	defer cancel()

	_, err := orchestrator.WaitForJob(ctx, jobName)
	assert.ErrorIs(t, err, model.ErrTimeout)
}
