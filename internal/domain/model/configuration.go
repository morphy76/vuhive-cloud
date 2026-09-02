package model

import (
	"strings"
	"time"

	"github.com/google/uuid"
)

// Configuration is the domain entity representing an attached vuhive.yaml test scenario configuration.
type Configuration struct {
	id          string
	suiteID     string
	name        string
	contentYAML string
	s3ConfigKey string
	isDefault   bool
	createdAt   time.Time
}

// NewConfiguration creates a new Configuration entity.
func NewConfiguration(suiteID, name, contentYAML, s3ConfigKey string, isDefault bool) (*Configuration, error) {
	trimmedSuiteID := strings.TrimSpace(suiteID)
	trimmedName := strings.TrimSpace(name)
	trimmedYAML := strings.TrimSpace(contentYAML)
	trimmedKey := strings.TrimSpace(s3ConfigKey)

	if trimmedSuiteID == "" || trimmedYAML == "" {
		return nil, ErrValidation
	}
	if trimmedName == "" {
		return nil, ErrEmptyName
	}
	if trimmedKey == "" {
		return nil, ErrEmptyS3Key
	}

	return &Configuration{
		id:          uuid.NewString(),
		suiteID:     trimmedSuiteID,
		name:        trimmedName,
		contentYAML: contentYAML,
		s3ConfigKey: trimmedKey,
		isDefault:   isDefault,
		createdAt:   time.Now().UTC(),
	}, nil
}

// NewConfigurationWithID reconstructs a Configuration entity from persistence.
func NewConfigurationWithID(
	id, suiteID, name, contentYAML, s3ConfigKey string,
	isDefault bool,
	createdAt time.Time,
) (*Configuration, error) {
	trimmedSuiteID := strings.TrimSpace(suiteID)
	trimmedName := strings.TrimSpace(name)
	trimmedYAML := strings.TrimSpace(contentYAML)
	trimmedKey := strings.TrimSpace(s3ConfigKey)

	if trimmedSuiteID == "" || trimmedYAML == "" {
		return nil, ErrValidation
	}
	if trimmedName == "" {
		return nil, ErrEmptyName
	}
	if trimmedKey == "" {
		return nil, ErrEmptyS3Key
	}

	return &Configuration{
		id:          id,
		suiteID:     trimmedSuiteID,
		name:        trimmedName,
		contentYAML: contentYAML,
		s3ConfigKey: trimmedKey,
		isDefault:   isDefault,
		createdAt:   createdAt,
	}, nil
}

// ID returns the unique identifier.
func (c *Configuration) ID() string {
	return c.id
}

// EntityID implements the Entity interface.
func (c *Configuration) EntityID() string {
	return c.id
}

// SuiteID returns the parent test suite ID.
func (c *Configuration) SuiteID() string {
	return c.suiteID
}

// Name returns the configuration profile name.
func (c *Configuration) Name() string {
	return c.name
}

// ContentYAML returns the raw YAML content.
func (c *Configuration) ContentYAML() string {
	return c.contentYAML
}

// S3ConfigKey returns the S3 storage key.
func (c *Configuration) S3ConfigKey() string {
	return c.s3ConfigKey
}

// IsDefault returns whether this configuration is default for the suite.
func (c *Configuration) IsDefault() bool {
	return c.isDefault
}

// CreatedAt returns the creation timestamp.
func (c *Configuration) CreatedAt() time.Time {
	return c.createdAt
}

// SetDefault updates the isDefault flag.
func (c *Configuration) SetDefault(isDefault bool) {
	c.isDefault = isDefault
}

// UpdateContent updates the YAML content and storage key.
func (c *Configuration) UpdateContent(contentYAML, s3ConfigKey string) error {
	trimmedYAML := strings.TrimSpace(contentYAML)
	trimmedKey := strings.TrimSpace(s3ConfigKey)

	if trimmedYAML == "" {
		return ErrValidation
	}
	if trimmedKey == "" {
		return ErrEmptyS3Key
	}

	c.contentYAML = contentYAML
	c.s3ConfigKey = trimmedKey
	return nil
}

// Compile-time interface assertion
var _ Entity = (*Configuration)(nil)
