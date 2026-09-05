package outbound

import (
	"context"
	"time"

	"github.com/morphy76/vuhive-cloud/internal/domain/model"
)

// RunnerJobOptions specifies the parameters needed to dispatch a runner Job.
type RunnerJobOptions struct {
	S3BinaryKey    string
	S3ConfigKey    string
	WorkerIndex    *int
	WorkerCount    *int
	BarrierEnabled bool
	BarrierTimeout time.Duration
	EnvVars        map[string]string
	JobNameSuffix  string
}

// RunnerOrchestratorPort defines the driven port for dispatching and managing runner pods on Kubernetes.
type RunnerOrchestratorPort interface {
	DispatchJob(ctx context.Context, run *model.TestRun, profile *model.RunnerProfile, opts RunnerJobOptions) (string, error)
	AbortJob(ctx context.Context, k8sJobName, namespace string) error
}
