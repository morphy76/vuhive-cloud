package model

import (
	"strings"
	"time"

	"github.com/google/uuid"
)

// TestSuiteState represents the lifecycle state of a TestSuite aggregate.
type TestSuiteState string

const (
	TestSuiteStateDraft    TestSuiteState = "DRAFT"
	TestSuiteStateActive   TestSuiteState = "ACTIVE"
	TestSuiteStateArchived TestSuiteState = "ARCHIVED"
)

// IsValid checks if the TestSuiteState is a recognized state.
func (s TestSuiteState) IsValid() bool {
	switch s {
	case TestSuiteStateDraft, TestSuiteStateActive, TestSuiteStateArchived:
		return true
	default:
		return false
	}
}

// TestSuite is the domain aggregate root representing a managed load test suite.
type TestSuite struct {
	id          string
	name        string
	description string
	state       TestSuiteState
	createdAt   time.Time
	updatedAt   time.Time
}

// NewTestSuite creates a new TestSuite aggregate in DRAFT state.
func NewTestSuite(name, description string) (*TestSuite, error) {
	trimmedName := strings.TrimSpace(name)
	if trimmedName == "" {
		return nil, ErrEmptyName
	}

	now := time.Now().UTC()
	return &TestSuite{
		id:          uuid.NewString(),
		name:        trimmedName,
		description: strings.TrimSpace(description),
		state:       TestSuiteStateDraft,
		createdAt:   now,
		updatedAt:   now,
	}, nil
}

// NewTestSuiteWithID reconstructs a TestSuite aggregate from persistence.
func NewTestSuiteWithID(id, name, description string, state TestSuiteState, createdAt, updatedAt time.Time) (*TestSuite, error) {
	trimmedName := strings.TrimSpace(name)
	if trimmedName == "" {
		return nil, ErrEmptyName
	}
	if !state.IsValid() {
		return nil, ErrInvalidStateTransition
	}

	return &TestSuite{
		id:          id,
		name:        trimmedName,
		description: description,
		state:       state,
		createdAt:   createdAt,
		updatedAt:   updatedAt,
	}, nil
}

// ID returns the unique identifier of the test suite.
func (s *TestSuite) ID() string {
	return s.id
}

// EntityID implements the Entity interface.
func (s *TestSuite) EntityID() string {
	return s.id
}

// AggregateType implements the AggregateRoot interface.
func (s *TestSuite) AggregateType() string {
	return "TestSuite"
}

// Name returns the name of the test suite.
func (s *TestSuite) Name() string {
	return s.name
}

// Description returns the description of the test suite.
func (s *TestSuite) Description() string {
	return s.description
}

// State returns the current lifecycle state of the test suite.
func (s *TestSuite) State() TestSuiteState {
	return s.state
}

// CreatedAt returns the timestamp when the suite was created.
func (s *TestSuite) CreatedAt() time.Time {
	return s.createdAt
}

// UpdatedAt returns the timestamp when the suite was last updated.
func (s *TestSuite) UpdatedAt() time.Time {
	return s.updatedAt
}

// Activate transitions the test suite into the ACTIVE state.
func (s *TestSuite) Activate() error {
	if s.state == TestSuiteStateActive {
		return ErrInvalidStateTransition
	}
	s.state = TestSuiteStateActive
	s.updatedAt = time.Now().UTC()
	return nil
}

// Archive transitions the test suite into the ARCHIVED state.
func (s *TestSuite) Archive() error {
	if s.state == TestSuiteStateArchived {
		return ErrInvalidStateTransition
	}
	s.state = TestSuiteStateArchived
	s.updatedAt = time.Now().UTC()
	return nil
}

// UpdateDetails updates the name and description of the test suite.
func (s *TestSuite) UpdateDetails(name, description string) error {
	if s.state == TestSuiteStateArchived {
		return ErrInvalidStateTransition
	}
	trimmedName := strings.TrimSpace(name)
	if trimmedName == "" {
		return ErrEmptyName
	}
	s.name = trimmedName
	s.description = strings.TrimSpace(description)
	s.updatedAt = time.Now().UTC()
	return nil
}

// Compile-time interface assertion
var (
	_ Entity        = (*TestSuite)(nil)
	_ AggregateRoot = (*TestSuite)(nil)
)
