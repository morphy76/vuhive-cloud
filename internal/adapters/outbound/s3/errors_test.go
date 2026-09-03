package s3_test

import (
	"errors"
	"fmt"
	"testing"

	"github.com/aws/aws-sdk-go-v2/aws"
	s3types "github.com/aws/aws-sdk-go-v2/service/s3/types"
	"github.com/aws/smithy-go"
	"github.com/morphy76/vuhive-cloud/internal/adapters/outbound/s3"
	"github.com/morphy76/vuhive-cloud/internal/domain/model"
	"github.com/stretchr/testify/assert"
)

func TestMapError(t *testing.T) {
	t.Run("nil error returns nil", func(t *testing.T) {
		assert.NoError(t, s3.MapError(nil))
	})

	t.Run("types.NoSuchKey maps to model.ErrNotFound", func(t *testing.T) {
		err := &s3types.NoSuchKey{
			Message: aws.String("The specified key does not exist."),
		}
		mapped := s3.MapError(err)
		assert.ErrorIs(t, mapped, model.ErrNotFound)
	})

	t.Run("types.NotFound maps to model.ErrNotFound", func(t *testing.T) {
		err := &s3types.NotFound{
			Message: aws.String("Not Found"),
		}
		mapped := s3.MapError(err)
		assert.ErrorIs(t, mapped, model.ErrNotFound)
	})

	t.Run("smithy generic API error with NotFound maps to model.ErrNotFound", func(t *testing.T) {
		err := &smithy.GenericAPIError{
			Code:    "NotFound",
			Message: "Resource not found",
		}
		mapped := s3.MapError(err)
		assert.ErrorIs(t, mapped, model.ErrNotFound)
	})

	t.Run("smithy generic API error with NoSuchKey maps to model.ErrNotFound", func(t *testing.T) {
		err := &smithy.GenericAPIError{
			Code:    "NoSuchKey",
			Message: "Key does not exist",
		}
		mapped := s3.MapError(err)
		assert.ErrorIs(t, mapped, model.ErrNotFound)
	})

	t.Run("smithy generic API error with NoSuchBucket maps to model.ErrNotFound", func(t *testing.T) {
		err := &smithy.GenericAPIError{
			Code:    "NoSuchBucket",
			Message: "Bucket does not exist",
		}
		mapped := s3.MapError(err)
		assert.ErrorIs(t, mapped, model.ErrNotFound)
	})

	t.Run("domain errors pass through", func(t *testing.T) {
		mapped := s3.MapError(model.ErrEmptyS3Key)
		assert.ErrorIs(t, mapped, model.ErrEmptyS3Key)
	})

	t.Run("generic error preserved", func(t *testing.T) {
		customErr := errors.New("connection reset by peer")
		mapped := s3.MapError(customErr)
		assert.Equal(t, customErr, mapped)
	})

	t.Run("wrapped smithy error maps to model.ErrNotFound", func(t *testing.T) {
		wrapped := fmt.Errorf("operation failed: %w", &smithy.GenericAPIError{
			Code:    "NoSuchKey",
			Message: "Key missing",
		})
		mapped := s3.MapError(wrapped)
		assert.ErrorIs(t, mapped, model.ErrNotFound)
	})
}
