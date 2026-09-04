package runner

import "context"

// Initializer defines the contract for initializing runner pod shared storage.
type Initializer interface {
	Init(ctx context.Context, cfg InitConfig) error
}

// Wrapper defines the contract for executing runner binary and managing its lifecycle.
type Wrapper interface {
	Run(ctx context.Context, cfg WrapperConfig, extraArgs []string) (int, error)
}
