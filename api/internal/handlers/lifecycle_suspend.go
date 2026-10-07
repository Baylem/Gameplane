package handlers

import (
	"context"
	"encoding/json"
	"errors"
	"reflect"

	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/apimachinery/pkg/util/wait"
	"k8s.io/client-go/util/retry"

	"github.com/ValgulNecron/gameplane/api/internal/kube"
	"github.com/ValgulNecron/gameplane/api/internal/rbac"
)

// patchServerSuspend retries controller write contention without replaying a
// power request against a replacement server or changed configuration/grants.
func patchServerSuspend(ctx context.Context, k *kube.Client, cluster, ns, name string, suspend bool) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	resource := k.Dynamic.Resource(kube.GVRs["servers"]).Namespace(ns)
	original, err := resource.Get(ctx, name, metav1.GetOptions{})
	if err != nil {
		return err
	}
	if original.GetUID() == "" || original.GetDeletionTimestamp() != nil ||
		rbac.ValidateServerIdentity(ctx, cluster, ns, name, string(original.GetUID())) != nil {
		return apierrors.NewNotFound(kube.GVRs["servers"].GroupResource(), name)
	}
	var lastConflict error
	attempt := 0
	err = wait.ExponentialBackoffWithContext(ctx, retry.DefaultRetry, func(ctx context.Context) (bool, error) {
		if err := ctx.Err(); err != nil {
			return false, err
		}
		current := original
		if attempt > 0 {
			fresh, err := resource.Get(ctx, name, metav1.GetOptions{})
			if err != nil {
				return false, err
			}
			if fresh.GetUID() != original.GetUID() || fresh.GetDeletionTimestamp() != nil {
				return false, apierrors.NewNotFound(kube.GVRs["servers"].GroupResource(), name)
			}
			if !reflect.DeepEqual(fresh.Object["spec"], original.Object["spec"]) ||
				fresh.GetAnnotations()[ownerIDAnnotation] != original.GetAnnotations()[ownerIDAnnotation] ||
				fresh.GetAnnotations()[collaboratorsAnnotation] != original.GetAnnotations()[collaboratorsAnnotation] {
				return false, lastConflict
			}
			current = fresh
		}
		body, err := json.Marshal(map[string]any{
			"metadata": map[string]any{"uid": original.GetUID(), "resourceVersion": current.GetResourceVersion()},
			"spec":     map[string]any{"suspend": suspend},
		})
		if err != nil {
			return false, err
		}
		attempt++
		_, err = resource.Patch(ctx, name, types.MergePatchType, body, metav1.PatchOptions{})
		if apierrors.IsConflict(err) {
			lastConflict = err
			return false, nil
		}
		return err == nil, err
	})
	if wait.Interrupted(err) {
		if ctxErr := ctx.Err(); ctxErr != nil {
			return ctxErr
		}
		if lastConflict != nil && !errors.Is(err, context.Canceled) && !errors.Is(err, context.DeadlineExceeded) {
			return lastConflict
		}
	}
	return err
}
