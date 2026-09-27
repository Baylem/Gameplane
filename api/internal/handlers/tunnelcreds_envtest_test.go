//go:build envtest

package handlers

import (
	"context"
	"net/http"
	"testing"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"

	"github.com/ValgulNecron/gameplane/api/internal/scope"
)

// TestTunnelCreds_Put_KeepsActiveProviderKeyDuringSwitch_Envtest exercises
// the credential-rotation patch against a real apiserver. Unlike the
// package's fake-clientset tests, a real Secret converts stringData into
// data on write and drops stringData, so this is the test that actually
// proves the merge patch built by tunnelCredentialPatch deletes and keeps
// the right keys once the apiserver applies it.
func TestTunnelCreds_Put_KeepsActiveProviderKeyDuringSwitch_Envtest(t *testing.T) {
	name := uniqueResourceName("tun")

	gs := &unstructured.Unstructured{Object: map[string]any{
		"apiVersion": "gameplane.local/v1alpha1",
		"kind":       "GameServer",
		"metadata":   map[string]any{"name": name},
		"spec": map[string]any{
			"templateRef": map[string]any{"name": "minecraft"},
			"networking": map[string]any{
				"tunnel": map[string]any{
					"enabled":  false,
					"provider": "frp",
				},
			},
		},
	}}
	if _, err := kubeC.Dynamic.Resource(gvrServers()).Namespace(scope.DefaultNamespace).
		Create(context.Background(), gs, metav1.CreateOptions{}); err != nil {
		t.Fatalf("create gameserver: %v", err)
	}
	t.Cleanup(func() {
		_ = kubeC.Dynamic.Resource(gvrServers()).Namespace(scope.DefaultNamespace).
			Delete(context.Background(), name, metav1.DeleteOptions{})
	})

	path := "/servers/" + name + ":tunnel-credentials"

	// The frp tunnel is already running with a saved token.
	resp := doJSON(t, http.MethodPut, path, putReq{
		Provider: "frp",
		Values:   map[string]string{"token": "frp-token"},
	})
	if resp.StatusCode != http.StatusNoContent {
		t.Fatalf("frp PUT status = %d, want 204; body=%s", resp.StatusCode, readBody(t, resp))
	}

	// The dashboard saves tailscale credentials before the GameServer's
	// spec.networking.tunnel.provider is switched away from frp.
	resp = doJSON(t, http.MethodPut, path, putReq{
		Provider: "tailscale",
		Values:   map[string]string{"authKey": "ts-key"},
	})
	if resp.StatusCode != http.StatusNoContent {
		t.Fatalf("tailscale PUT status = %d, want 204; body=%s", resp.StatusCode, readBody(t, resp))
	}

	secretName := name + "-tunnel-auth"
	secret, err := kubeC.Typed.CoreV1().Secrets(scope.DefaultNamespace).
		Get(context.Background(), secretName, metav1.GetOptions{})
	if err != nil {
		t.Fatalf("get secret: %v", err)
	}
	if got := string(secret.Data["token"]); got != "frp-token" {
		t.Fatalf("token = %q, want frp-token (active provider key must survive an in-flight switch)", got)
	}
	if got := string(secret.Data["authKey"]); got != "ts-key" {
		t.Fatalf("authKey = %q, want ts-key", got)
	}
	if len(secret.StringData) != 0 {
		t.Fatalf("stringData = %v, want empty (apiserver folds writes into data)", secret.StringData)
	}

	// Once the spec provider field actually switches to tailscale, saving
	// credentials again must clean up the now-inactive frp key.
	current, err := kubeC.Dynamic.Resource(gvrServers()).Namespace(scope.DefaultNamespace).
		Get(context.Background(), name, metav1.GetOptions{})
	if err != nil {
		t.Fatalf("get gameserver: %v", err)
	}
	if err := unstructured.SetNestedField(current.Object, "tailscale", "spec", "networking", "tunnel", "provider"); err != nil {
		t.Fatalf("set provider: %v", err)
	}
	if _, err := kubeC.Dynamic.Resource(gvrServers()).Namespace(scope.DefaultNamespace).
		Update(context.Background(), current, metav1.UpdateOptions{}); err != nil {
		t.Fatalf("update gameserver spec provider: %v", err)
	}

	resp = doJSON(t, http.MethodPut, path, putReq{
		Provider: "tailscale",
		Values:   map[string]string{"authKey": "ts-key-2"},
	})
	if resp.StatusCode != http.StatusNoContent {
		t.Fatalf("tailscale re-save PUT status = %d, want 204; body=%s", resp.StatusCode, readBody(t, resp))
	}

	secret, err = kubeC.Typed.CoreV1().Secrets(scope.DefaultNamespace).
		Get(context.Background(), secretName, metav1.GetOptions{})
	if err != nil {
		t.Fatalf("get secret after spec switch: %v", err)
	}
	if _, ok := secret.Data["token"]; ok {
		t.Fatal("frp key still present after the spec provider switched away from frp")
	}
	if got := string(secret.Data["authKey"]); got != "ts-key-2" {
		t.Fatalf("authKey = %q, want ts-key-2", got)
	}
}
