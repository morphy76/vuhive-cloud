package model

import (
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/robfig/cron/v3"
)

var cronParser = cron.NewParser(
	cron.Minute | cron.Hour | cron.Dom | cron.Month | cron.Dow | cron.Descriptor,
)

// ValidateCronExpression validates whether a string is a standard 5-part cron schedule or standard descriptor.
func ValidateCronExpression(expr string) error {
	trimmed := strings.TrimSpace(expr)
	if trimmed == "" {
		return ErrInvalidCronExpression
	}
	_, err := cronParser.Parse(trimmed)
	if err != nil {
		return fmt.Errorf("%w: %v", ErrInvalidCronExpression, err)
	}
	return nil
}

// Schedule is the domain aggregate root representing a recurring Kubernetes CronJob schedule.
type Schedule struct {
	id              string
	suiteID         string
	artifactID      string
	configurationID *string
	runnerProfileID string
	name            string
	cronExpression  string
	k8sCronJobName  string
	isActive        bool
	createdAt       time.Time
	updatedAt       time.Time
}

// NewSchedule creates a new active Schedule aggregate.
func NewSchedule(
	suiteID, artifactID string,
	configurationID *string,
	runnerProfileID, name, cronExpression string,
) (*Schedule, error) {
	trimmedSuiteID := strings.TrimSpace(suiteID)
	trimmedArtifactID := strings.TrimSpace(artifactID)
	trimmedProfileID := strings.TrimSpace(runnerProfileID)
	trimmedName := strings.TrimSpace(name)

	if trimmedSuiteID == "" || trimmedArtifactID == "" || trimmedProfileID == "" {
		return nil, ErrValidation
	}
	if trimmedName == "" {
		return nil, ErrEmptyName
	}
	if err := ValidateCronExpression(cronExpression); err != nil {
		return nil, err
	}

	var cfgID *string
	if configurationID != nil && strings.TrimSpace(*configurationID) != "" {
		c := strings.TrimSpace(*configurationID)
		cfgID = &c
	}

	id := uuid.NewString()
	k8sName := fmt.Sprintf("vuhive-sched-%s", id[:8])
	now := time.Now().UTC()

	return &Schedule{
		id:              id,
		suiteID:         trimmedSuiteID,
		artifactID:      trimmedArtifactID,
		configurationID: cfgID,
		runnerProfileID: trimmedProfileID,
		name:            trimmedName,
		cronExpression:  strings.TrimSpace(cronExpression),
		k8sCronJobName:  k8sName,
		isActive:        true,
		createdAt:       now,
		updatedAt:       now,
	}, nil
}

// NewScheduleWithID reconstructs a Schedule aggregate from persistence.
func NewScheduleWithID(
	id, suiteID, artifactID string,
	configurationID *string,
	runnerProfileID, name, cronExpression, k8sCronJobName string,
	isActive bool,
	createdAt, updatedAt time.Time,
) (*Schedule, error) {
	if strings.TrimSpace(suiteID) == "" || strings.TrimSpace(artifactID) == "" || strings.TrimSpace(runnerProfileID) == "" {
		return nil, ErrValidation
	}
	if strings.TrimSpace(name) == "" {
		return nil, ErrEmptyName
	}
	if err := ValidateCronExpression(cronExpression); err != nil {
		return nil, err
	}

	return &Schedule{
		id:              id,
		suiteID:         suiteID,
		artifactID:      artifactID,
		configurationID: configurationID,
		runnerProfileID: runnerProfileID,
		name:            name,
		cronExpression:  cronExpression,
		k8sCronJobName:  k8sCronJobName,
		isActive:        isActive,
		createdAt:       createdAt,
		updatedAt:       updatedAt,
	}, nil
}

// ID returns the unique identifier.
func (s *Schedule) ID() string {
	return s.id
}

// EntityID implements the Entity interface.
func (s *Schedule) EntityID() string {
	return s.id
}

// AggregateType implements the AggregateRoot interface.
func (s *Schedule) AggregateType() string {
	return "Schedule"
}

// SuiteID returns the target test suite ID.
func (s *Schedule) SuiteID() string {
	return s.suiteID
}

// ArtifactID returns the target binary artifact ID.
func (s *Schedule) ArtifactID() string {
	return s.artifactID
}

// ConfigurationID returns the attached configuration ID, if any.
func (s *Schedule) ConfigurationID() *string {
	return s.configurationID
}

// RunnerProfileID returns the runner profile ID.
func (s *Schedule) RunnerProfileID() string {
	return s.runnerProfileID
}

// Name returns the schedule name.
func (s *Schedule) Name() string {
	return s.name
}

// CronExpression returns the cron expression.
func (s *Schedule) CronExpression() string {
	return s.cronExpression
}

// K8sCronJobName returns the associated Kubernetes CronJob resource name.
func (s *Schedule) K8sCronJobName() string {
	return s.k8sCronJobName
}

// IsActive returns whether the schedule is currently enabled.
func (s *Schedule) IsActive() bool {
	return s.isActive
}

// CreatedAt returns when the schedule was created.
func (s *Schedule) CreatedAt() time.Time {
	return s.createdAt
}

// UpdatedAt returns when the schedule was last updated.
func (s *Schedule) UpdatedAt() time.Time {
	return s.updatedAt
}

// Activate enables the schedule.
func (s *Schedule) Activate() error {
	if s.isActive {
		return ErrInvalidStateTransition
	}
	s.isActive = true
	s.updatedAt = time.Now().UTC()
	return nil
}

// Deactivate disables the schedule.
func (s *Schedule) Deactivate() error {
	if !s.isActive {
		return ErrInvalidStateTransition
	}
	s.isActive = false
	s.updatedAt = time.Now().UTC()
	return nil
}

// UpdateCronExpression updates the cron expression after validation.
func (s *Schedule) UpdateCronExpression(expr string) error {
	if err := ValidateCronExpression(expr); err != nil {
		return err
	}
	s.cronExpression = strings.TrimSpace(expr)
	s.updatedAt = time.Now().UTC()
	return nil
}

// Compile-time interface assertions
var (
	_ Entity        = (*Schedule)(nil)
	_ AggregateRoot = (*Schedule)(nil)
)
