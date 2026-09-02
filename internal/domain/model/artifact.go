package model

import (
	"strings"
	"time"

	"github.com/google/uuid"
)

// Platform represents the target operating system and architecture for compiled test binaries.
type Platform string

const (
	PlatformLinuxAmd64 Platform = "linux/amd64"
	PlatformLinuxArm64 Platform = "linux/arm64"
)

// IsValid checks whether the platform is supported.
func (p Platform) IsValid() bool {
	switch p {
	case PlatformLinuxAmd64, PlatformLinuxArm64:
		return true
	default:
		return false
	}
}

// ParsePlatform parses and validates a platform string.
func ParsePlatform(s string) (Platform, error) {
	p := Platform(strings.TrimSpace(s))
	if !p.IsValid() {
		return "", ErrInvalidPlatform
	}
	return p, nil
}

// ArtifactStatus represents the compilation lifecycle state of a binary artifact.
type ArtifactStatus string

const (
	ArtifactStatusPending  ArtifactStatus = "PENDING"
	ArtifactStatusBuilding ArtifactStatus = "BUILDING"
	ArtifactStatusReady    ArtifactStatus = "READY"
	ArtifactStatusFailed   ArtifactStatus = "FAILED"
)

// IsValid checks whether the ArtifactStatus is recognized.
func (s ArtifactStatus) IsValid() bool {
	switch s {
	case ArtifactStatusPending, ArtifactStatusBuilding, ArtifactStatusReady, ArtifactStatusFailed:
		return true
	default:
		return false
	}
}

// Artifact is the domain entity representing a compiled multi-arch test binary artifact.
type Artifact struct {
	id              string
	suiteID         string
	platform        Platform
	s3BinaryKey     string
	sha256Checksum  string
	buildLogsS3Key  string
	status          ArtifactStatus
	errorMessage    string
	createdAt       time.Time
}

// NewArtifact creates a new Artifact entity in PENDING status.
func NewArtifact(suiteID string, platform Platform) (*Artifact, error) {
	trimmedSuiteID := strings.TrimSpace(suiteID)
	if trimmedSuiteID == "" {
		return nil, ErrValidation
	}
	if !platform.IsValid() {
		return nil, ErrInvalidPlatform
	}

	return &Artifact{
		id:        uuid.NewString(),
		suiteID:   trimmedSuiteID,
		platform:  platform,
		status:    ArtifactStatusPending,
		createdAt: time.Now().UTC(),
	}, nil
}

// NewArtifactWithID reconstructs an Artifact entity from persistence.
func NewArtifactWithID(
	id, suiteID string,
	platform Platform,
	s3BinaryKey, sha256Checksum, buildLogsS3Key string,
	status ArtifactStatus,
	errorMessage string,
	createdAt time.Time,
) (*Artifact, error) {
	trimmedSuiteID := strings.TrimSpace(suiteID)
	if trimmedSuiteID == "" {
		return nil, ErrValidation
	}
	if !platform.IsValid() {
		return nil, ErrInvalidPlatform
	}
	if !status.IsValid() {
		return nil, ErrInvalidStateTransition
	}

	return &Artifact{
		id:             id,
		suiteID:        trimmedSuiteID,
		platform:       platform,
		s3BinaryKey:    s3BinaryKey,
		sha256Checksum: sha256Checksum,
		buildLogsS3Key: buildLogsS3Key,
		status:         status,
		errorMessage:   errorMessage,
		createdAt:      createdAt,
	}, nil
}

// ID returns the unique identifier of the artifact.
func (a *Artifact) ID() string {
	return a.id
}

// EntityID implements the Entity interface.
func (a *Artifact) EntityID() string {
	return a.id
}

// SuiteID returns the ID of the parent test suite.
func (a *Artifact) SuiteID() string {
	return a.suiteID
}

// Platform returns the compilation platform.
func (a *Artifact) Platform() Platform {
	return a.platform
}

// S3BinaryKey returns the S3 object storage key where the binary is stored.
func (a *Artifact) S3BinaryKey() string {
	return a.s3BinaryKey
}

// SHA256Checksum returns the SHA-256 hash of the compiled binary.
func (a *Artifact) SHA256Checksum() string {
	return a.sha256Checksum
}

// BuildLogsS3Key returns the S3 key where the build logs are stored.
func (a *Artifact) BuildLogsS3Key() string {
	return a.buildLogsS3Key
}

// Status returns the current compilation lifecycle status.
func (a *Artifact) Status() ArtifactStatus {
	return a.status
}

// ErrorMessage returns the failure error message, if any.
func (a *Artifact) ErrorMessage() string {
	return a.errorMessage
}

// CreatedAt returns the timestamp when the artifact was created.
func (a *Artifact) CreatedAt() time.Time {
	return a.createdAt
}

// MarkBuilding transitions the artifact from PENDING to BUILDING.
func (a *Artifact) MarkBuilding() error {
	if a.status == ArtifactStatusReady || a.status == ArtifactStatusFailed {
		return ErrTerminalState
	}
	if a.status != ArtifactStatusPending {
		return ErrInvalidStateTransition
	}
	a.status = ArtifactStatusBuilding
	return nil
}

// MarkReady transitions the artifact to READY upon successful compilation and upload.
func (a *Artifact) MarkReady(s3BinaryKey, sha256Checksum string) error {
	if a.status == ArtifactStatusReady || a.status == ArtifactStatusFailed {
		return ErrTerminalState
	}
	if a.status != ArtifactStatusBuilding {
		return ErrInvalidStateTransition
	}
	trimmedKey := strings.TrimSpace(s3BinaryKey)
	if trimmedKey == "" {
		return ErrEmptyS3Key
	}
	trimmedChecksum := strings.TrimSpace(sha256Checksum)
	if !isValidSHA256(trimmedChecksum) {
		return ErrInvalidChecksum
	}

	a.s3BinaryKey = trimmedKey
	a.sha256Checksum = trimmedChecksum
	a.status = ArtifactStatusReady
	return nil
}

// MarkFailed transitions the artifact to FAILED if build or static checks fail.
func (a *Artifact) MarkFailed(errorMessage, buildLogsS3Key string) error {
	if a.status == ArtifactStatusReady || a.status == ArtifactStatusFailed {
		return ErrTerminalState
	}
	a.errorMessage = errorMessage
	a.buildLogsS3Key = buildLogsS3Key
	a.status = ArtifactStatusFailed
	return nil
}

func isValidSHA256(s string) bool {
	if len(s) != 64 {
		return false
	}
	for i := 0; i < len(s); i++ {
		c := s[i]
		if !((c >= '0' && c <= '9') || (c >= 'a' && c <= 'f') || (c >= 'A' && c <= 'F')) {
			return false
		}
	}
	return true
}

// Compile-time interface assertion
var _ Entity = (*Artifact)(nil)
