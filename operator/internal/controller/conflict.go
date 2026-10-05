package controller

import (
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	ctrl "sigs.k8s.io/controller-runtime"
)

// requeueOnConflict turns an optimistic-concurrency conflict ("the object has
// been modified") into a quiet requeue. Conflicts are an expected, transient
// race between reconcilers and the API server cache; returning them as errors
// logs a spurious ERROR on every race. Any other result passes through.
func requeueOnConflict(res ctrl.Result, err error) (ctrl.Result, error) {
	if apierrors.IsConflict(err) {
		return ctrl.Result{Requeue: true}, nil
	}
	return res, err
}
