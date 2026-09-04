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

func TestScheduleOrchestrator(t *testing.T) {
	ctx := context.Background()
	fakeClient := fake.NewSimpleClientset()
	cfg := k8s.Config{
		RunnerNamespace: "vuhive-runners",
	}

	orchestrator := k8s.NewScheduleOrchestrator(fakeClient, cfg)

	res, err := model.NewResourceRequirements("500m", "1000m", "256Mi", "512Mi")
	require.NoError(t, err)

	profile, err := model.NewRunnerProfile(
		"test-profile",
		"Test Profile",
		"alpine:3.20",
		res,
		nil,
		model.Affinity{},
		nil,
	)
	require.NoError(t, err)

	schedule, err := model.NewSchedule(
		"suite-1",
		"art-1",
		nil,
		profile.ID(),
		"daily-test",
		"0 1 * * *",
	)
	require.NoError(t, err)

	opts := outbound.RunnerJobOptions{
		S3BinaryKey: "artifacts/runner",
	}

	t.Run("CreateCronJob creates batch/v1 CronJob on Kubernetes", func(t *testing.T) {
		name, err := orchestrator.CreateCronJob(ctx, schedule, profile, opts)
		require.NoError(t, err)
		assert.Equal(t, schedule.K8sCronJobName(), name)

		created, err := fakeClient.BatchV1().CronJobs("vuhive-runners").Get(ctx, name, metav1.GetOptions{})
		require.NoError(t, err)
		assert.Equal(t, "0 1 * * *", created.Spec.Schedule)
		assert.Equal(t, schedule.ID(), created.Labels["vuhive.io/schedule-id"])
	})

	t.Run("UpdateCronJob modifies schedule expression on existing CronJob", func(t *testing.T) {
		require.NoError(t, schedule.UpdateCronExpression("*/30 * * * *"))

		err := orchestrator.UpdateCronJob(ctx, schedule)
		require.NoError(t, err)

		updated, err := fakeClient.BatchV1().CronJobs("vuhive-runners").Get(ctx, schedule.K8sCronJobName(), metav1.GetOptions{})
		require.NoError(t, err)
		assert.Equal(t, "*/30 * * * *", updated.Spec.Schedule)
	})

	t.Run("DeleteCronJob removes CronJob from Kubernetes", func(t *testing.T) {
		err := orchestrator.DeleteCronJob(ctx, schedule.K8sCronJobName(), "vuhive-runners")
		require.NoError(t, err)

		_, err = fakeClient.BatchV1().CronJobs("vuhive-runners").Get(ctx, schedule.K8sCronJobName(), metav1.GetOptions{})
		assert.Error(t, err)
	})
}
