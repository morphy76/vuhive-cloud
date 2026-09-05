package k8s_test

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	corev1 "k8s.io/api/core/v1"

	"github.com/morphy76/vuhive-cloud/internal/adapters/outbound/k8s"
	s3adapter "github.com/morphy76/vuhive-cloud/internal/adapters/outbound/s3"
	"github.com/morphy76/vuhive-cloud/internal/application/ports/outbound"
	"github.com/morphy76/vuhive-cloud/internal/domain/model"
)

func TestRunnerJobGenerator_GenerateJob(t *testing.T) {
	cfg := k8s.DefaultConfig()
	cfg.S3Endpoint = "http://minio.local:9000"
	cfg.S3Bucket = "vuhive-artifacts"
	cfg.S3AccessKeyID = "testaccess"
	cfg.S3SecretAccessKey = "testsecret"
	cfg.S3Region = "us-east-1"
	cfg.APICallbackURL = "http://api.vuhive.local/api/v1/runs"

	generator := k8s.NewRunnerJobGenerator(cfg)

	resources, err := model.NewResourceRequirements("2000m", "4000m", "2Gi", "4Gi")
	require.NoError(t, err)

	affinity := model.Affinity{
		NodeSelectorTerms: []model.NodeAffinityTerm{
			{
				Key:      "role",
				Operator: "In",
				Values:   []string{"load-generator"},
			},
		},
	}

	tolerations := []model.Toleration{
		{
			Key:      "vuhive.io/load-generator",
			Operator: "Exists",
			Effect:   "NoSchedule",
		},
	}

	nodeSelector := map[string]string{
		"disktype": "ssd",
	}

	profile, err := model.NewRunnerProfile(
		"high-perf",
		"High performance runner",
		"custom-runner:v1",
		resources,
		nodeSelector,
		affinity,
		tolerations,
	)
	require.NoError(t, err)

	t.Run("successfully generate runner job manifest with full configuration", func(t *testing.T) {
		cfgID := "cfg-99"
		run, err := model.NewTestRun("suite-123", "art-456", &cfgID, profile.ID(), nil)
		require.NoError(t, err)

		opts := outbound.RunnerJobOptions{
			S3BinaryKey: "vuhive-binaries/suite-123/art-456/linux-amd64/runner",
			S3ConfigKey: "vuhive-configs/suite-123/vuhive.yaml",
		}

		job, err := generator.GenerateJob(run, profile, opts)
		require.NoError(t, err)
		require.NotNil(t, job)

		// Metadata
		assert.Contains(t, job.Name, "vuhive-run-")
		assert.Equal(t, cfg.RunnerNamespace, job.Namespace)
		assert.Equal(t, "vuhive-runner", job.Labels["app.kubernetes.io/name"])
		assert.Equal(t, run.ID(), job.Labels["vuhive.io/run-id"])
		assert.Equal(t, "suite-123", job.Labels["vuhive.io/suite-id"])

		// Job Spec
		require.NotNil(t, job.Spec.BackoffLimit)
		assert.Equal(t, int32(0), *job.Spec.BackoffLimit)
		require.NotNil(t, job.Spec.ActiveDeadlineSeconds)
		assert.Equal(t, int64(3600), *job.Spec.ActiveDeadlineSeconds)
		require.NotNil(t, job.Spec.TTLSecondsAfterFinished)
		assert.Equal(t, int32(86400), *job.Spec.TTLSecondsAfterFinished)

		// Pod Spec
		podSpec := job.Spec.Template.Spec
		assert.Equal(t, corev1.RestartPolicyNever, podSpec.RestartPolicy)

		// Pod Security Context
		require.NotNil(t, podSpec.SecurityContext)
		assert.True(t, *podSpec.SecurityContext.RunAsNonRoot)
		assert.Equal(t, int64(10001), *podSpec.SecurityContext.RunAsUser)
		assert.Equal(t, int64(10001), *podSpec.SecurityContext.RunAsGroup)
		assert.Equal(t, int64(10001), *podSpec.SecurityContext.FSGroup)
		require.NotNil(t, podSpec.SecurityContext.SeccompProfile)
		assert.Equal(t, corev1.SeccompProfileTypeRuntimeDefault, podSpec.SecurityContext.SeccompProfile.Type)

		// NodeSelector, Affinity, Tolerations
		assert.Equal(t, "ssd", podSpec.NodeSelector["disktype"])

		require.NotNil(t, podSpec.Affinity)
		require.NotNil(t, podSpec.Affinity.NodeAffinity)
		require.NotNil(t, podSpec.Affinity.NodeAffinity.RequiredDuringSchedulingIgnoredDuringExecution)
		terms := podSpec.Affinity.NodeAffinity.RequiredDuringSchedulingIgnoredDuringExecution.NodeSelectorTerms
		require.Len(t, terms, 1)
		assert.Equal(t, "role", terms[0].MatchExpressions[0].Key)
		assert.Equal(t, corev1.NodeSelectorOpIn, terms[0].MatchExpressions[0].Operator)
		assert.Equal(t, []string{"load-generator"}, terms[0].MatchExpressions[0].Values)

		require.Len(t, podSpec.Tolerations, 1)
		assert.Equal(t, "vuhive.io/load-generator", podSpec.Tolerations[0].Key)
		assert.Equal(t, corev1.TolerationOpExists, podSpec.Tolerations[0].Operator)
		assert.Equal(t, corev1.TaintEffectNoSchedule, podSpec.Tolerations[0].Effect)

		// Volumes
		require.Len(t, podSpec.Volumes, 1)
		assert.Equal(t, "shared-workspace", podSpec.Volumes[0].Name)
		assert.NotNil(t, podSpec.Volumes[0].EmptyDir)

		// Init Container
		require.Len(t, podSpec.InitContainers, 1)
		initC := podSpec.InitContainers[0]
		assert.Equal(t, "fetch-artifacts", initC.Name)
		assert.Equal(t, cfg.RunnerInitImage, initC.Image)
		require.NotNil(t, initC.SecurityContext)
		assert.False(t, *initC.SecurityContext.AllowPrivilegeEscalation)
		assert.True(t, *initC.SecurityContext.ReadOnlyRootFilesystem)
		assert.Equal(t, corev1.Capability("ALL"), initC.SecurityContext.Capabilities.Drop[0])

		envMap := make(map[string]string)
		for _, e := range initC.Env {
			envMap[e.Name] = e.Value
		}
		assert.Equal(t, "http://minio.local:9000", envMap["S3_ENDPOINT"])
		assert.Equal(t, "vuhive-artifacts", envMap["S3_BUCKET"])
		assert.Equal(t, "testaccess", envMap["S3_ACCESS_KEY_ID"])
		assert.Equal(t, "testsecret", envMap["S3_SECRET_ACCESS_KEY"])
		assert.Equal(t, "vuhive-binaries/suite-123/art-456/linux-amd64/runner", envMap["S3_BINARY_KEY"])
		assert.Equal(t, "vuhive-configs/suite-123/vuhive.yaml", envMap["S3_CONFIG_KEY"])
		assert.Equal(t, "/shared", envMap["SHARED_DIR"])

		require.Len(t, initC.VolumeMounts, 1)
		assert.Equal(t, "shared-workspace", initC.VolumeMounts[0].Name)
		assert.Equal(t, "/shared", initC.VolumeMounts[0].MountPath)

		// Main Container
		require.Len(t, podSpec.Containers, 1)
		runnerC := podSpec.Containers[0]
		assert.Equal(t, "runner", runnerC.Name)
		assert.Equal(t, "custom-runner:v1", runnerC.Image)
		assert.Equal(t, []string{"/shared/entrypoint.sh"}, runnerC.Command)
		require.NotNil(t, runnerC.SecurityContext)
		assert.False(t, *runnerC.SecurityContext.AllowPrivilegeEscalation)
		assert.True(t, *runnerC.SecurityContext.ReadOnlyRootFilesystem)

		// Resources
		assert.Equal(t, "2", runnerC.Resources.Requests.Cpu().String())
		assert.Equal(t, "2Gi", runnerC.Resources.Requests.Memory().String())
		assert.Equal(t, "4", runnerC.Resources.Limits.Cpu().String())
		assert.Equal(t, "4Gi", runnerC.Resources.Limits.Memory().String())

		// Runner Container Envs
		runnerEnvMap := make(map[string]string)
		for _, e := range runnerC.Env {
			runnerEnvMap[e.Name] = e.Value
		}
		expectedReportKey, err := s3adapter.KeySummaryReport(run.ID())
		require.NoError(t, err)
		expectedLogsKey, err := s3adapter.KeyExecutionLogs(run.ID())
		require.NoError(t, err)

		assert.Equal(t, run.ID(), runnerEnvMap["VUHIVE_RUN_ID"])
		assert.Equal(t, expectedReportKey, runnerEnvMap["S3_REPORT_KEY"])
		assert.Equal(t, expectedLogsKey, runnerEnvMap["S3_LOGS_KEY"])
		assert.Equal(t, "http://api.vuhive.local/api/v1/runs", runnerEnvMap["API_CALLBACK_URL"])
	})

	t.Run("successfully generate worker job manifest with partitioning options", func(t *testing.T) {
		run, err := model.NewTestRun("suite-123", "art-456", nil, profile.ID(), nil)
		require.NoError(t, err)

		workerIdx := 2
		workerCnt := 5
		opts := outbound.RunnerJobOptions{
			S3BinaryKey: "vuhive-binaries/suite-123/art-456/linux-amd64/runner",
			S3ConfigKey: "vuhive-configs/suite-123/vuhive.yaml",
			WorkerIndex: &workerIdx,
			WorkerCount: &workerCnt,
			EnvVars: map[string]string{
				"VUHIVE_WORKER_INDEX":           "2",
				"VUHIVE_WORKER_COUNT":           "5",
				"VUHIVE_SCENARIOS_CHECKOUT_VUS": "4",
			},
		}

		job, err := generator.GenerateJob(run, profile, opts)
		require.NoError(t, err)
		require.NotNil(t, job)

		// Name and labels
		assert.Contains(t, job.Name, "-worker-2")
		assert.Equal(t, "2", job.Labels["vuhive.io/worker-index"])
		assert.Equal(t, "5", job.Labels["vuhive.io/worker-count"])

		// Main container env vars
		runnerC := job.Spec.Template.Spec.Containers[0]
		runnerEnvMap := make(map[string]string)
		for _, e := range runnerC.Env {
			runnerEnvMap[e.Name] = e.Value
		}

		expectedReportKey, err := s3adapter.KeyWorkerSummaryReport(run.ID(), 2)
		require.NoError(t, err)
		expectedLogsKey, err := s3adapter.KeyWorkerExecutionLogs(run.ID(), 2)
		require.NoError(t, err)

		assert.Equal(t, expectedReportKey, runnerEnvMap["S3_REPORT_KEY"])
		assert.Equal(t, expectedLogsKey, runnerEnvMap["S3_LOGS_KEY"])
		assert.Equal(t, "2", runnerEnvMap["VUHIVE_WORKER_INDEX"])
		assert.Equal(t, "5", runnerEnvMap["VUHIVE_WORKER_COUNT"])
		assert.Equal(t, "4", runnerEnvMap["VUHIVE_SCENARIOS_CHECKOUT_VUS"])
	})

	t.Run("validation error on missing binary key", func(t *testing.T) {
		run, err := model.NewTestRun("suite-123", "art-456", nil, profile.ID(), nil)
		require.NoError(t, err)

		_, err = generator.GenerateJob(run, profile, outbound.RunnerJobOptions{})
		assert.ErrorIs(t, err, model.ErrValidation)
	})

	t.Run("validation error on nil run or nil profile", func(t *testing.T) {
		_, err := generator.GenerateJob(nil, profile, outbound.RunnerJobOptions{S3BinaryKey: "some/key"})
		assert.ErrorIs(t, err, model.ErrValidation)

		run, err := model.NewTestRun("suite-123", "art-456", nil, profile.ID(), nil)
		require.NoError(t, err)

		_, err = generator.GenerateJob(run, nil, outbound.RunnerJobOptions{S3BinaryKey: "some/key"})
		assert.ErrorIs(t, err, model.ErrValidation)
	})

	t.Run("generate multi-worker job with barrier enabled", func(t *testing.T) {
		run, err := model.NewTestRun("suite-123", "art-456", nil, profile.ID(), nil)
		require.NoError(t, err)

		workerCount := 3
		opts := outbound.RunnerJobOptions{
			S3BinaryKey:    "vuhive-binaries/suite-123/art-456/linux-amd64/runner",
			WorkerCount:    &workerCount,
			BarrierEnabled: true,
			BarrierTimeout: 45 * time.Second,
		}

		job, err := generator.GenerateJob(run, profile, opts)
		require.NoError(t, err)
		require.NotNil(t, job)

		require.NotNil(t, job.Spec.Parallelism)
		assert.Equal(t, int32(3), *job.Spec.Parallelism)
		require.NotNil(t, job.Spec.Completions)
		assert.Equal(t, int32(3), *job.Spec.Completions)

		runnerC := job.Spec.Template.Spec.Containers[0]
		envMap := make(map[string]string)
		var workerIDRef *corev1.ObjectFieldSelector
		for _, e := range runnerC.Env {
			envMap[e.Name] = e.Value
			if e.Name == "VUHIVE_WORKER_ID" && e.ValueFrom != nil && e.ValueFrom.FieldRef != nil {
				workerIDRef = e.ValueFrom.FieldRef
			}
		}

		assert.Equal(t, "3", envMap["VUHIVE_WORKER_COUNT"])
		assert.Equal(t, "true", envMap["VUHIVE_BARRIER_ENABLED"])
		assert.Equal(t, "45s", envMap["VUHIVE_BARRIER_TIMEOUT"])
		require.NotNil(t, workerIDRef)
		assert.Equal(t, "metadata.name", workerIDRef.FieldPath)
	})
}

