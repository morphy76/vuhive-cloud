package k8s_test

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	batchv1 "k8s.io/api/batch/v1"
	corev1 "k8s.io/api/core/v1"

	"github.com/morphy76/vuhive-cloud/internal/adapters/outbound/k8s"
	"github.com/morphy76/vuhive-cloud/internal/application/ports/outbound"
	"github.com/morphy76/vuhive-cloud/internal/domain/model"
)

func TestCronJobGenerator_GenerateCronJob(t *testing.T) {
	cfg := k8s.Config{
		RunnerNamespace:               "custom-runners",
		RunnerDefaultImage:            "ghcr.io/morphy76/vuhive-cloud/runner-default:v1.0",
		RunnerInitImage:               "ghcr.io/morphy76/vuhive-cloud/runner-init:v1.0",
		RunnerActiveDeadlineSeconds:   1800,
		RunnerBackoffLimit:            0,
		RunnerTTLSecondsAfterFinished: 3600,
		S3Endpoint:                    "http://minio:9000",
		S3Region:                      "us-east-1",
		S3Bucket:                      "vuhive-artifacts",
		S3AccessKeyID:                 "minioadmin",
		S3SecretAccessKey:             "miniopassword",
		S3UsePathStyle:                true,
		APICallbackURL:                "http://vuhive-cloud:8080/api/v1/runs/complete",
	}

	generator := k8s.NewCronJobGenerator(cfg)

	suiteID := "suite-1111-2222"
	artifactID := "art-3333-4444"
	configID := "cfg-5555-6666"
	profileID := "prof-7777-8888"

	res, err := model.NewResourceRequirements("500m", "1000m", "256Mi", "512Mi")
	require.NoError(t, err)

	affinity := model.Affinity{
		NodeSelectorTerms: []model.NodeAffinityTerm{
			{
				Key:      "topology.kubernetes.io/zone",
				Operator: "In",
				Values:   []string{"zone-a", "zone-b"},
			},
		},
	}

	tolerations := []model.Toleration{
		{
			Key:      "dedicated",
			Operator: "Equal",
			Value:    "load-tests",
			Effect:   "NoSchedule",
		},
	}

	nodeSelector := map[string]string{
		"node-role.kubernetes.io/worker": "true",
	}

	profile, err := model.NewRunnerProfile(
		"dedicated-profile",
		"Profile for load test execution",
		"custom-runner:v2",
		res,
		nodeSelector,
		affinity,
		tolerations,
	)
	require.NoError(t, err)

	schedule, err := model.NewSchedule(
		suiteID,
		artifactID,
		&configID,
		profileID,
		"nightly-load-test",
		"0 2 * * *",
	)
	require.NoError(t, err)

	opts := outbound.RunnerJobOptions{
		S3BinaryKey: "suites/suite-1111-2222/artifacts/art-3333-4444/runner",
		S3ConfigKey: "suites/suite-1111-2222/configs/cfg-5555-6666/vuhive.yaml",
	}

	t.Run("successfully generates complete CronJob manifest", func(t *testing.T) {
		cronJob, err := generator.GenerateCronJob(schedule, profile, opts)
		require.NoError(t, err)
		require.NotNil(t, cronJob)

		// Metadata validation
		assert.Equal(t, schedule.K8sCronJobName(), cronJob.Name)
		assert.Equal(t, "custom-runners", cronJob.Namespace)
		assert.Equal(t, "vuhive-runner", cronJob.Labels["app.kubernetes.io/name"])
		assert.Equal(t, schedule.ID(), cronJob.Labels["vuhive.io/schedule-id"])
		assert.Equal(t, suiteID, cronJob.Labels["vuhive.io/suite-id"])

		// Spec validation
		assert.Equal(t, "0 2 * * *", cronJob.Spec.Schedule)
		assert.Equal(t, batchv1.ForbidConcurrent, cronJob.Spec.ConcurrencyPolicy)

		// Job template validation
		jobMeta := cronJob.Spec.JobTemplate.ObjectMeta
		assert.Equal(t, "vuhive-runner", jobMeta.Labels["app.kubernetes.io/name"])
		assert.Equal(t, schedule.ID(), jobMeta.Labels["vuhive.io/schedule-id"])
		assert.Equal(t, suiteID, jobMeta.Labels["vuhive.io/suite-id"])
		assert.Equal(t, artifactID, jobMeta.Labels["vuhive.io/artifact-id"])
		assert.Equal(t, profileID, jobMeta.Labels["vuhive.io/runner-profile-id"])

		jobSpec := cronJob.Spec.JobTemplate.Spec
		require.NotNil(t, jobSpec.BackoffLimit)
		assert.Equal(t, int32(0), *jobSpec.BackoffLimit)
		require.NotNil(t, jobSpec.ActiveDeadlineSeconds)
		assert.Equal(t, int64(1800), *jobSpec.ActiveDeadlineSeconds)

		// Pod template validation
		podSpec := jobSpec.Template.Spec
		assert.Equal(t, corev1.RestartPolicyNever, podSpec.RestartPolicy)
		assert.Equal(t, nodeSelector, podSpec.NodeSelector)
		require.NotNil(t, podSpec.Affinity)
		require.NotNil(t, podSpec.Affinity.NodeAffinity)
		assert.Len(t, podSpec.Tolerations, 1)
		assert.Equal(t, "dedicated", podSpec.Tolerations[0].Key)

		// Init container
		require.Len(t, podSpec.InitContainers, 1)
		initC := podSpec.InitContainers[0]
		assert.Equal(t, "fetch-artifacts", initC.Name)
		assert.Equal(t, "ghcr.io/morphy76/vuhive-cloud/runner-init:v1.0", initC.Image)

		findEnv := func(envs []corev1.EnvVar, name string) *corev1.EnvVar {
			for _, e := range envs {
				if e.Name == name {
					return &e
				}
			}
			return nil
		}

		binEnv := findEnv(initC.Env, "S3_BINARY_KEY")
		require.NotNil(t, binEnv)
		assert.Equal(t, opts.S3BinaryKey, binEnv.Value)

		cfgEnv := findEnv(initC.Env, "S3_CONFIG_KEY")
		require.NotNil(t, cfgEnv)
		assert.Equal(t, opts.S3ConfigKey, cfgEnv.Value)

		// Runner container
		require.Len(t, podSpec.Containers, 1)
		runnerC := podSpec.Containers[0]
		assert.Equal(t, "runner", runnerC.Name)
		assert.Equal(t, "custom-runner:v2", runnerC.Image)
		assert.Equal(t, []string{"/shared/entrypoint.sh"}, runnerC.Command)

		assert.Equal(t, "500m", runnerC.Resources.Requests.Cpu().String())
		assert.Equal(t, "1", runnerC.Resources.Limits.Cpu().String())
		assert.Equal(t, "256Mi", runnerC.Resources.Requests.Memory().String())
		assert.Equal(t, "512Mi", runnerC.Resources.Limits.Memory().String())
	})

	t.Run("validation errors on invalid inputs", func(t *testing.T) {
		_, err := generator.GenerateCronJob(nil, profile, opts)
		assert.ErrorIs(t, err, model.ErrValidation)

		_, err = generator.GenerateCronJob(schedule, nil, opts)
		assert.ErrorIs(t, err, model.ErrValidation)

		_, err = generator.GenerateCronJob(schedule, profile, outbound.RunnerJobOptions{})
		assert.ErrorIs(t, err, model.ErrValidation)
	})
}
