package model

import (
	"regexp"
	"strings"
	"time"

	"github.com/google/uuid"
)

// secretKeyPattern validates that a secret key consists of uppercase letters, digits, and underscores,
// and starts with an uppercase letter.
var secretKeyPattern = regexp.MustCompile(`^[A-Z][A-Z0-9_]*$`)

// Secret is the domain entity representing an encrypted secret scoped to a test suite.
type Secret struct {
	id             string
	suiteID        string
	key            string
	encryptedValue []byte
	createdAt      time.Time
	updatedAt      time.Time
}

// ValidateSecretKey checks that a key follows the naming convention: uppercase letters, digits,
// and underscores, starting with an uppercase letter.
func ValidateSecretKey(key string) error {
	trimmed := strings.TrimSpace(key)
	if trimmed == "" || !secretKeyPattern.MatchString(trimmed) {
		return ErrInvalidSecretKey
	}
	return nil
}

// NewSecret creates a new Secret entity with a generated UUID.
func NewSecret(suiteID, key string, encryptedValue []byte) (*Secret, error) {
	trimmedSuiteID := strings.TrimSpace(suiteID)
	trimmedKey := strings.TrimSpace(key)

	if trimmedSuiteID == "" {
		return nil, ErrValidation
	}
	if err := ValidateSecretKey(trimmedKey); err != nil {
		return nil, err
	}
	if len(encryptedValue) == 0 {
		return nil, ErrValidation
	}

	now := time.Now().UTC()
	return &Secret{
		id:             uuid.NewString(),
		suiteID:        trimmedSuiteID,
		key:            trimmedKey,
		encryptedValue: encryptedValue,
		createdAt:      now,
		updatedAt:      now,
	}, nil
}

// NewSecretWithID reconstructs a Secret entity from persistence.
func NewSecretWithID(id, suiteID, key string, encryptedValue []byte, createdAt, updatedAt time.Time) (*Secret, error) {
	trimmedSuiteID := strings.TrimSpace(suiteID)
	trimmedKey := strings.TrimSpace(key)

	if trimmedSuiteID == "" {
		return nil, ErrValidation
	}
	if err := ValidateSecretKey(trimmedKey); err != nil {
		return nil, err
	}
	if len(encryptedValue) == 0 {
		return nil, ErrValidation
	}

	return &Secret{
		id:             id,
		suiteID:        trimmedSuiteID,
		key:            trimmedKey,
		encryptedValue: encryptedValue,
		createdAt:      createdAt,
		updatedAt:      updatedAt,
	}, nil
}

// ID returns the unique identifier.
func (s *Secret) ID() string {
	return s.id
}

// EntityID implements the Entity interface.
func (s *Secret) EntityID() string {
	return s.id
}

// SuiteID returns the parent test suite ID.
func (s *Secret) SuiteID() string {
	return s.suiteID
}

// Key returns the secret key name.
func (s *Secret) Key() string {
	return s.key
}

// EncryptedValue returns the encrypted secret value.
func (s *Secret) EncryptedValue() []byte {
	return s.encryptedValue
}

// CreatedAt returns the creation timestamp.
func (s *Secret) CreatedAt() time.Time {
	return s.createdAt
}

// UpdatedAt returns the last update timestamp.
func (s *Secret) UpdatedAt() time.Time {
	return s.updatedAt
}

// UpdateValue replaces the encrypted value and bumps the updatedAt timestamp.
func (s *Secret) UpdateValue(encryptedValue []byte) error {
	if len(encryptedValue) == 0 {
		return ErrValidation
	}
	s.encryptedValue = encryptedValue
	s.updatedAt = time.Now().UTC()
	return nil
}

// Compile-time interface assertion
var _ Entity = (*Secret)(nil)
