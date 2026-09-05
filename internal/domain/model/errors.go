package model

import "errors"

var (
	// ErrNotFound indicates a requested entity was not found.
	ErrNotFound = errors.New("resource not found")

	// ErrConflict indicates a resource collision or unique constraint violation.
	ErrConflict = errors.New("resource already exists")

	// ErrInvalidStateTransition indicates an illegal transition between lifecycle states.
	ErrInvalidStateTransition = errors.New("invalid state transition")

	// ErrTerminalState indicates an operation was attempted on an aggregate in a terminal state.
	ErrTerminalState = errors.New("cannot transition from a terminal state")

	// ErrInvalidPlatform indicates an unsupported OS/architecture platform.
	ErrInvalidPlatform = errors.New("unsupported platform: must be linux/amd64 or linux/arm64")

	// ErrInvalidCronExpression indicates a malformed or unsupported cron schedule string.
	ErrInvalidCronExpression = errors.New("invalid cron expression")

	// ErrInvalidResourceQuantity indicates a malformed or illogical CPU/RAM quantity specification.
	ErrInvalidResourceQuantity = errors.New("invalid resource quantity")

	// ErrEmptyName indicates an entity name was empty or whitespace only.
	ErrEmptyName = errors.New("name cannot be empty")

	// ErrInvalidChecksum indicates a sha256 checksum string was invalid.
	ErrInvalidChecksum = errors.New("invalid sha256 checksum: must be 64 hexadecimal characters")

	// ErrEmptyS3Key indicates a required S3 storage key was empty.
	ErrEmptyS3Key = errors.New("s3 key cannot be empty")

	// ErrValidation indicates a generic domain validation rule violation.
	ErrValidation = errors.New("validation failed")

	// ErrTimeout indicates an operation timed out before completion.
	ErrTimeout = errors.New("operation timed out")

	// ErrBuildFailed indicates a build compilation job failed.
	ErrBuildFailed = errors.New("build compilation failed")

	// ErrInvalidAffinity indicates an invalid Kubernetes node affinity configuration.
	ErrInvalidAffinity = errors.New("invalid affinity configuration")

	// ErrInvalidToleration indicates an invalid Kubernetes toleration configuration.
	ErrInvalidToleration = errors.New("invalid toleration configuration")

	// ErrBarrierNotFound indicates the barrier session was not found.
	ErrBarrierNotFound = errors.New("barrier session not found")

	// ErrBarrierAborted indicates the rendezvous was aborted by a worker or coordinator.
	ErrBarrierAborted = errors.New("barrier rendezvous aborted")

	// ErrBarrierTimeout indicates the barrier timed out waiting for all participants.
	ErrBarrierTimeout = errors.New("barrier rendezvous timed out")

	// ErrBarrierReleased indicates the barrier has already released participants.
	ErrBarrierReleased = errors.New("barrier session already released")

	// ErrWorkerAlreadyRegistered indicates a worker ID has already joined the barrier.
	ErrWorkerAlreadyRegistered = errors.New("worker already registered in barrier")

	// ErrInvalidWorkerCount indicates an invalid expected worker count.
	ErrInvalidWorkerCount = errors.New("invalid worker count: must be at least 1")

	// ErrInvalidWorkerIndex indicates worker index is negative or >= worker count.
	ErrInvalidWorkerIndex = errors.New("worker index out of bounds")

	// ErrNoScenariosDefined indicates configuration lacks scenario definitions.
	ErrNoScenariosDefined = errors.New("no scenarios defined in configuration")
)
