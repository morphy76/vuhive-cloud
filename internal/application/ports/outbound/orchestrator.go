package outbound

import (
	"context"

	"github.com/morphy76/vuhive-cloud/internal/domain/model"
)

// RunnerOrchestratorPort defines the driven port for dispatching and managing runner pods on Kubernetes.
type RunnerOrchestratorPort interface {
	DispatchJob(ctx context.Context, run *model.TestRun, profile *model.RunnerProfile) (string, error)
	AbortJob(ctx context.Context, k8sJobName, namespace string) error
}
