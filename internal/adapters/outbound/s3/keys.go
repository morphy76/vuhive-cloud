package s3

import (
	"fmt"
	"strings"

	"github.com/morphy76/vuhive-cloud/internal/domain/model"
)

// KeySourceTarball builds the S3 key for a test suite source tarball.
func KeySourceTarball(suiteID, version string) (string, error) {
	trimmedSuiteID := strings.TrimSpace(suiteID)
	if trimmedSuiteID == "" {
		return "", fmt.Errorf("%w: suiteID cannot be empty", model.ErrValidation)
	}

	trimmedVersion := strings.TrimSpace(version)
	if trimmedVersion == "" {
		return fmt.Sprintf("suites/%s/sources/source.tar.gz", trimmedSuiteID), nil
	}

	return fmt.Sprintf("suites/%s/sources/source-%s.tar.gz", trimmedSuiteID, trimmedVersion), nil
}

// KeyBinaryArtifact builds the S3 key for an executable runner binary artifact.
func KeyBinaryArtifact(suiteID, artifactID string, platform model.Platform) (string, error) {
	trimmedSuiteID := strings.TrimSpace(suiteID)
	if trimmedSuiteID == "" {
		return "", fmt.Errorf("%w: suiteID cannot be empty", model.ErrValidation)
	}

	trimmedArtifactID := strings.TrimSpace(artifactID)
	if trimmedArtifactID == "" {
		return "", fmt.Errorf("%w: artifactID cannot be empty", model.ErrValidation)
	}

	switch platform {
	case model.PlatformLinuxAmd64:
		return fmt.Sprintf("suites/%s/artifacts/%s/linux-amd64/runner", trimmedSuiteID, trimmedArtifactID), nil
	case model.PlatformLinuxArm64:
		return fmt.Sprintf("suites/%s/artifacts/%s/linux-arm64/runner", trimmedSuiteID, trimmedArtifactID), nil
	default:
		return "", model.ErrInvalidPlatform
	}
}

// KeyBuildLogs builds the S3 key for an artifact compilation build log.
func KeyBuildLogs(suiteID, artifactID string) (string, error) {
	trimmedSuiteID := strings.TrimSpace(suiteID)
	if trimmedSuiteID == "" {
		return "", fmt.Errorf("%w: suiteID cannot be empty", model.ErrValidation)
	}

	trimmedArtifactID := strings.TrimSpace(artifactID)
	if trimmedArtifactID == "" {
		return "", fmt.Errorf("%w: artifactID cannot be empty", model.ErrValidation)
	}

	return fmt.Sprintf("suites/%s/artifacts/%s/build.log", trimmedSuiteID, trimmedArtifactID), nil
}

// KeyConfiguration builds the S3 key for a test configuration YAML file.
func KeyConfiguration(suiteID, configID string) (string, error) {
	trimmedSuiteID := strings.TrimSpace(suiteID)
	if trimmedSuiteID == "" {
		return "", fmt.Errorf("%w: suiteID cannot be empty", model.ErrValidation)
	}

	trimmedConfigID := strings.TrimSpace(configID)
	if trimmedConfigID == "" {
		return "", fmt.Errorf("%w: configID cannot be empty", model.ErrValidation)
	}

	return fmt.Sprintf("suites/%s/configs/%s.yaml", trimmedSuiteID, trimmedConfigID), nil
}

// KeyExecutionLogs builds the S3 key for a test run's execution logs.
func KeyExecutionLogs(runID string) (string, error) {
	trimmedRunID := strings.TrimSpace(runID)
	if trimmedRunID == "" {
		return "", fmt.Errorf("%w: runID cannot be empty", model.ErrValidation)
	}

	return fmt.Sprintf("runs/%s/run.log", trimmedRunID), nil
}

// KeySummaryReport builds the S3 key for a test run's summary.json report.
func KeySummaryReport(runID string) (string, error) {
	trimmedRunID := strings.TrimSpace(runID)
	if trimmedRunID == "" {
		return "", fmt.Errorf("%w: runID cannot be empty", model.ErrValidation)
	}

	return fmt.Sprintf("runs/%s/summary.json", trimmedRunID), nil
}
