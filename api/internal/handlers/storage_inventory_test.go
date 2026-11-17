package handlers

import (
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/go-chi/chi/v5"
	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	dynamicfake "k8s.io/client-go/dynamic/fake"
	kubefake "k8s.io/client-go/kubernetes/fake"
	kubetesting "k8s.io/client-go/testing"

	"github.com/GameplanePanel/gameplane/api/internal/auth"
	"github.com/GameplanePanel/gameplane/api/internal/kube"
	"github.com/GameplanePanel/gameplane/api/internal/rbac"
)

const inventoryPoolAnnotationFixture = `{"provider":"openebs-lvm","namespace":"openebs","volumeGroup":"openebs-chicago-1","thinPool":"openebs-chicago-1_thinpool","storageClasses":["openebs-lvm"]}`

// Wire assertions intentionally do not reference new production helpers or fields:
// this regression patch must compile against the unchanged v0.4.1 handlers.
type inventoryPoolStatsWire struct {
	Nodes                 int    `json:"nodes"`
	TotalStorageBytes     int64  `json:"totalStorageBytes"`
	UsedStorageBytes      int64  `json:"usedStorageBytes"`
	StorageCapacitySource string `json:"storageCapacitySource"`
}

type inventoryPoolFleetWire struct {
	Items []struct {
		Cluster string                 `json:"cluster"`
		Stats   inventoryPoolStatsWire `json:"stats"`
	} `json:"items"`
	Partial bool `json:"partial"`
	Issues  []struct {
		Cluster string `json:"cluster"`
		Code    string `json:"code"`
	} `json:"issues"`
}

func inventoryPoolNodeFixture(name, annotation string) *corev1.Node {
	node := readyNode(name, true, "4", "8Gi") // Reports system EPHEMERAL = 100Gi.
	node.Labels["openebs.io/nodename"] = name
	node.Labels["kubernetes.io/hostname"] = name
	if annotation != "" {
		node.Annotations = map[string]string{"gameplane.local/storage-pool": annotation}
	}
	return node
}

func inventoryPoolObjectFixture(name string, size any) *unstructured.Unstructured {
	return &unstructured.Unstructured{Object: map[string]any{
		"apiVersion": "local.openebs.io/v1alpha1", "kind": "LVMNode",
		"metadata": map[string]any{"name": name, "namespace": "openebs"},
		// This is the real OpenEBS shape: volumeGroups is top-level, not status/spec.
		"volumeGroups": []any{map[string]any{
			"name": "openebs-chicago-1", "size": "372336Mi", "free": "42608Mi",
			"thinPools": []any{map[string]any{"name": "openebs-chicago-1_thinpool", "size": size, "free": "300Gi"}},
		}},
	}}
}

func inventoryPoolPVFixture(name, size, class, vg, node string, phase corev1.PersistentVolumePhase) *corev1.PersistentVolume {
	pv := boundPV(name, size, true)
	pv.Status.Phase = phase
	pv.Spec.StorageClassName = class
	pv.Spec.CSI = &corev1.CSIPersistentVolumeSource{
		Driver: "local.csi.openebs.io", VolumeHandle: name,
		VolumeAttributes: map[string]string{"openebs.io/volgroup": vg},
	}
	pv.Spec.NodeAffinity = &corev1.VolumeNodeAffinity{Required: &corev1.NodeSelector{NodeSelectorTerms: []corev1.NodeSelectorTerm{{
		MatchExpressions: []corev1.NodeSelectorRequirement{{Key: "openebs.io/nodename", Operator: corev1.NodeSelectorOpIn, Values: []string{node}}},
	}}}}
	return pv
}

func inventoryPoolClientFixture(annotation string, pool *unstructured.Unstructured, extras ...runtime.Object) *kube.Client {
	var dynamicObjects []runtime.Object
	if pool != nil {
		dynamicObjects = append(dynamicObjects, pool)
	}
	client := fleetTestClient(dynamicObjects...)
	objects := []runtime.Object{
		inventoryPoolNodeFixture("pool-node", annotation),
		inventoryPoolPVFixture("selected", "7Gi", "openebs-lvm", "openebs-chicago-1", "pool-node", corev1.VolumeBound),
	}
	client.Typed = kubefake.NewClientset(append(objects, extras...)...)
	return client
}

func inventoryPoolRequestFixture(t *testing.T, route string, remote *kube.Client, user *auth.User) (*httptest.ResponseRecorder, *kube.Client) {
	t.Helper()
	home := fleetTestClient(newCluster("remote", nil, nil))
	home.Typed = kubefake.NewClientset(readyNode("home-node", true, "4", "8Gi"), boundPV("home-only", "91Gi", true))
	reg := clusterTestRegistry(home)
	reg.Set("remote", remote)
	router := chi.NewRouter()
	router.Use(rbac.Middleware(reg))
	MountCluster(router, reg, nil, "test", false, "")
	MountFleet(router, reg, nil)
	return inventoryRequest(t, router, user, http.MethodGet, route+"?cluster=remote"), home
}

func inventoryPoolDecodeFixture(t *testing.T, route string, rr *httptest.ResponseRecorder) inventoryPoolStatsWire {
	t.Helper()
	if rr.Code != http.StatusOK {
		t.Fatalf("%s status=%d body=%s", route, rr.Code, rr.Body)
	}
	if route == "/cluster/stats" {
		var stats inventoryPoolStatsWire
		if err := json.Unmarshal(rr.Body.Bytes(), &stats); err != nil {
			t.Fatal(err)
		}
		return stats
	}
	var fleet inventoryPoolFleetWire
	if err := json.Unmarshal(rr.Body.Bytes(), &fleet); err != nil {
		t.Fatal(err)
	}
	if fleet.Partial || len(fleet.Items) != 1 || fleet.Items[0].Cluster != "remote" {
		t.Fatalf("unexpected inventory coverage: %+v", fleet)
	}
	return fleet.Items[0].Stats
}

func inventoryPoolNoHomeReadFixture(t *testing.T, home *kube.Client) {
	t.Helper()
	if len(home.Typed.(*kubefake.Clientset).Actions()) != 0 {
		t.Fatal("selected remote storage queried Home node/PV inventory")
	}
	for _, action := range home.Dynamic.(*dynamicfake.FakeDynamicClient).Actions() {
		if action.GetResource().Resource == "lvmnodes" {
			t.Fatal("selected remote storage queried Home LVMNode")
		}
	}
}

func TestStoragePoolInventory_SelectedPoolAndPVScope(t *testing.T) {
	for _, route := range []string{"/cluster/stats", "/fleet/inventory"} {
		for _, quantity := range []struct {
			name string
			size any
		}{{"quantity string", "320Gi"}, {"integer bytes", int64(320 << 30)}} {
			t.Run(route+"/"+quantity.name, func(t *testing.T) {
				otherDriver := inventoryPoolPVFixture("other-driver", "19Gi", "openebs-lvm", "openebs-chicago-1", "pool-node", corev1.VolumeBound)
				otherDriver.Spec.CSI.Driver = "unrelated.csi.example"
				remote := inventoryPoolClientFixture(inventoryPoolAnnotationFixture, inventoryPoolObjectFixture("pool-node", quantity.size),
					inventoryPoolPVFixture("released", "11Gi", "openebs-lvm", "openebs-chicago-1", "pool-node", corev1.VolumeReleased),
					inventoryPoolPVFixture("wrong-class", "13Gi", "another-class", "openebs-chicago-1", "pool-node", corev1.VolumeBound),
					inventoryPoolPVFixture("wrong-vg", "17Gi", "openebs-lvm", "another-vg", "pool-node", corev1.VolumeBound),
					inventoryPoolPVFixture("wrong-node", "23Gi", "openebs-lvm", "openebs-chicago-1", "another-node", corev1.VolumeBound), otherDriver)
				rr, home := inventoryPoolRequestFixture(t, route, remote, inventoryUser("remote", "*", "cluster:read"))
				stats := inventoryPoolDecodeFixture(t, route, rr)
				if stats.TotalStorageBytes != 320<<30 || stats.UsedStorageBytes != 7<<30 || stats.StorageCapacitySource != "openebs-lvm" {
					t.Fatalf("want 320Gi pool/7Gi provisioned, not 100Gi EPHEMERAL or 20Gi physical-used: %+v", stats)
				}
				var lookups int
				for _, action := range remote.Dynamic.(*dynamicfake.FakeDynamicClient).Actions() {
					if action.GetResource().Resource != "lvmnodes" {
						continue
					}
					get, ok := action.(kubetesting.GetAction)
					if !ok || action.GetNamespace() != "openebs" || get.GetName() != "pool-node" {
						t.Fatalf("pool discovery exceeded exact named GET: %v", action)
					}
					lookups++
				}
				if lookups == 0 {
					t.Fatal("configured capacity did not read the named LVMNode")
				}
				inventoryPoolNoHomeReadFixture(t, home)
			})
		}
	}
}

func TestStoragePoolInventory_ConfiguredFailuresDoNotFallback(t *testing.T) {
	for _, route := range []string{"/cluster/stats", "/fleet/inventory"} {
		for _, tc := range []struct {
			name       string
			annotation string
			modify     func(*unstructured.Unstructured)
			lookupErr  error
		}{
			{name: "malformed annotation", annotation: "{"},
			{name: "unsupported provider", annotation: strings.Replace(inventoryPoolAnnotationFixture, "openebs-lvm\"", "unsupported\"", 1)},
			{name: "case-variant annotation alias", annotation: strings.Replace(inventoryPoolAnnotationFixture, "{", "{\"Provider\":\"openebs-lvm\",", 1)},
			{name: "duplicate annotation key", annotation: strings.Replace(inventoryPoolAnnotationFixture, "{", "{\"provider\":\"openebs-lvm\",", 1)},
			{name: "unknown field", annotation: strings.Replace(inventoryPoolAnnotationFixture, "{", "{\"unexpected\":true,", 1)},
			{name: "empty classes", annotation: strings.Replace(inventoryPoolAnnotationFixture, "[\"openebs-lvm\"]", "[]", 1)},
			{name: "invalid quantity", modify: func(p *unstructured.Unstructured) { inventoryPoolSizeFixture(p, "not-a-quantity") }},
			{name: "zero capacity", modify: func(p *unstructured.Unstructured) { inventoryPoolSizeFixture(p, "0") }},
			{name: "quantity overflow", modify: func(p *unstructured.Unstructured) { inventoryPoolSizeFixture(p, "9223372036854775808") }},
			{name: "missing thin pool", modify: func(p *unstructured.Unstructured) { p.Object["volumeGroups"] = []any{} }},
			{name: "duplicate matching VG", modify: func(p *unstructured.Unstructured) {
				groups := p.Object["volumeGroups"].([]any)
				p.Object["volumeGroups"] = append(groups, runtime.DeepCopyJSONValue(groups[0]))
			}},
			{name: "duplicate matching pool", modify: func(p *unstructured.Unstructured) {
				group := p.Object["volumeGroups"].([]any)[0].(map[string]any)
				pools := group["thinPools"].([]any)
				group["thinPools"] = append(pools, runtime.DeepCopyJSONValue(pools[0]))
			}},
			{name: "forbidden lookup", lookupErr: apierrors.NewForbidden(schema.GroupResource{Group: "local.openebs.io", Resource: "lvmnodes"}, "pool-node", errors.New("private-pool-detail"))},
			{name: "missing LVMNode", lookupErr: apierrors.NewNotFound(schema.GroupResource{Group: "local.openebs.io", Resource: "lvmnodes"}, "pool-node")},
		} {
			t.Run(route+"/"+tc.name, func(t *testing.T) {
				annotation := tc.annotation
				if annotation == "" {
					annotation = inventoryPoolAnnotationFixture
				}
				pool := inventoryPoolObjectFixture("pool-node", "320Gi")
				if tc.modify != nil {
					tc.modify(pool)
				}
				remote := inventoryPoolClientFixture(annotation, pool)
				if tc.lookupErr != nil {
					remote.Dynamic.(*dynamicfake.FakeDynamicClient).PrependReactor("get", "lvmnodes", func(kubetesting.Action) (bool, runtime.Object, error) {
						return true, nil, tc.lookupErr
					})
				}
				rr, home := inventoryPoolRequestFixture(t, route, remote, inventoryUser("remote", "*", "cluster:read"))
				if route == "/cluster/stats" {
					if rr.Code == http.StatusOK {
						t.Fatalf("configured failure became known node/zero capacity: %s", rr.Body)
					}
				} else {
					var fleet inventoryPoolFleetWire
					if rr.Code != http.StatusOK || json.Unmarshal(rr.Body.Bytes(), &fleet) != nil || !fleet.Partial || len(fleet.Items) != 0 || len(fleet.Issues) == 0 || fleet.Issues[0].Cluster != "remote" {
						t.Fatalf("configured failure did not preserve unavailable fleet coverage: status=%d body=%s", rr.Code, rr.Body)
					}
				}
				if strings.Contains(rr.Body.String(), "private-pool-detail") {
					t.Fatal("pool lookup leaked upstream details")
				}
				inventoryPoolNoHomeReadFixture(t, home)
			})
		}
	}
}

func inventoryPoolSizeFixture(pool *unstructured.Unstructured, value any) {
	pool.Object["volumeGroups"].([]any)[0].(map[string]any)["thinPools"].([]any)[0].(map[string]any)["size"] = value
}

func TestStoragePoolInventory_SummedCapacityOverflowUnavailable(t *testing.T) {
	for _, route := range []string{"/cluster/stats", "/fleet/inventory"} {
		t.Run(route, func(t *testing.T) {
			remote := inventoryPoolClientFixture(inventoryPoolAnnotationFixture, inventoryPoolObjectFixture("pool-node", "5Ei"), inventoryPoolNodeFixture("second-node", inventoryPoolAnnotationFixture))
			if err := remote.Dynamic.(*dynamicfake.FakeDynamicClient).Tracker().Add(inventoryPoolObjectFixture("second-node", "5Ei")); err != nil {
				t.Fatal(err)
			}
			rr, _ := inventoryPoolRequestFixture(t, route, remote, inventoryUser("remote", "*", "cluster:read"))
			if route == "/cluster/stats" {
				if rr.Code == http.StatusOK {
					t.Fatalf("overflowing pool sum became known capacity: %s", rr.Body)
				}
			} else {
				var fleet inventoryPoolFleetWire
				if err := json.Unmarshal(rr.Body.Bytes(), &fleet); err != nil || !fleet.Partial || len(fleet.Items) != 0 {
					t.Fatalf("overflowing pool sum lost unknown coverage: %s", rr.Body)
				}
			}
		})
	}
}

func TestStoragePoolInventory_RequiresClusterInventoryAuthority(t *testing.T) {
	for _, route := range []string{"/cluster/stats", "/fleet/inventory"} {
		t.Run(route, func(t *testing.T) {
			remote := inventoryPoolClientFixture(inventoryPoolAnnotationFixture, inventoryPoolObjectFixture("pool-node", "320Gi"))
			rr, home := inventoryPoolRequestFixture(t, route, remote, inventoryUser("remote", "gameplane-games", "*"))
			if route == "/cluster/stats" && rr.Code != http.StatusForbidden {
				t.Fatalf("namespace grant read pool inventory: %d", rr.Code)
			}
			if route == "/fleet/inventory" {
				var fleet inventoryPoolFleetWire
				if err := json.Unmarshal(rr.Body.Bytes(), &fleet); err != nil || len(fleet.Items) != 0 || fleet.Partial {
					t.Fatalf("namespace grant probed pool inventory: %s", rr.Body)
				}
			}
			if len(remote.Typed.(*kubefake.Clientset).Actions()) != 0 || len(remote.Dynamic.(*dynamicfake.FakeDynamicClient).Actions()) != 0 {
				t.Fatal("namespace-only authority triggered a pool or node/PV lookup")
			}
			inventoryPoolNoHomeReadFixture(t, home)
		})
	}
}

func TestStoragePoolInventory_LegacyUnconfigured(t *testing.T) {
	for _, route := range []string{"/cluster/stats", "/fleet/inventory"} {
		t.Run(route, func(t *testing.T) {
			remote := inventoryPoolClientFixture("", inventoryPoolObjectFixture("pool-node", "320Gi"))
			rr, home := inventoryPoolRequestFixture(t, route, remote, inventoryUser("remote", "*", "cluster:read"))
			stats := inventoryPoolDecodeFixture(t, route, rr)
			if stats.TotalStorageBytes != 100<<30 || stats.UsedStorageBytes != 7<<30 || stats.StorageCapacitySource != "node-ephemeral" {
				t.Fatalf("unconfigured inventory changed legacy calculation/source: %+v", stats)
			}
			for _, action := range remote.Dynamic.(*dynamicfake.FakeDynamicClient).Actions() {
				if action.GetResource().Resource == "lvmnodes" {
					t.Fatal("unconfigured cluster read an unselected storage backend")
				}
			}
			inventoryPoolNoHomeReadFixture(t, home)
		})
	}
}

func TestStoragePoolInventory_NodeAffinityBooleanScope(t *testing.T) {
	in := func(key, value string) corev1.NodeSelectorRequirement {
		return corev1.NodeSelectorRequirement{Key: key, Operator: corev1.NodeSelectorOpIn, Values: []string{value}}
	}
	for _, route := range []string{"/cluster/stats", "/fleet/inventory"} {
		for _, tc := range []struct {
			name  string
			terms []corev1.NodeSelectorTerm
			want  int64
		}{
			{"OR terms", []corev1.NodeSelectorTerm{
				{MatchExpressions: []corev1.NodeSelectorRequirement{in("openebs.io/nodename", "other-node")}},
				{MatchExpressions: []corev1.NodeSelectorRequirement{in("openebs.io/nodename", "pool-node")}},
			}, 7 << 30},
			{"AND requirements", []corev1.NodeSelectorTerm{{MatchExpressions: []corev1.NodeSelectorRequirement{
				in("openebs.io/nodename", "pool-node"), in("missing-label", "required"),
			}}}, 0},
			{"empty term matches no node", []corev1.NodeSelectorTerm{{}}, 0},
			{"metadata name field", []corev1.NodeSelectorTerm{{MatchFields: []corev1.NodeSelectorRequirement{
				in("metadata.name", "pool-node"),
			}}}, 7 << 30},
			{"fields AND expressions", []corev1.NodeSelectorTerm{{
				MatchExpressions: []corev1.NodeSelectorRequirement{in("openebs.io/nodename", "pool-node")},
				MatchFields:      []corev1.NodeSelectorRequirement{in("metadata.name", "other-node")},
			}}, 0},
		} {
			t.Run(route+"/"+tc.name, func(t *testing.T) {
				remote := inventoryPoolClientFixture(inventoryPoolAnnotationFixture, inventoryPoolObjectFixture("pool-node", "320Gi"))
				pv, err := remote.Typed.CoreV1().PersistentVolumes().Get(t.Context(), "selected", metav1.GetOptions{})
				if err != nil {
					t.Fatal(err)
				}
				pv.Spec.NodeAffinity.Required.NodeSelectorTerms = tc.terms
				if _, err := remote.Typed.CoreV1().PersistentVolumes().Update(t.Context(), pv, metav1.UpdateOptions{}); err != nil {
					t.Fatal(err)
				}
				rr, _ := inventoryPoolRequestFixture(t, route, remote, inventoryUser("remote", "*", "cluster:read"))
				stats := inventoryPoolDecodeFixture(t, route, rr)
				if stats.TotalStorageBytes != 320<<30 || stats.UsedStorageBytes != tc.want {
					t.Fatalf("affinity scope: %+v want provisioned=%d", stats, tc.want)
				}
			})
		}
	}
}
