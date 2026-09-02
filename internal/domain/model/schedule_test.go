package model_test

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/morphy76/vuhive-cloud/internal/domain/model"
)

func TestValidateCronExpression(t *testing.T) {
	t.Run("valid standard 5-part cron expressions", func(t *testing.T) {
		validExprs := []string{
			"0 2 * * *",       // 2:00 AM every day
			"*/15 * * * *",    // every 15 minutes
			"0 0 1 * *",       // 1st of every month
			"30 4 * * 1-5",    // 4:30 AM weekdays
			"@hourly",         // standard descriptors
			"@daily",
		}
		for _, expr := range validExprs {
			err := model.ValidateCronExpression(expr)
			assert.NoError(t, err, "expected valid: %s", expr)
		}
	})

	t.Run("invalid cron expressions", func(t *testing.T) {
		invalidExprs := []string{
			"",
			"invalid",
			"* * *",           // too few fields
			"60 * * * *",      // invalid minute (0-59)
			"* 25 * * *",      // invalid hour (0-23)
			"* * 32 * *",      // invalid day of month (1-31)
			"* * * 13 *",      // invalid month (1-12)
			"* * * * 8",       // invalid day of week (0-7)
			"non-existent-descriptor",
		}
		for _, expr := range invalidExprs {
			err := model.ValidateCronExpression(expr)
			assert.ErrorIs(t, err, model.ErrInvalidCronExpression, "expected invalid: %s", expr)
		}
	})
}

func TestNewSchedule(t *testing.T) {
	t.Run("successfully create a new active schedule", func(t *testing.T) {
		sched, err := model.NewSchedule(
			"suite-1", "art-1", nil, "prof-1",
			"nightly-run", "0 2 * * *",
		)
		require.NoError(t, err)
		require.NotNil(t, sched)

		assert.NotEmpty(t, sched.ID())
		assert.Equal(t, sched.ID(), sched.EntityID())
		assert.Equal(t, "Schedule", sched.AggregateType())
		assert.Equal(t, "suite-1", sched.SuiteID())
		assert.Equal(t, "art-1", sched.ArtifactID())
		assert.Nil(t, sched.ConfigurationID())
		assert.Equal(t, "prof-1", sched.RunnerProfileID())
		assert.Equal(t, "nightly-run", sched.Name())
		assert.Equal(t, "0 2 * * *", sched.CronExpression())
		assert.NotEmpty(t, sched.K8sCronJobName())
		assert.True(t, sched.IsActive())
		assert.False(t, sched.CreatedAt().IsZero())
	})

	t.Run("fail with invalid cron expression", func(t *testing.T) {
		_, err := model.NewSchedule(
			"suite-1", "art-1", nil, "prof-1",
			"bad-cron", "invalid-cron",
		)
		assert.ErrorIs(t, err, model.ErrInvalidCronExpression)
	})

	t.Run("fail with empty name or missing IDs", func(t *testing.T) {
		_, err := model.NewSchedule("", "art-1", nil, "prof-1", "name", "0 2 * * *")
		assert.ErrorIs(t, err, model.ErrValidation)

		_, err = model.NewSchedule("suite-1", "", nil, "prof-1", "name", "0 2 * * *")
		assert.ErrorIs(t, err, model.ErrValidation)

		_, err = model.NewSchedule("suite-1", "art-1", nil, "", "name", "0 2 * * *")
		assert.ErrorIs(t, err, model.ErrValidation)

		_, err = model.NewSchedule("suite-1", "art-1", nil, "prof-1", "", "0 2 * * *")
		assert.ErrorIs(t, err, model.ErrEmptyName)
	})
}

func TestSchedule_Lifecycle(t *testing.T) {
	sched, err := model.NewSchedule("suite-1", "art-1", nil, "prof-1", "nightly", "0 2 * * *")
	require.NoError(t, err)
	assert.True(t, sched.IsActive())

	t.Run("deactivate active schedule", func(t *testing.T) {
		err := sched.Deactivate()
		require.NoError(t, err)
		assert.False(t, sched.IsActive())
	})

	t.Run("fail deactivating already inactive schedule", func(t *testing.T) {
		err := sched.Deactivate()
		assert.ErrorIs(t, err, model.ErrInvalidStateTransition)
	})

	t.Run("activate inactive schedule", func(t *testing.T) {
		err := sched.Activate()
		require.NoError(t, err)
		assert.True(t, sched.IsActive())
	})

	t.Run("fail activating already active schedule", func(t *testing.T) {
		err := sched.Activate()
		assert.ErrorIs(t, err, model.ErrInvalidStateTransition)
	})

	t.Run("update cron expression successfully", func(t *testing.T) {
		err := sched.UpdateCronExpression("0 4 * * *")
		require.NoError(t, err)
		assert.Equal(t, "0 4 * * *", sched.CronExpression())
	})

	t.Run("fail updating with invalid cron expression", func(t *testing.T) {
		err := sched.UpdateCronExpression("bad cron")
		assert.ErrorIs(t, err, model.ErrInvalidCronExpression)
		assert.Equal(t, "0 4 * * *", sched.CronExpression())
	})
}

func TestNewScheduleWithID(t *testing.T) {
	now := time.Now().UTC()
	sched, err := model.NewScheduleWithID(
		"sched-1", "suite-1", "art-1", nil, "prof-1",
		"nightly", "0 2 * * *", "vuhive-sched-1", true, now, now,
	)
	require.NoError(t, err)
	assert.Equal(t, "sched-1", sched.ID())

	_, err = model.NewScheduleWithID(
		"sched-1", "suite-1", "art-1", nil, "prof-1",
		"nightly", "invalid-cron", "vuhive-sched-1", true, now, now,
	)
	assert.ErrorIs(t, err, model.ErrInvalidCronExpression)
}
