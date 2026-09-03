package k8s

import (
	"context"
	"errors"
	"fmt"

	apierrors "k8s.io/apimachinery/pkg/api/errors"

	"github.com/morphy76/vuhive-cloud/internal/domain/model"
)

// MapK8sError translates Kubernetes API and context errors into domain errors.
func MapK8sError(err error) error {
	if err == nil {
		return nil
	}

	if errors.Is(err, context.DeadlineExceeded) {
		return fmt.Errorf("%w: %v", model.ErrTimeout, err)
	}

	if errors.Is(err, context.Canceled) {
		return err
	}

	if apierrors.IsNotFound(err) {
		return fmt.Errorf("%w: %v", model.ErrNotFound, err)
	}

	if apierrors.IsAlreadyExists(err) {
		return fmt.Errorf("%w: %v", model.ErrConflict, err)
	}

	if apierrors.IsTimeout(err) || apierrors.IsServerTimeout(err) {
		return fmt.Errorf("%w: %v", model.ErrTimeout, err)
	}

	if apierrors.IsBadRequest(err) || apierrors.IsInvalid(err) {
		return fmt.Errorf("%w: %v", model.ErrValidation, err)
	}

	return err
}
