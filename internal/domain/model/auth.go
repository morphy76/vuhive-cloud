package model

import (
	"strings"
	"time"
)

// Standard realm and client roles configured in Keycloak realm.
const (
	RoleAdmin     = "vuhive-admin"
	RoleDeployer  = "vuhive-deployer"
	RoleDeveloper = "vuhive-developer"
	RoleViewer    = "vuhive-viewer"
	RoleRunner    = "vuhive-runner"
)

// Standard groups configured in Keycloak realm.
const (
	GroupAdministrators = "/administrators"
	GroupDeployers      = "/deployers"
	GroupDevelopers     = "/developers"
	GroupViewers        = "/viewers"
)

// Claims represents validated identity, roles, and group memberships extracted from an OIDC JWT.
type Claims struct {
	subject   string
	username  string
	email     string
	roles     []string
	groups    []string
	expiresAt time.Time
}

// NewClaims constructs a validated Claims instance.
func NewClaims(subject, username, email string, roles, groups []string, expiresAt time.Time) *Claims {
	return &Claims{
		subject:   strings.TrimSpace(subject),
		username:  strings.TrimSpace(username),
		email:     strings.TrimSpace(email),
		roles:     roles,
		groups:    groups,
		expiresAt: expiresAt,
	}
}

// Subject returns the user or service account unique identifier (sub claim).
func (c *Claims) Subject() string {
	return c.subject
}

// Username returns the preferred username claim.
func (c *Claims) Username() string {
	return c.username
}

// Email returns the email address claim.
func (c *Claims) Email() string {
	return c.email
}

// Roles returns all raw role assignments.
func (c *Claims) Roles() []string {
	cp := make([]string, len(c.roles))
	copy(cp, c.roles)
	return cp
}

// Groups returns all raw group paths.
func (c *Claims) Groups() []string {
	cp := make([]string, len(c.groups))
	copy(cp, c.groups)
	return cp
}

// ExpiresAt returns the token expiration timestamp.
func (c *Claims) ExpiresAt() time.Time {
	return c.expiresAt
}

// IsExpired checks if the token has passed its expiration time.
func (c *Claims) IsExpired() bool {
	if c.expiresAt.IsZero() {
		return false
	}
	return time.Now().After(c.expiresAt)
}

// InGroup returns true if the claims include the specified group path.
func (c *Claims) InGroup(group string) bool {
	norm := strings.TrimSpace(group)
	for _, g := range c.groups {
		if strings.TrimSpace(g) == norm {
			return true
		}
	}
	return false
}

// HasRole returns true if the user possesses the role directly, through composite role inheritance,
// or via assigned group mapping.
func (c *Claims) HasRole(role string) bool {
	target := strings.TrimSpace(role)

	// Direct check
	for _, r := range c.roles {
		if strings.TrimSpace(r) == target {
			return true
		}
	}

	// Administrators group and vuhive-admin role satisfy all roles
	if c.InGroup(GroupAdministrators) || c.hasDirectRole(RoleAdmin) {
		return true
	}

	switch target {
	case RoleViewer:
		// Deployers, developers, and viewers groups/roles all satisfy viewer permissions
		return c.hasDirectRole(RoleDeployer) ||
			c.hasDirectRole(RoleDeveloper) ||
			c.hasDirectRole(RoleViewer) ||
			c.InGroup(GroupDeployers) ||
			c.InGroup(GroupDevelopers) ||
			c.InGroup(GroupViewers)

	case RoleDeployer:
		return c.InGroup(GroupDeployers)

	case RoleDeveloper:
		return c.InGroup(GroupDevelopers)

	case RoleAdmin:
		return c.InGroup(GroupAdministrators)

	default:
		return false
	}
}

// HasAnyRole checks if any of the provided roles is satisfied.
func (c *Claims) HasAnyRole(roles ...string) bool {
	for _, r := range roles {
		if c.HasRole(r) {
			return true
		}
	}
	return false
}

func (c *Claims) hasDirectRole(role string) bool {
	target := strings.TrimSpace(role)
	for _, r := range c.roles {
		if strings.TrimSpace(r) == target {
			return true
		}
	}
	return false
}
