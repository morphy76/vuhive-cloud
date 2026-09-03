package s3

import (
	"errors"

	s3types "github.com/aws/aws-sdk-go-v2/service/s3/types"
	"github.com/aws/smithy-go"
	"github.com/morphy76/vuhive-cloud/internal/domain/model"
)

// MapError translates AWS S3 and SDK driver errors into domain errors.
func MapError(err error) error {
	if err == nil {
		return nil
	}

	// Direct check for domain errors
	if errors.Is(err, model.ErrNotFound) || errors.Is(err, model.ErrEmptyS3Key) || errors.Is(err, model.ErrValidation) {
		return err
	}

	// Check typed S3 errors
	var noSuchKey *s3types.NoSuchKey
	if errors.As(err, &noSuchKey) {
		return model.ErrNotFound
	}

	var notFound *s3types.NotFound
	if errors.As(err, &notFound) {
		return model.ErrNotFound
	}

	var noSuchBucket *s3types.NoSuchBucket
	if errors.As(err, &noSuchBucket) {
		return model.ErrNotFound
	}

	// Check generic Smithy API errors
	var apiErr smithy.APIError
	if errors.As(err, &apiErr) {
		switch apiErr.ErrorCode() {
		case "NoSuchKey", "NotFound", "NoSuchBucket":
			return model.ErrNotFound
		}
	}

	return err
}
