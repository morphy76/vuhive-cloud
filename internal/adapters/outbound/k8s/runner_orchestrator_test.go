package k8s_test

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes/fake"

	"github.com/morphy76/vuhive-cloud/internal/adapters/outbound/k8s"
	"github.com/morphy76/vuhive-cloud/internal/application/ports/outbound"
	"github.com/morphy76/vuhive-cloud/internal/domain/model"
)

func TestRunnerOrchestrator(t *testing.T) {
	ctx := context.Background()
	cfg := k8s.DefaultConfig()
	cfg.RunnerNamespace = "vuhive-runners"

	resources, err := model.NewResourceRequirements("1000m", "2000m", "1Gi", "2Gi")
	require.NoError(t, err)

	profile, err := model.NewRunnerProfile(
		"default",
		"Default runner profile",
		"alpine:3.20",
		resources,
		nil,
		model.Affinity{},
		nil,
	)
	require.NoError(t, err)

	t.Run("successfully dispatch job into kubernetes cluster", func(t *testing.T) {
		fakeClient := fake.NewSimpleClientset()
		orchestrator := k8s.NewRunnerOrchestrator(fakeClient, cfg)

		run, err := model.NewTestRun("suite-1", "art-1", nil, profile.ID(), nil)
		require.NoError(t, err)

		opts := outbound.RunnerJobOptions{
			S3BinaryKey: "vuhive-binaries/suite-1/art-1/runner",
			S3ConfigKey: "vuhive-configs/suite-1/config.yaml",
		}

		jobName, err := orchestrator.DispatchJob(ctx, run, profile, opts)
		require.NoError(t, err)
		assert.Contains(t, jobName, "vuhive-run-")

		// Verify job exists in fake cluster
		job, err := fakeClient.BatchV1().Jobs("vuhive-runners").Get(ctx, jobName, metav1.GetOptions{})
		require.NoError(t, err)
		assert.Equal(t, jobName, job.Name)
		assert.Equal(t, run.ID(), job.Labels["vuhive.io/run-id"])
	})

	t.Run("dispatch fails with validation error on nil run", func(t *testing.T) {
		fakeClient := fake.NewSimpleClientset()
		orchestrator := k8s.NewRunnerOrchestrator(fakeClient, cfg)

		opts := outbound.RunnerJobOptions{
			S3BinaryKey: "key",
		}
		_, err := orchestrator.DispatchJob(ctx, nil, profile, opts)
		assert.ErrorIs(t, err, model.ErrValidation)
	})

	t.Run("successfully abort existing job", func(t *testing.T) {
		fakeClient := fake.NewSimpleClientset()
		orchestrator := k8s.NewRunnerOrchestrator(fakeClient, cfg)

		run, err := model.NewTestRun("suite-1", "art-1", nil, profile.ID(), nil)
		require.NoError(t, err)

		opts := outbound.RunnerJobOptions{
			S3BinaryKey: "vuhive-binaries/suite-1/art-1/runner",
		}
		jobName, err := orchestrator.DispatchJob(ctx, run, profile, opts)
		require.NoError(t, err)

		err = orchestrator.AbortJob(ctx, jobName, "vuhive-runners")
		require.NoError(t, err)

		_, err = fakeClient.BatchV1().Jobs("vuhive-runners").Get(ctx, jobName, metav1.GetOptions{})
		assert.Error(t, err)
	})

	t.Run("abort fails when job does not exist", func(t *testing.T) {
		fakeClient := fake.NewSimpleClientset()
		orchestrator := k8s.NewRunnerOrchestrator(fakeClient, cfg)

		err := orchestrator.AbortJob(ctx, "non-existent-job", "vuhive-runners")
		assert.ErrorIs(t, err, model.ErrNotFound)
	})
}
