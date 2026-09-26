package kube

import (
	"context"
	"testing"
	"time"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/types"
	dynamicfake "k8s.io/client-go/dynamic/fake"
	kubefake "k8s.io/client-go/kubernetes/fake"
	"k8s.io/client-go/kubernetes/scheme"
	"k8s.io/client-go/tools/cache"
)

func newTestCluster(name string) *unstructured.Unstructured {
	return &unstructured.Unstructured{
		Object: map[string]any{
			"apiVersion": "gameplane.local/v1alpha1",
			"kind":       "Cluster",
			"metadata": map[string]any{
				"name": name,
			},
			"spec": map[string]any{
				"displayName": "Test Cluster " + name,
				"kubeconfigSecret": map[string]any{
					"name": "cluster-" + name + "-kubeconfig",
					"key":  "kubeconfig",
				},
			},
		},
	}
}

func kubeconfig() []byte {
	return []byte(`apiVersion: v1
kind: Config
clusters:
- cluster:
    server: https://example.com:6443
  name: test
contexts:
- context:
    cluster: test
    user: admin
  name: test
current-context: test
users:
- name: admin
  user:
    token: fake-token`)
}

func TestWatchClusters_LoadsRemoteCluster(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	// Set up fake clients with Cluster CRD support.
	scm := runtime.NewScheme()
	_ = scheme.AddToScheme(scm)

	gvkr := map[schema.GroupVersionResource]string{
		GVRCluster: "ClusterList",
	}
	dyn := dynamicfake.NewSimpleDynamicClientWithCustomListKinds(scm, gvkr, newTestCluster("remote"))
	typed := kubefake.NewClientset()

	// Create the kubeconfig Secret.
	secret := &corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "cluster-remote-kubeconfig",
			Namespace: "gameplane-system",
			Labels: map[string]string{
				ClusterKubeconfigLabel: "true",
			},
		},
		Type: corev1.SecretTypeOpaque,
		Data: map[string][]byte{
			"kubeconfig": kubeconfig(),
		},
	}
	_, _ = typed.CoreV1().Secrets("gameplane-system").Create(ctx, secret, metav1.CreateOptions{})

	home := &Client{Dynamic: dyn, Typed: typed}
	reg := NewRegistry("local")
	reg.Set("local", home) // The local cluster must exist.

	// Run the watcher.
	go WatchClusters(ctx, home, reg, "gameplane-system")

	// Poll until the remote cluster is registered or timeout.
	deadline := time.Now().Add(3 * time.Second)
	for time.Now().Before(deadline) {
		if _, ok := reg.Get("remote"); ok {
			return // Success!
		}
		time.Sleep(100 * time.Millisecond)
	}
	t.Fatal("watcher did not load remote cluster")
}

func TestWatchClusters_IgnoresLocalCluster(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	scm := runtime.NewScheme()
	_ = scheme.AddToScheme(scm)

	// Create a fake Cluster with name "local" (the default).
	gvkr := map[schema.GroupVersionResource]string{
		GVRCluster: "ClusterList",
	}
	dyn := dynamicfake.NewSimpleDynamicClientWithCustomListKinds(scm, gvkr, newTestCluster("local"))
	typed := kubefake.NewClientset()

	home := &Client{Dynamic: dyn, Typed: typed}
	reg := NewRegistry("local")
	reg.Set("local", home)

	// Store the original local client to verify it's not replaced.
	originalLocal := reg.Default()

	go WatchClusters(ctx, home, reg, "gameplane-system")

	// Wait a bit for the watcher to process.
	time.Sleep(1 * time.Second)

	// The local client should remain unchanged.
	if reg.Default() != originalLocal {
		t.Fatal("watcher replaced the local cluster client")
	}
}

func TestWatchClusters_DeletesCluster(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	scm := runtime.NewScheme()
	_ = scheme.AddToScheme(scm)

	gvkr := map[schema.GroupVersionResource]string{
		GVRCluster: "ClusterList",
	}
	// Start with a remote cluster registered.
	dyn := dynamicfake.NewSimpleDynamicClientWithCustomListKinds(scm, gvkr, newTestCluster("remote"))
	typed := kubefake.NewClientset()

	// Create the kubeconfig Secret.
	secret := &corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "cluster-remote-kubeconfig",
			Namespace: "gameplane-system",
			Labels: map[string]string{
				ClusterKubeconfigLabel: "true",
			},
		},
		Type: corev1.SecretTypeOpaque,
		Data: map[string][]byte{
			"kubeconfig": kubeconfig(),
		},
	}
	_, _ = typed.CoreV1().Secrets("gameplane-system").Create(ctx, secret, metav1.CreateOptions{})

	home := &Client{Dynamic: dyn, Typed: typed}
	reg := NewRegistry("local")
	reg.Set("local", home)

	go WatchClusters(ctx, home, reg, "gameplane-system")

	// Wait for the cluster to be loaded.
	deadline := time.Now().Add(2 * time.Second)
	for time.Now().Before(deadline) {
		if _, ok := reg.Get("remote"); ok {
			break
		}
		time.Sleep(100 * time.Millisecond)
	}

	// Now delete it from the dynamic client.
	_ = dyn.Resource(GVRCluster).Delete(ctx, "remote", metav1.DeleteOptions{})

	// Poll until it's removed.
	deadline = time.Now().Add(3 * time.Second)
	for time.Now().Before(deadline) {
		if _, ok := reg.Get("remote"); !ok {
			return // Success!
		}
		time.Sleep(100 * time.Millisecond)
	}
	t.Fatal("watcher did not delete remote cluster")
}

func TestRemoveDeletedCluster_RemovesClientForObjectAndTombstone(t *testing.T) {
	cases := map[string]any{
		"cluster object":           newTestCluster("remote"),
		"tombstone with object":    cache.DeletedFinalStateUnknown{Key: "remote", Obj: newTestCluster("remote")},
		"tombstone without object": cache.DeletedFinalStateUnknown{Key: "remote"},
	}
	for name, obj := range cases {
		t.Run(name, func(t *testing.T) {
			reg := NewRegistry("local")
			reg.Set("local", &Client{})
			reg.Set("remote", &Client{})

			removeDeletedCluster(reg, obj)

			if _, ok := reg.Get("remote"); ok {
				t.Fatal("remote cluster client still registered after delete")
			}
			if _, ok := reg.Get("local"); !ok {
				t.Fatal("local cluster client removed")
			}
		})
	}
}

func TestRemoveDeletedCluster_KeepsDefaultCluster(t *testing.T) {
	reg := NewRegistry("local")
	reg.Set("local", &Client{})

	removeDeletedCluster(reg, newTestCluster("local"))
	removeDeletedCluster(reg, cache.DeletedFinalStateUnknown{Key: "local", Obj: newTestCluster("local")})

	if _, ok := reg.Get("local"); !ok {
		t.Fatal("default cluster client removed by a delete event")
	}
}

func TestRemoveDeletedCluster_KeepsNewerRegistrationWithSameName(t *testing.T) {
	cases := map[string]any{
		"unstructured":       newTestCluster("remote"),
		"tombstone with obj": cache.DeletedFinalStateUnknown{Key: "remote", Obj: newTestCluster("remote")},
	}

	for name, obj := range cases {
		t.Run(name, func(t *testing.T) {
			// Update the object in the test case if it's a Cluster object.
			if u, ok := obj.(*unstructured.Unstructured); ok {
				u.SetUID(types.UID("uid-old"))
			} else if ts, ok := obj.(cache.DeletedFinalStateUnknown); ok {
				if u, ok := ts.Obj.(*unstructured.Unstructured); ok {
					u.SetUID(types.UID("uid-old"))
				}
			}

			reg := NewRegistry("local")
			reg.Set("local", &Client{})
			// Register "remote" with a new UID.
			reg.SetWithUID("remote", "uid-new", &Client{})

			// Try to remove with an old UID.
			removeDeletedCluster(reg, obj)

			// The newer registration should be kept.
			if _, ok := reg.Get("remote"); !ok {
				t.Fatal("newer cluster registration with same name was removed")
			}

			// Now try with matching UID.
			reg.SetWithUID("remote", "uid-match", &Client{})
			objMatch := newTestCluster("remote")
			objMatch.SetUID(types.UID("uid-match"))
			removeDeletedCluster(reg, objMatch)

			// This time it should be removed.
			if _, ok := reg.Get("remote"); ok {
				t.Fatal("cluster with matching UID was not removed")
			}
		})
	}
}

func TestLoadCluster_SkipsClusterBeingDeleted(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	scm := runtime.NewScheme()
	_ = scheme.AddToScheme(scm)

	// Create a test cluster with a deletionTimestamp and UID.
	clusterObj := newTestCluster("remote")
	clusterObj.SetUID(types.UID("uid-123"))
	clusterObj.SetDeletionTimestamp(&metav1.Time{Time: time.Now()})

	gvkr := map[schema.GroupVersionResource]string{
		GVRCluster: "ClusterList",
	}
	dyn := dynamicfake.NewSimpleDynamicClientWithCustomListKinds(scm, gvkr, clusterObj)
	typed := kubefake.NewClientset()

	home := &Client{Dynamic: dyn, Typed: typed}
	reg := NewRegistry("local")
	reg.Set("local", home)

	// Pre-register "remote" with the same UID.
	reg.SetWithUID("remote", "uid-123", &Client{})

	// Call loadCluster; it should skip the cluster and remove it from the registry.
	err := loadCluster(ctx, home, reg, "gameplane-system", "remote")
	if err != nil {
		t.Fatalf("loadCluster() error = %v", err)
	}

	// "remote" should not be registered.
	if _, ok := reg.Get("remote"); ok {
		t.Fatal("cluster being deleted was registered")
	}

	// Test that a newly loaded cluster is registered when not being deleted.
	clusterObj2 := newTestCluster("remote2")
	clusterObj2.SetUID(types.UID("uid-456"))
	dyn.Resource(GVRCluster).Create(ctx, clusterObj2, metav1.CreateOptions{})

	// Create the kubeconfig Secret for remote2.
	secret := &corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "cluster-remote2-kubeconfig",
			Namespace: "gameplane-system",
			Labels: map[string]string{
				ClusterKubeconfigLabel: "true",
			},
		},
		Type: corev1.SecretTypeOpaque,
		Data: map[string][]byte{
			"kubeconfig": kubeconfig(),
		},
	}
	_, _ = typed.CoreV1().Secrets("gameplane-system").Create(ctx, secret, metav1.CreateOptions{})

	err = loadCluster(ctx, home, reg, "gameplane-system", "remote2")
	if err != nil {
		t.Fatalf("loadCluster() error = %v", err)
	}

	// "remote2" should be registered.
	if _, ok := reg.Get("remote2"); !ok {
		t.Fatal("cluster without deletionTimestamp was not registered")
	}
}
