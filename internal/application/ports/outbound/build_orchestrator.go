package outbound

import (
	"context"
	"io"

	"github.com/morphy76/vuhive-cloud/internal/domain/model"
)

// BuildJobOptions specifies the parameters needed to generate and dispatch an isolated compilation Job.
type BuildJobOptions struct {
	SuiteID         string
	ArtifactID      string
	Platform        model.Platform
	SourceURL       string
	BinaryUploadURL string
}

// BuildJobExecution encapsulates the result of a finished build Job execution.
type BuildJobExecution struct {
	JobName        string
	ExitCode       int
	SHA256Checksum string
	Logs           io.ReadCloser
}

// BuildOrchestratorPort defines the driven port for dispatching and managing compilation Jobs on Kubernetes.
type BuildOrchestratorPort interface {
	DispatchBuildJob(ctx context.Context, opts BuildJobOptions) (string, error)
	StreamJobLogs(ctx context.Context, jobName string) (io.ReadCloser, error)
	WaitForJob(ctx context.Context, jobName string) (*BuildJobExecution, error)
	DeleteJob(ctx context.Context, jobName string) error
}
