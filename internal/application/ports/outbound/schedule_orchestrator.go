package outbound

import (
	"context"

	"github.com/morphy76/vuhive-cloud/internal/domain/model"
)

// ScheduleOrchestratorPort defines the driven port for managing native Kubernetes CronJobs.
type ScheduleOrchestratorPort interface {
	CreateCronJob(ctx context.Context, schedule *model.Schedule, profile *model.RunnerProfile, opts RunnerJobOptions) (string, error)
	UpdateCronJob(ctx context.Context, schedule *model.Schedule) error
	DeleteCronJob(ctx context.Context, k8sCronJobName, namespace string) error
}
