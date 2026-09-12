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

	// ErrInvalidDeadline indicates an invalid active deadline seconds value.
	ErrInvalidDeadline = errors.New("active deadline seconds must be greater than zero")

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
	ErrInvalidWorkerCount = errors.New("worker count must be at least 1")

	// ErrInvalidWorkerIndex indicates worker index is negative or >= worker count.
	ErrInvalidWorkerIndex = errors.New("worker index out of bounds")

	// ErrNoScenariosDefined indicates configuration lacks scenario definitions.
	ErrNoScenariosDefined = errors.New("no scenarios defined in configuration")

	// ErrRunInFlight indicates an operation cannot be performed because the test run is still executing.
	ErrRunInFlight = errors.New("test run is still in flight")

	// ErrReportNotFound indicates an execution summary report was not found.
	ErrReportNotFound = errors.New("summary report not found")

	// ErrLogsNotFound indicates test execution logs were not found.
	ErrLogsNotFound = errors.New("execution logs not found")

	// ErrMissingGoMod indicates the uploaded archive does not contain a go.mod file.
	ErrMissingGoMod = errors.New("missing go.mod in source archive")

	// ErrMissingVuhiveDependency indicates go.mod does not declare github.com/morphy76/vuhive as a direct dependency.
	ErrMissingVuhiveDependency = errors.New("go.mod must declare github.com/morphy76/vuhive as a direct dependency")

	// ErrForbiddenImport indicates source code imports a disallowed or suspicious package.
	ErrForbiddenImport = errors.New("source code imports disallowed package")

	// ErrForbiddenPackageMain indicates user code illegally declared package main or func main().
	ErrForbiddenPackageMain = errors.New("user source code must declare package scenario and must not define package main or func main()")

	// ErrMissingScenarioContract indicates the source package fails to implement a valid vuhive.Scenario contract.
	ErrMissingScenarioContract = errors.New("source package does not declare or implement a valid vuhive.Scenario contract")

	// ErrInvalidArchive indicates the source package archive could not be read or is corrupted.
	ErrInvalidArchive = errors.New("source package is not a valid tar.gz archive")

	// ErrInsecureOverrideForbidden indicates that overriding the import blocklist is not permitted by cluster policy.
	ErrInsecureOverrideForbidden = errors.New("import blocklist override is forbidden by cluster policy")

	// ErrUnauthorized indicates missing, expired, or invalid authentication credentials.
	ErrUnauthorized = errors.New("unauthorized: authentication required")

	// ErrForbidden indicates authenticated caller lacks required role or permission.
	ErrForbidden = errors.New("forbidden: insufficient permissions")

	// ErrTokenExpired indicates the authentication token has expired.
	ErrTokenExpired = errors.New("token expired")

	// ErrInvalidToken indicates the token format or signature is invalid.
	ErrInvalidToken = errors.New("invalid token")

	// ErrUnsupportedGoVersion indicates a Go version older than 1.26 or invalid.
	ErrUnsupportedGoVersion = errors.New("unsupported go version: vuhive requires Go >= 1.26")
)

