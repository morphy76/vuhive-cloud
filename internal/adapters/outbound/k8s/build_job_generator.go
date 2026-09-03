package k8s

import (
	"fmt"
	"strings"

	batchv1 "k8s.io/api/batch/v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/morphy76/vuhive-cloud/internal/application/ports/outbound"
	"github.com/morphy76/vuhive-cloud/internal/domain/model"
)

// BuildJobGenerator constructs Kubernetes batch/v1 Job manifests for compiling Go test suites.
type BuildJobGenerator struct {
	cfg Config
}

// NewBuildJobGenerator creates a new BuildJobGenerator with the given configuration.
func NewBuildJobGenerator(cfg Config) *BuildJobGenerator {
	return &BuildJobGenerator{cfg: cfg}
}

// GenerateBuildJob generates an isolated, ephemeral Kubernetes batch/v1 Job for artifact compilation.
func (g *BuildJobGenerator) GenerateBuildJob(opts outbound.BuildJobOptions) (*batchv1.Job, error) {
	trimmedSuiteID := strings.TrimSpace(opts.SuiteID)
	if trimmedSuiteID == "" {
		return nil, fmt.Errorf("%w: suiteID cannot be empty", model.ErrValidation)
	}

	trimmedArtifactID := strings.TrimSpace(opts.ArtifactID)
	if trimmedArtifactID == "" {
		return nil, fmt.Errorf("%w: artifactID cannot be empty", model.ErrValidation)
	}

	if !opts.Platform.IsValid() {
		return nil, model.ErrInvalidPlatform
	}

	trimmedSourceURL := strings.TrimSpace(opts.SourceURL)
	if trimmedSourceURL == "" {
		return nil, fmt.Errorf("%w: sourceURL cannot be empty", model.ErrValidation)
	}

	trimmedBinaryUploadURL := strings.TrimSpace(opts.BinaryUploadURL)
	if trimmedBinaryUploadURL == "" {
		return nil, fmt.Errorf("%w: binaryUploadURL cannot be empty", model.ErrValidation)
	}

	var goarch string
	var platformLabel string
	switch opts.Platform {
	case model.PlatformLinuxAmd64:
		goarch = "amd64"
		platformLabel = "linux-amd64"
	case model.PlatformLinuxArm64:
		goarch = "arm64"
		platformLabel = "linux-arm64"
	default:
		return nil, model.ErrInvalidPlatform
	}

	jobName := formatBuildJobName(trimmedArtifactID)
	runAsUser := int64(10001)
	runAsGroup := int64(10001)
	runAsNonRoot := true
	allowPrivilegeEscalation := false
	backoffLimit := g.cfg.BackoffLimit
	activeDeadlineSeconds := g.cfg.ActiveDeadlineSeconds
	ttlSecondsAfterFinished := g.cfg.TTLSecondsAfterFinished

	buildScript := fmt.Sprintf(`set -e
mkdir -p /workspace/src /workspace/bin /workspace/.cache /workspace/go
echo "Downloading source archive..."
wget -qO /workspace/source.tar.gz "${SOURCE_URL}"
echo "Extracting source archive..."
tar -xzf /workspace/source.tar.gz -C /workspace/src
cd /workspace/src
echo "Compiling static binary for GOOS=linux GOARCH=%s..."
CGO_ENABLED=0 GOOS=linux GOARCH=%s go build -trimpath -ldflags="-s -w" -o /workspace/bin/runner .
echo "Calculating SHA256 checksum..."
CHECKSUM=$(sha256sum /workspace/bin/runner | awk '{print $1}')
echo "VUHIVE_ARTIFACT_SHA256=${CHECKSUM}"
echo "Uploading binary to storage..."
cat << 'EOF' > /workspace/upload.go
package main
import (
	"fmt"
	"net/http"
	"os"
)
func main() {
	if len(os.Args) < 3 {
		fmt.Fprintf(os.Stderr, "usage: %%s <file> <upload_url>\n", os.Args[0])
		os.Exit(1)
	}
	f, err := os.Open(os.Args[1])
	if err != nil {
		fmt.Fprintf(os.Stderr, "failed to open file: %%v\n", err)
		os.Exit(1)
	}
	defer f.Close()
	fi, err := f.Stat()
	if err != nil {
		fmt.Fprintf(os.Stderr, "failed to stat file: %%v\n", err)
		os.Exit(1)
	}
	req, err := http.NewRequest(http.MethodPut, os.Args[2], f)
	if err != nil {
		fmt.Fprintf(os.Stderr, "failed to create request: %%v\n", err)
		os.Exit(1)
	}
	req.ContentLength = fi.Size()
	req.Header.Set("Content-Type", "application/octet-stream")
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		fmt.Fprintf(os.Stderr, "upload request failed: %%v\n", err)
		os.Exit(1)
	}
	defer resp.Body.Close()
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		fmt.Fprintf(os.Stderr, "upload failed with HTTP status: %%s\n", resp.Status)
		os.Exit(2)
	}
	fmt.Println("Binary upload completed successfully")
}
EOF
go run /workspace/upload.go /workspace/bin/runner "${BINARY_UPLOAD_URL}"
echo "Build completed successfully"
`, goarch, goarch)

	job := &batchv1.Job{
		ObjectMeta: metav1.ObjectMeta{
			Name:      jobName,
			Namespace: g.cfg.Namespace,
			Labels: map[string]string{
				"app.kubernetes.io/name": "vuhive-builder",
				"vuhive.io/suite-id":     trimmedSuiteID,
				"vuhive.io/artifact-id":  trimmedArtifactID,
				"vuhive.io/platform":     platformLabel,
			},
		},
		Spec: batchv1.JobSpec{
			BackoffLimit:            &backoffLimit,
			ActiveDeadlineSeconds:   &activeDeadlineSeconds,
			TTLSecondsAfterFinished: &ttlSecondsAfterFinished,
			Template: corev1.PodTemplateSpec{
				ObjectMeta: metav1.ObjectMeta{
					Labels: map[string]string{
						"app.kubernetes.io/name": "vuhive-builder",
						"vuhive.io/suite-id":     trimmedSuiteID,
						"vuhive.io/artifact-id":  trimmedArtifactID,
						"vuhive.io/platform":     platformLabel,
					},
				},
				Spec: corev1.PodSpec{
					RestartPolicy: corev1.RestartPolicyNever,
					SecurityContext: &corev1.PodSecurityContext{
						RunAsNonRoot: &runAsNonRoot,
						RunAsUser:    &runAsUser,
						RunAsGroup:   &runAsGroup,
						FSGroup:      &runAsGroup,
						SeccompProfile: &corev1.SeccompProfile{
							Type: corev1.SeccompProfileTypeRuntimeDefault,
						},
					},
					Volumes: []corev1.Volume{
						{
							Name: "workspace",
							VolumeSource: corev1.VolumeSource{
								EmptyDir: &corev1.EmptyDirVolumeSource{},
							},
						},
					},
					Containers: []corev1.Container{
						{
							Name:            "builder",
							Image:           g.cfg.BuilderImage,
							ImagePullPolicy: corev1.PullIfNotPresent,
							SecurityContext: &corev1.SecurityContext{
								AllowPrivilegeEscalation: &allowPrivilegeEscalation,
								Capabilities: &corev1.Capabilities{
									Drop: []corev1.Capability{"ALL"},
								},
							},
							Command: []string{"/bin/sh", "-c", buildScript},
							Env: []corev1.EnvVar{
								{Name: "CGO_ENABLED", Value: "0"},
								{Name: "GOOS", Value: "linux"},
								{Name: "GOARCH", Value: goarch},
								{Name: "GOCACHE", Value: "/workspace/.cache"},
								{Name: "GOPATH", Value: "/workspace/go"},
								{Name: "SOURCE_URL", Value: trimmedSourceURL},
								{Name: "BINARY_UPLOAD_URL", Value: trimmedBinaryUploadURL},
							},
							Resources: corev1.ResourceRequirements{
								Requests: corev1.ResourceList{
									corev1.ResourceCPU:    resource.MustParse(g.cfg.CPURequest),
									corev1.ResourceMemory: resource.MustParse(g.cfg.MemoryRequest),
								},
								Limits: corev1.ResourceList{
									corev1.ResourceCPU:    resource.MustParse(g.cfg.CPULimit),
									corev1.ResourceMemory: resource.MustParse(g.cfg.MemoryLimit),
								},
							},
							VolumeMounts: []corev1.VolumeMount{
								{
									Name:      "workspace",
									MountPath: "/workspace",
								},
							},
						},
					},
				},
			},
		},
	}

	return job, nil
}

func formatBuildJobName(artifactID string) string {
	cleaned := strings.ToLower(artifactID)
	name := fmt.Sprintf("vuhive-build-%s", cleaned)
	if len(name) > 63 {
		name = name[:63]
	}
	return strings.TrimRight(name, "-")
}
