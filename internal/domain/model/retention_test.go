package model_test

import (
	"testing"
	"time"

	"github.com/morphy76/vuhive-cloud/internal/domain/model"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestDefaultRetentionPolicy(t *testing.T) {
	p := model.DefaultRetentionPolicy()
	assert.Equal(t, 14, p.LogsTTLDays)
	assert.Equal(t, 90, p.ReportsTTLDays)
	assert.Equal(t, 30, p.SourcesTTLDays)
	assert.Equal(t, 30, p.BinariesTTLDays)
	assert.Equal(t, 180, p.RunsTTLDays)
	assert.False(t, p.ArchiveOnly)

	assert.Equal(t, 14*24*time.Hour, p.LogsDuration())
	assert.Equal(t, 90*24*time.Hour, p.ReportsDuration())
	assert.Equal(t, 30*24*time.Hour, p.SourcesDuration())
	assert.Equal(t, 30*24*time.Hour, p.BinariesDuration())
	assert.Equal(t, 180*24*time.Hour, p.RunsDuration())

	require.NoError(t, p.Validate())
}

func TestRetentionPolicy_Validate(t *testing.T) {
	tests := []struct {
		name      string
		policy    model.RetentionPolicy
		wantError bool
	}{
		{
			name: "valid custom policy",
			policy: model.RetentionPolicy{
				LogsTTLDays:     7,
				ReportsTTLDays:  30,
				SourcesTTLDays:  15,
				BinariesTTLDays: 15,
				RunsTTLDays:     60,
			},
			wantError: false,
		},
		{
			name: "negative logs ttl",
			policy: model.RetentionPolicy{
				LogsTTLDays:     -1,
				ReportsTTLDays:  30,
				SourcesTTLDays:  15,
				BinariesTTLDays: 15,
				RunsTTLDays:     60,
			},
			wantError: true,
		},
		{
			name: "negative reports ttl",
			policy: model.RetentionPolicy{
				LogsTTLDays:     7,
				ReportsTTLDays:  -5,
				SourcesTTLDays:  15,
				BinariesTTLDays: 15,
				RunsTTLDays:     60,
			},
			wantError: true,
		},
		{
			name: "negative sources ttl",
			policy: model.RetentionPolicy{
				LogsTTLDays:     7,
				ReportsTTLDays:  30,
				SourcesTTLDays:  -2,
				BinariesTTLDays: 15,
				RunsTTLDays:     60,
			},
			wantError: true,
		},
		{
			name: "negative binaries ttl",
			policy: model.RetentionPolicy{
				LogsTTLDays:     7,
				ReportsTTLDays:  30,
				SourcesTTLDays:  15,
				BinariesTTLDays: -1,
				RunsTTLDays:     60,
			},
			wantError: true,
		},
		{
			name: "negative runs ttl",
			policy: model.RetentionPolicy{
				LogsTTLDays:     7,
				ReportsTTLDays:  30,
				SourcesTTLDays:  15,
				BinariesTTLDays: 15,
				RunsTTLDays:     -10,
			},
			wantError: true,
		},
		{
			name: "runs ttl less than reports ttl",
			policy: model.RetentionPolicy{
				LogsTTLDays:     7,
				ReportsTTLDays:  90,
				SourcesTTLDays:  15,
				BinariesTTLDays: 15,
				RunsTTLDays:     30, // Reports retained longer than runs itself
			},
			wantError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.policy.Validate()
			if tt.wantError {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestRetentionPolicy_Merge(t *testing.T) {
	base := model.DefaultRetentionPolicy()
	override := model.RetentionPolicy{
		LogsTTLDays: 3,
		RunsTTLDays: 365,
		ArchiveOnly: true,
	}

	merged := base.Merge(override)
	assert.Equal(t, 3, merged.LogsTTLDays)
	assert.Equal(t, 90, merged.ReportsTTLDays)   // kept from base
	assert.Equal(t, 30, merged.SourcesTTLDays)   // kept from base
	assert.Equal(t, 30, merged.BinariesTTLDays)  // kept from base
	assert.Equal(t, 365, merged.RunsTTLDays)     // overridden
	assert.True(t, merged.ArchiveOnly)           // overridden
}
