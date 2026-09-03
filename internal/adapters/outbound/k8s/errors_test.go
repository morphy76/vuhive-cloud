package k8s_test

import (
	"context"
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/runtime/schema"

	"github.com/morphy76/vuhive-cloud/internal/adapters/outbound/k8s"
	"github.com/morphy76/vuhive-cloud/internal/domain/model"
)

func TestMapK8sError(t *testing.T) {
	assert.Nil(t, k8s.MapK8sError(nil))

	t.Run("NotFound maps to ErrNotFound", func(t *testing.T) {
		err := apierrors.NewNotFound(schema.GroupResource{Resource: "jobs"}, "test-job")
		mapped := k8s.MapK8sError(err)
		assert.ErrorIs(t, mapped, model.ErrNotFound)
	})

	t.Run("AlreadyExists maps to ErrConflict", func(t *testing.T) {
		err := apierrors.NewAlreadyExists(schema.GroupResource{Resource: "jobs"}, "test-job")
		mapped := k8s.MapK8sError(err)
		assert.ErrorIs(t, mapped, model.ErrConflict)
	})

	t.Run("Timeout maps to ErrTimeout", func(t *testing.T) {
		err := apierrors.NewTimeoutError("timed out", 5)
		mapped := k8s.MapK8sError(err)
		assert.ErrorIs(t, mapped, model.ErrTimeout)
	})

	t.Run("context.DeadlineExceeded maps to ErrTimeout", func(t *testing.T) {
		mapped := k8s.MapK8sError(context.DeadlineExceeded)
		assert.ErrorIs(t, mapped, model.ErrTimeout)
	})

	t.Run("context.Canceled preserves error", func(t *testing.T) {
		mapped := k8s.MapK8sError(context.Canceled)
		assert.ErrorIs(t, mapped, context.Canceled)
	})

	t.Run("Unknown error is preserved", func(t *testing.T) {
		customErr := errors.New("custom error")
		mapped := k8s.MapK8sError(customErr)
		assert.Equal(t, customErr, mapped)
	})
}
