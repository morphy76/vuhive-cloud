package k8s

import (
	"fmt"
	"sort"
	"strconv"
	"strings"

	batchv1 "k8s.io/api/batch/v1"

	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/morphy76/vuhive-cloud/internal/adapters/outbound/s3"
	"github.com/morphy76/vuhive-cloud/internal/application/ports/outbound"
	"github.com/morphy76/vuhive-cloud/internal/domain/model"
)

// RunnerJobGenerator constructs Kubernetes batch/v1 Job manifests for executing load test runners.
type RunnerJobGenerator struct {
	cfg Config
}

// NewRunnerJobGenerator creates a new RunnerJobGenerator with the given configuration.
func NewRunnerJobGenerator(cfg Config) *RunnerJobGenerator {
	return &RunnerJobGenerator{cfg: cfg}
}

// GenerateJob constructs a Kubernetes batch/v1 Job manifest according to the test run and runner profile.
func (g *RunnerJobGenerator) GenerateJob(
	run *model.TestRun,
	profile *model.RunnerProfile,
	opts outbound.RunnerJobOptions,
) (*batchv1.Job, error) {
	if run == nil {
		return nil, fmt.Errorf("%w: test run cannot be nil", model.ErrValidation)
	}
	if profile == nil {
		return nil, fmt.Errorf("%w: runner profile cannot be nil", model.ErrValidation)
	}

	binaryKey := strings.TrimSpace(opts.S3BinaryKey)
	if binaryKey == "" {
		return nil, fmt.Errorf("%w: S3 binary key cannot be empty", model.ErrValidation)
	}
	configKey := strings.TrimSpace(opts.S3ConfigKey)

	namespace := strings.TrimSpace(run.K8sNamespace())
	if namespace == "" {
		namespace = strings.TrimSpace(g.cfg.RunnerNamespace)
	}
	if namespace == "" {
		namespace = model.DefaultRunnerNamespace
	}

	suffix := strings.TrimSpace(opts.JobNameSuffix)
	if suffix == "" && opts.WorkerIndex != nil {
		suffix = fmt.Sprintf("worker-%d", *opts.WorkerIndex)
	}
	jobName := formatRunnerJobName(run.ID(), suffix)

	runAsUser := int64(10001)
	runAsGroup := int64(10001)
	runAsNonRoot := true
	allowPrivilegeEscalation := false
	readOnlyRootFilesystem := true

	backoffLimit := g.cfg.RunnerBackoffLimit
	activeDeadlineSeconds := g.cfg.RunnerActiveDeadlineSeconds
	if activeDeadlineSeconds <= 0 {
		activeDeadlineSeconds = 3600
	}
	ttlSecondsAfterFinished := g.cfg.RunnerTTLSecondsAfterFinished
	if ttlSecondsAfterFinished <= 0 {
		ttlSecondsAfterFinished = 86400
	}

	initImage := strings.TrimSpace(g.cfg.RunnerInitImage)
	if initImage == "" {
		initImage = "ghcr.io/morphy76/vuhive-cloud/runner-init:latest"
	}

	runnerImage := strings.TrimSpace(profile.RunnerImage())
	if runnerImage == "" {
		runnerImage = strings.TrimSpace(g.cfg.RunnerDefaultImage)
	}
	if runnerImage == "" {
		runnerImage = model.DefaultRunnerImage
	}

	labels := map[string]string{
		"app.kubernetes.io/name": "vuhive-runner",
		"vuhive.io/run-id":       run.ID(),
		"vuhive.io/suite-id":     run.SuiteID(),
	}
	if run.ScheduleID() != nil && *run.ScheduleID() != "" {
		labels["vuhive.io/schedule-id"] = *run.ScheduleID()
	}
	if opts.WorkerIndex != nil {
		labels["vuhive.io/worker-index"] = strconv.Itoa(*opts.WorkerIndex)
	}
	if opts.WorkerCount != nil {
		labels["vuhive.io/worker-count"] = strconv.Itoa(*opts.WorkerCount)
	}


	// Affinity mapping
	var k8sAffinity *corev1.Affinity
	if len(profile.Affinity().NodeSelectorTerms) > 0 {
		var terms []corev1.NodeSelectorTerm
		for _, term := range profile.Affinity().NodeSelectorTerms {
			terms = append(terms, corev1.NodeSelectorTerm{
				MatchExpressions: []corev1.NodeSelectorRequirement{
					{
						Key:      term.Key,
						Operator: corev1.NodeSelectorOperator(term.Operator),
						Values:   term.Values,
					},
				},
			})
		}
		k8sAffinity = &corev1.Affinity{
			NodeAffinity: &corev1.NodeAffinity{
				RequiredDuringSchedulingIgnoredDuringExecution: &corev1.NodeSelector{
					NodeSelectorTerms: terms,
				},
			},
		}
	}

	// Tolerations mapping
	var k8sTolerations []corev1.Toleration
	for _, tol := range profile.Tolerations() {
		k8sTolerations = append(k8sTolerations, corev1.Toleration{
			Key:               tol.Key,
			Operator:          corev1.TolerationOperator(tol.Operator),
			Value:             tol.Value,
			Effect:            corev1.TaintEffect(tol.Effect),
			TolerationSeconds: tol.TolerationSeconds,
		})
	}

	// Resource requirements
	res := profile.Resources()
	cpuReq, err := resource.ParseQuantity(res.CPURequest())
	if err != nil {
		return nil, fmt.Errorf("%w: invalid cpu request %q", model.ErrValidation, res.CPURequest())
	}
	cpuLim, err := resource.ParseQuantity(res.CPULimit())
	if err != nil {
		return nil, fmt.Errorf("%w: invalid cpu limit %q", model.ErrValidation, res.CPULimit())
	}
	memReq, err := resource.ParseQuantity(res.MemoryRequest())
	if err != nil {
		return nil, fmt.Errorf("%w: invalid memory request %q", model.ErrValidation, res.MemoryRequest())
	}
	memLim, err := resource.ParseQuantity(res.MemoryLimit())
	if err != nil {
		return nil, fmt.Errorf("%w: invalid memory limit %q", model.ErrValidation, res.MemoryLimit())
	}

	// S3 keys
	var reportKey, logsKey string
	if opts.WorkerIndex != nil {
		reportKey, err = s3.KeyWorkerSummaryReport(run.ID(), *opts.WorkerIndex)
		if err != nil {
			return nil, err
		}
		logsKey, err = s3.KeyWorkerExecutionLogs(run.ID(), *opts.WorkerIndex)
		if err != nil {
			return nil, err
		}
	} else {
		reportKey, err = s3.KeySummaryReport(run.ID())
		if err != nil {
			return nil, err
		}
		logsKey, err = s3.KeyExecutionLogs(run.ID())
		if err != nil {
			return nil, err
		}
	}

	s3EnvVars := g.buildS3EnvVars()

	// Init container env
	initEnvs := append([]corev1.EnvVar{}, s3EnvVars...)
	initEnvs = append(initEnvs,
		corev1.EnvVar{Name: "S3_BINARY_KEY", Value: binaryKey},
		corev1.EnvVar{Name: "S3_CONFIG_KEY", Value: configKey},
		corev1.EnvVar{Name: "SHARED_DIR", Value: "/shared"},
	)

	// Runner container env
	runnerEnvs := append([]corev1.EnvVar{}, s3EnvVars...)
	runnerEnvs = append(runnerEnvs,
		corev1.EnvVar{Name: "VUHIVE_RUN_ID", Value: run.ID()},
		corev1.EnvVar{Name: "S3_REPORT_KEY", Value: reportKey},
		corev1.EnvVar{Name: "S3_LOGS_KEY", Value: logsKey},
	)
	if g.cfg.APICallbackURL != "" {
		runnerEnvs = append(runnerEnvs, corev1.EnvVar{Name: "API_CALLBACK_URL", Value: g.cfg.APICallbackURL})
	}
	if len(opts.EnvVars) > 0 {
		keys := make([]string, 0, len(opts.EnvVars))
		for k := range opts.EnvVars {
			keys = append(keys, k)
		}
		sort.Strings(keys)
		for _, k := range keys {
			runnerEnvs = append(runnerEnvs, corev1.EnvVar{Name: k, Value: opts.EnvVars[k]})
		}
	}


	job := &batchv1.Job{
		ObjectMeta: metav1.ObjectMeta{
			Name:      jobName,
			Namespace: namespace,
			Labels:    labels,
		},
		Spec: batchv1.JobSpec{
			BackoffLimit:            &backoffLimit,
			ActiveDeadlineSeconds:   &activeDeadlineSeconds,
			TTLSecondsAfterFinished: &ttlSecondsAfterFinished,
			Template: corev1.PodTemplateSpec{
				ObjectMeta: metav1.ObjectMeta{
					Labels: labels,
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
					NodeSelector: profile.NodeSelector(),
					Affinity:     k8sAffinity,
					Tolerations:  k8sTolerations,
					Volumes: []corev1.Volume{
						{
							Name: "shared-workspace",
							VolumeSource: corev1.VolumeSource{
								EmptyDir: &corev1.EmptyDirVolumeSource{},
							},
						},
					},
					InitContainers: []corev1.Container{
						{
							Name:            "fetch-artifacts",
							Image:           initImage,
							ImagePullPolicy: corev1.PullIfNotPresent,
							SecurityContext: &corev1.SecurityContext{
								AllowPrivilegeEscalation: &allowPrivilegeEscalation,
								ReadOnlyRootFilesystem:   &readOnlyRootFilesystem,
								Capabilities: &corev1.Capabilities{
									Drop: []corev1.Capability{"ALL"},
								},
							},
							Env: initEnvs,
							VolumeMounts: []corev1.VolumeMount{
								{
									Name:      "shared-workspace",
									MountPath: "/shared",
								},
							},
						},
					},
					Containers: []corev1.Container{
						{
							Name:            "runner",
							Image:           runnerImage,
							ImagePullPolicy: corev1.PullIfNotPresent,
							SecurityContext: &corev1.SecurityContext{
								AllowPrivilegeEscalation: &allowPrivilegeEscalation,
								ReadOnlyRootFilesystem:   &readOnlyRootFilesystem,
								Capabilities: &corev1.Capabilities{
									Drop: []corev1.Capability{"ALL"},
								},
							},
							Command: []string{"/shared/entrypoint.sh"},
							Env:     runnerEnvs,
							Resources: corev1.ResourceRequirements{
								Requests: corev1.ResourceList{
									corev1.ResourceCPU:    cpuReq,
									corev1.ResourceMemory: memReq,
								},
								Limits: corev1.ResourceList{
									corev1.ResourceCPU:    cpuLim,
									corev1.ResourceMemory: memLim,
								},
							},
							VolumeMounts: []corev1.VolumeMount{
								{
									Name:      "shared-workspace",
									MountPath: "/shared",
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

func (g *RunnerJobGenerator) buildS3EnvVars() []corev1.EnvVar {
	var envs []corev1.EnvVar
	if g.cfg.S3Endpoint != "" {
		envs = append(envs, corev1.EnvVar{Name: "S3_ENDPOINT", Value: g.cfg.S3Endpoint})
	}
	if g.cfg.S3Region != "" {
		envs = append(envs, corev1.EnvVar{Name: "S3_REGION", Value: g.cfg.S3Region})
	}
	if g.cfg.S3Bucket != "" {
		envs = append(envs, corev1.EnvVar{Name: "S3_BUCKET", Value: g.cfg.S3Bucket})
	}
	if g.cfg.S3AccessKeyID != "" {
		envs = append(envs, corev1.EnvVar{Name: "S3_ACCESS_KEY_ID", Value: g.cfg.S3AccessKeyID})
	}
	if g.cfg.S3SecretAccessKey != "" {
		envs = append(envs, corev1.EnvVar{Name: "S3_SECRET_ACCESS_KEY", Value: g.cfg.S3SecretAccessKey})
	}
	if g.cfg.S3UsePathStyle {
		envs = append(envs, corev1.EnvVar{Name: "S3_USE_PATH_STYLE", Value: "true"})
	}
	return envs
}

func formatRunnerJobName(runID string, suffix string) string {
	cleaned := strings.ToLower(strings.TrimSpace(runID))
	name := fmt.Sprintf("vuhive-run-%s", cleaned)
	trimmedSuffix := strings.ToLower(strings.TrimSpace(suffix))
	if trimmedSuffix != "" {
		name = fmt.Sprintf("%s-%s", name, trimmedSuffix)
	}
	if len(name) > 63 {
		name = name[:63]
	}
	return strings.TrimRight(name, "-")
}
