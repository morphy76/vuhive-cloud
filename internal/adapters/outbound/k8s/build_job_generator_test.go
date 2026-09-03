package k8s_test

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	corev1 "k8s.io/api/core/v1"

	"github.com/morphy76/vuhive-cloud/internal/adapters/outbound/k8s"
	"github.com/morphy76/vuhive-cloud/internal/application/ports/outbound"
	"github.com/morphy76/vuhive-cloud/internal/domain/model"
)

func TestBuildJobGenerator_GenerateBuildJob_Amd64(t *testing.T) {
	cfg := k8s.DefaultConfig()
	generator := k8s.NewBuildJobGenerator(cfg)

	opts := outbound.BuildJobOptions{
		SuiteID:         "11111111-1111-1111-1111-111111111111",
		ArtifactID:      "22222222-2222-2222-2222-222222222222",
		Platform:        model.PlatformLinuxAmd64,
		SourceURL:       "https://s3.example.com/source.tar.gz?presigned=true",
		BinaryUploadURL: "https://s3.example.com/binary?presigned=true",
	}

	job, err := generator.GenerateBuildJob(opts)
	require.NoError(t, err)
	require.NotNil(t, job)

	// Verify Metadata
	assert.Equal(t, cfg.Namespace, job.Namespace)
	assert.Contains(t, job.Name, "vuhive-build-22222222")
	assert.Equal(t, "vuhive-builder", job.Labels["app.kubernetes.io/name"])
	assert.Equal(t, opts.SuiteID, job.Labels["vuhive.io/suite-id"])
	assert.Equal(t, opts.ArtifactID, job.Labels["vuhive.io/artifact-id"])
	assert.Equal(t, "linux-amd64", job.Labels["vuhive.io/platform"])

	// Verify Spec
	require.NotNil(t, job.Spec.BackoffLimit)
	assert.Equal(t, int32(0), *job.Spec.BackoffLimit)
	require.NotNil(t, job.Spec.ActiveDeadlineSeconds)
	assert.Equal(t, int64(600), *job.Spec.ActiveDeadlineSeconds)

	podSpec := job.Spec.Template.Spec
	assert.Equal(t, corev1.RestartPolicyNever, podSpec.RestartPolicy)

	// Pod Security Context
	require.NotNil(t, podSpec.SecurityContext)
	require.NotNil(t, podSpec.SecurityContext.RunAsNonRoot)
	assert.True(t, *podSpec.SecurityContext.RunAsNonRoot)
	require.NotNil(t, podSpec.SecurityContext.RunAsUser)
	assert.Equal(t, int64(10001), *podSpec.SecurityContext.RunAsUser)

	// Container checks
	require.Len(t, podSpec.Containers, 1)
	container := podSpec.Containers[0]
	assert.Equal(t, "builder", container.Name)
	assert.Equal(t, "golang:1.26-alpine", container.Image)

	// Security Context of container
	require.NotNil(t, container.SecurityContext)
	require.NotNil(t, container.SecurityContext.AllowPrivilegeEscalation)
	assert.False(t, *container.SecurityContext.AllowPrivilegeEscalation)
	require.NotNil(t, container.SecurityContext.Capabilities)
	assert.Contains(t, container.SecurityContext.Capabilities.Drop, corev1.Capability("ALL"))

	// Verify Environment Variables
	envMap := make(map[string]string)
	for _, env := range container.Env {
		envMap[env.Name] = env.Value
	}
	assert.Equal(t, "0", envMap["CGO_ENABLED"])
	assert.Equal(t, "linux", envMap["GOOS"])
	assert.Equal(t, "amd64", envMap["GOARCH"])
	assert.Equal(t, opts.SourceURL, envMap["SOURCE_URL"])
	assert.Equal(t, opts.BinaryUploadURL, envMap["BINARY_UPLOAD_URL"])

	// Verify Compilation Commands in script
	assert.NotEmpty(t, container.Command)
	script := container.Command[len(container.Command)-1]
	assert.Contains(t, script, "CGO_ENABLED=0")
	assert.Contains(t, script, "go build -trimpath -ldflags=\"-s -w\"")
	assert.Contains(t, script, "/workspace/bin/runner")
	assert.Contains(t, script, "VUHIVE_ARTIFACT_SHA256=")

	// Verify Volume Mounts
	require.Len(t, container.VolumeMounts, 1)
	assert.Equal(t, "workspace", container.VolumeMounts[0].Name)
	assert.Equal(t, "/workspace", container.VolumeMounts[0].MountPath)
}

func TestBuildJobGenerator_GenerateBuildJob_Arm64(t *testing.T) {
	cfg := k8s.DefaultConfig()
	generator := k8s.NewBuildJobGenerator(cfg)

	opts := outbound.BuildJobOptions{
		SuiteID:         "11111111-1111-1111-1111-111111111111",
		ArtifactID:      "33333333-3333-3333-3333-333333333333",
		Platform:        model.PlatformLinuxArm64,
		SourceURL:       "https://s3.example.com/source.tar.gz",
		BinaryUploadURL: "https://s3.example.com/binary",
	}

	job, err := generator.GenerateBuildJob(opts)
	require.NoError(t, err)
	require.NotNil(t, job)

	assert.Equal(t, "linux-arm64", job.Labels["vuhive.io/platform"])

	container := job.Spec.Template.Spec.Containers[0]
	envMap := make(map[string]string)
	for _, env := range container.Env {
		envMap[env.Name] = env.Value
	}
	assert.Equal(t, "arm64", envMap["GOARCH"])
}

func TestBuildJobGenerator_ValidationErrors(t *testing.T) {
	cfg := k8s.DefaultConfig()
	generator := k8s.NewBuildJobGenerator(cfg)

	t.Run("empty suiteID fails", func(t *testing.T) {
		_, err := generator.GenerateBuildJob(outbound.BuildJobOptions{
			SuiteID:         "",
			ArtifactID:      "art-1",
			Platform:        model.PlatformLinuxAmd64,
			SourceURL:       "http://source",
			BinaryUploadURL: "http://upload",
		})
		assert.ErrorIs(t, err, model.ErrValidation)
	})

	t.Run("empty artifactID fails", func(t *testing.T) {
		_, err := generator.GenerateBuildJob(outbound.BuildJobOptions{
			SuiteID:         "suite-1",
			ArtifactID:      "",
			Platform:        model.PlatformLinuxAmd64,
			SourceURL:       "http://source",
			BinaryUploadURL: "http://upload",
		})
		assert.ErrorIs(t, err, model.ErrValidation)
	})

	t.Run("invalid platform fails", func(t *testing.T) {
		_, err := generator.GenerateBuildJob(outbound.BuildJobOptions{
			SuiteID:         "suite-1",
			ArtifactID:      "art-1",
			Platform:        model.Platform("windows/amd64"),
			SourceURL:       "http://source",
			BinaryUploadURL: "http://upload",
		})
		assert.ErrorIs(t, err, model.ErrInvalidPlatform)
	})

	t.Run("empty URLs fail", func(t *testing.T) {
		_, err := generator.GenerateBuildJob(outbound.BuildJobOptions{
			SuiteID:         "suite-1",
			ArtifactID:      "art-1",
			Platform:        model.PlatformLinuxAmd64,
			SourceURL:       "",
			BinaryUploadURL: "http://upload",
		})
		assert.ErrorIs(t, err, model.ErrValidation)

		_, err = generator.GenerateBuildJob(outbound.BuildJobOptions{
			SuiteID:         "suite-1",
			ArtifactID:      "art-1",
			Platform:        model.PlatformLinuxAmd64,
			SourceURL:       "http://source",
			BinaryUploadURL: "",
		})
		assert.ErrorIs(t, err, model.ErrValidation)
	})
}
