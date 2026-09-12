package model

import (
	"fmt"
	"strconv"
	"strings"
)

const (
	// MinSupportedGoMajor is the minimum major Go version supported by vuhive.
	MinSupportedGoMajor = 1
	// MinSupportedGoMinor is the minimum minor Go version supported by vuhive (1.26).
	MinSupportedGoMinor = 26
)

// ValidateGoVersion validates and normalizes a Go version string.
// If the version is empty or only whitespace, it returns empty string without error (using cluster default).
// If a version is provided, it verifies that the version is >= 1.26.
// Strips leading "go" prefix if present (e.g., "go1.26" -> "1.26").
func ValidateGoVersion(raw string) (string, error) {
	trimmed := strings.TrimSpace(raw)
	if trimmed == "" {
		return "", nil
	}

	clean := strings.TrimPrefix(trimmed, "go")
	parts := strings.Split(clean, ".")
	if len(parts) < 2 {
		return "", fmt.Errorf("%w: invalid format %q", ErrUnsupportedGoVersion, raw)
	}

	major, err := strconv.Atoi(parts[0])
	if err != nil {
		return "", fmt.Errorf("%w: invalid major version %q: %v", ErrUnsupportedGoVersion, parts[0], err)
	}

	minor, err := strconv.Atoi(parts[1])
	if err != nil {
		return "", fmt.Errorf("%w: invalid minor version %q: %v", ErrUnsupportedGoVersion, parts[1], err)
	}

	if major < MinSupportedGoMajor || (major == MinSupportedGoMajor && minor < MinSupportedGoMinor) {
		return "", fmt.Errorf("%w: Go %s is below minimum required Go 1.26", ErrUnsupportedGoVersion, clean)
	}

	return clean, nil
}
