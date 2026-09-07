package model_test

import (
	"testing"
	"time"

	"github.com/morphy76/vuhive-cloud/internal/domain/model"
	"github.com/stretchr/testify/assert"
)

func TestClaims_RoleChecking(t *testing.T) {
	claims := model.NewClaims("user-123", "alice", "alice@example.com", []string{model.RoleDeveloper}, []string{"/developers"}, time.Now().Add(time.Hour))

	assert.Equal(t, "user-123", claims.Subject())
	assert.Equal(t, "alice", claims.Username())
	assert.Equal(t, "alice@example.com", claims.Email())
	assert.True(t, claims.HasRole(model.RoleDeveloper))
	assert.False(t, claims.HasRole(model.RoleDeployer))
	assert.False(t, claims.HasRole(model.RoleAdmin))
	// Developers satisfy viewer requirement
	assert.True(t, claims.HasRole(model.RoleViewer))

	assert.True(t, claims.HasAnyRole(model.RoleAdmin, model.RoleDeveloper))
	assert.False(t, claims.HasAnyRole(model.RoleDeployer, model.RoleRunner))

	assert.True(t, claims.InGroup("/developers"))
	assert.False(t, claims.InGroup("/administrators"))
}

func TestClaims_AdminRoleSatisfiesAll(t *testing.T) {
	adminClaims := model.NewClaims("admin-1", "root", "root@example.com", []string{model.RoleAdmin}, []string{"/administrators"}, time.Now().Add(time.Hour))

	assert.True(t, adminClaims.HasRole(model.RoleAdmin))
	assert.True(t, adminClaims.HasRole(model.RoleDeployer))
	assert.True(t, adminClaims.HasRole(model.RoleDeveloper))
	assert.True(t, adminClaims.HasRole(model.RoleViewer))
	assert.True(t, adminClaims.HasRole(model.RoleRunner))
}

func TestClaims_DeployerRoleSatisfiesViewer(t *testing.T) {
	deployerClaims := model.NewClaims("dep-1", "bob", "bob@example.com", []string{model.RoleDeployer}, []string{"/deployers"}, time.Now().Add(time.Hour))

	assert.True(t, deployerClaims.HasRole(model.RoleDeployer))
	assert.True(t, deployerClaims.HasRole(model.RoleViewer))
	assert.False(t, deployerClaims.HasRole(model.RoleDeveloper))
	assert.False(t, deployerClaims.HasRole(model.RoleAdmin))
}

func TestClaims_GroupMapping(t *testing.T) {
	// User with group /administrators without explicit realm role gets admin permissions
	claims := model.NewClaims("grp-1", "charlie", "c@example.com", nil, []string{"/administrators"}, time.Now().Add(time.Hour))
	assert.True(t, claims.HasRole(model.RoleAdmin))
	assert.True(t, claims.HasRole(model.RoleDeployer))

	// User with /developers group
	devClaims := model.NewClaims("grp-2", "dan", "d@example.com", nil, []string{"/developers"}, time.Now().Add(time.Hour))
	assert.True(t, devClaims.HasRole(model.RoleDeveloper))
	assert.True(t, devClaims.HasRole(model.RoleViewer))
	assert.False(t, devClaims.HasRole(model.RoleDeployer))
}

func TestClaims_Expiration(t *testing.T) {
	validClaims := model.NewClaims("v-1", "eve", "eve@example.com", []string{model.RoleViewer}, nil, time.Now().Add(time.Hour))
	assert.False(t, validClaims.IsExpired())

	expiredClaims := model.NewClaims("v-2", "eve", "eve@example.com", []string{model.RoleViewer}, nil, time.Now().Add(-time.Hour))
	assert.True(t, expiredClaims.IsExpired())
}
