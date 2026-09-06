package model

import (
	"fmt"
	"time"
)

// Default retention TTL constants in days.
const (
	DefaultLogsTTLDays     = 14
	DefaultReportsTTLDays  = 90
	DefaultSourcesTTLDays  = 30
	DefaultBinariesTTLDays = 30
	DefaultRunsTTLDays     = 180
)

// RetentionPolicy defines data retention windows for execution artifacts and historical records.
type RetentionPolicy struct {
	LogsTTLDays     int  `json:"logs_ttl_days"`
	ReportsTTLDays  int  `json:"reports_ttl_days"`
	SourcesTTLDays  int  `json:"sources_ttl_days"`
	BinariesTTLDays int  `json:"binaries_ttl_days"`
	RunsTTLDays     int  `json:"runs_ttl_days"`
	ArchiveOnly     bool `json:"archive_only"`
}

// DefaultRetentionPolicy constructs the global baseline retention policy.
func DefaultRetentionPolicy() RetentionPolicy {
	return RetentionPolicy{
		LogsTTLDays:     DefaultLogsTTLDays,
		ReportsTTLDays:  DefaultReportsTTLDays,
		SourcesTTLDays:  DefaultSourcesTTLDays,
		BinariesTTLDays: DefaultBinariesTTLDays,
		RunsTTLDays:     DefaultRunsTTLDays,
		ArchiveOnly:     false,
	}
}

// LogsDuration returns the LogsTTLDays expressed as time.Duration.
func (p RetentionPolicy) LogsDuration() time.Duration {
	return time.Duration(p.LogsTTLDays) * 24 * time.Hour
}

// ReportsDuration returns the ReportsTTLDays expressed as time.Duration.
func (p RetentionPolicy) ReportsDuration() time.Duration {
	return time.Duration(p.ReportsTTLDays) * 24 * time.Hour
}

// SourcesDuration returns the SourcesTTLDays expressed as time.Duration.
func (p RetentionPolicy) SourcesDuration() time.Duration {
	return time.Duration(p.SourcesTTLDays) * 24 * time.Hour
}

// BinariesDuration returns the BinariesTTLDays expressed as time.Duration.
func (p RetentionPolicy) BinariesDuration() time.Duration {
	return time.Duration(p.BinariesTTLDays) * 24 * time.Hour
}

// RunsDuration returns the RunsTTLDays expressed as time.Duration.
func (p RetentionPolicy) RunsDuration() time.Duration {
	return time.Duration(p.RunsTTLDays) * 24 * time.Hour
}

// Validate checks whether the retention policy contains consistent, non-negative values.
func (p RetentionPolicy) Validate() error {
	if p.LogsTTLDays < 0 {
		return fmt.Errorf("%w: logs_ttl_days cannot be negative", ErrValidation)
	}
	if p.ReportsTTLDays < 0 {
		return fmt.Errorf("%w: reports_ttl_days cannot be negative", ErrValidation)
	}
	if p.SourcesTTLDays < 0 {
		return fmt.Errorf("%w: sources_ttl_days cannot be negative", ErrValidation)
	}
	if p.BinariesTTLDays < 0 {
		return fmt.Errorf("%w: binaries_ttl_days cannot be negative", ErrValidation)
	}
	if p.RunsTTLDays < 0 {
		return fmt.Errorf("%w: runs_ttl_days cannot be negative", ErrValidation)
	}
	if p.RunsTTLDays > 0 && p.ReportsTTLDays > 0 && p.RunsTTLDays < p.ReportsTTLDays {
		return fmt.Errorf("%w: runs_ttl_days (%d) cannot be less than reports_ttl_days (%d)", ErrValidation, p.RunsTTLDays, p.ReportsTTLDays)
	}
	return nil
}

// Merge creates a combined RetentionPolicy where positive or explicitly configured values
// from override supersede base settings.
func (p RetentionPolicy) Merge(override RetentionPolicy) RetentionPolicy {
	res := p
	if override.LogsTTLDays > 0 {
		res.LogsTTLDays = override.LogsTTLDays
	}
	if override.ReportsTTLDays > 0 {
		res.ReportsTTLDays = override.ReportsTTLDays
	}
	if override.SourcesTTLDays > 0 {
		res.SourcesTTLDays = override.SourcesTTLDays
	}
	if override.BinariesTTLDays > 0 {
		res.BinariesTTLDays = override.BinariesTTLDays
	}
	if override.RunsTTLDays > 0 {
		res.RunsTTLDays = override.RunsTTLDays
	}
	if override.ArchiveOnly {
		res.ArchiveOnly = true
	}
	return res
}
