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

	// ErrInvalidWorkerCount indicates worker count must be >= 1.
	ErrInvalidWorkerCount = errors.New("worker count must be at least 1")

	// ErrInvalidWorkerIndex indicates worker index is negative or >= worker count.
	ErrInvalidWorkerIndex = errors.New("worker index out of bounds")

	// ErrNoScenariosDefined indicates configuration lacks scenario definitions.
	ErrNoScenariosDefined = errors.New("no scenarios defined in configuration")
)
