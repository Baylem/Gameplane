//go:build envtest

package controller

import (
	"context"
	"strings"
	"testing"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"

	gameplanev1alpha1 "github.com/ValgulNecron/gameplane/operator/api/v1alpha1"
)

// TestReconcileTunnel_CredentialSecretWithoutOwnerRefRefused: a tunnel
// credentials Secret with no ownerReference to the GameServer must be
// refused, and the refusal must be visible as a TunnelReady=False /
// TunnelCredentialRefused status condition naming the Secret — not only
// as an operator log line (F-069).
func TestReconcileTunnel_CredentialSecretWithoutOwnerRefRefused(t *testing.T) {
	ctx := context.Background()
	ns := newNamespace(t)

	tmpl := wakeableTestTemplate(uniqueName("tunnel-cred-tmpl"), "minecraft")
	if err := k8sClient.Create(ctx, tmpl); err != nil {
		t.Fatalf("create template: %v", err)
	}
	deleteCleanup(t, tmpl)

	secName := "unowned-tunnel-creds"
	sec := &corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{Name: secName, Namespace: ns},
		StringData: map[string]string{"token": "fake"},
	}
	if err := k8sClient.Create(ctx, sec); err != nil {
		t.Fatalf("create secret: %v", err)
	}

	gs := &gameplanev1alpha1.GameServer{
		ObjectMeta: metav1.ObjectMeta{Name: "test-tunnel-cred-refused", Namespace: ns},
		Spec: gameplanev1alpha1.GameServerSpec{
			TemplateRef: gameplanev1alpha1.GameTemplateRef{Name: tmpl.Name},
			Networking: gameplanev1alpha1.GameServerNetworking{
				Tunnel: &gameplanev1alpha1.GameServerTunnel{
					Enabled:              true,
					Provider:             "frp",
					CredentialsSecretRef: &gameplanev1alpha1.SecretNameRef{Name: secName},
					Frp: &gameplanev1alpha1.FrpTunnelSpec{
						ServerAddr:  "relay.example.invalid",
						RemotePorts: []gameplanev1alpha1.RemotePortMapping{{Name: "game", RemotePort: 25565}},
					},
				},
			},
		},
	}
	if err := k8sClient.Create(ctx, gs); err != nil {
		t.Fatalf("create gameserver: %v", err)
	}

	r := &GameServerReconciler{Client: k8sClient, APIReader: k8sClient, Scheme: scheme}
	if err := r.reconcileTunnel(ctx, gs, tmpl, true); err == nil || !strings.Contains(err.Error(), "is not owned by GameServer") {
		t.Fatalf("reconcileTunnel: want ownership refusal, got %v", err)
	}

	eventually(t, func() (bool, string) {
		var got gameplanev1alpha1.GameServer
		if err := k8sClient.Get(ctx, client.ObjectKeyFromObject(gs), &got); err != nil {
			return false, "get gameserver: " + err.Error()
		}
		for _, c := range got.Status.Conditions {
			if c.Type != "TunnelReady" {
				continue
			}
			if c.Status != metav1.ConditionFalse {
				return false, "TunnelReady status=" + string(c.Status) + ", want False"
			}
			if c.Reason != "TunnelCredentialRefused" {
				return false, "TunnelReady reason=" + c.Reason + ", want TunnelCredentialRefused"
			}
			if !strings.Contains(c.Message, secName) {
				return false, "TunnelReady message does not name the secret: " + c.Message
			}
			return true, ""
		}
		return false, "no TunnelReady condition yet"
	})

	// Recovery: verify that once the Secret is owned, reconcileTunnel succeeds
	// and the condition clears to DeploymentNotReady.
	var c gameplanev1alpha1.GameServer
	if err := k8sClient.Get(ctx, client.ObjectKeyFromObject(gs), &c); err != nil {
		t.Fatalf("re-fetch gameserver: %v", err)
	}
	for _, cond := range c.Status.Conditions {
		if cond.Type == "TunnelReady" && !strings.Contains(cond.Message, "fake") {
			t.Fatalf("TunnelReady message must not contain 'fake': %s", cond.Message)
		}
	}

	// Patch the Secret with OwnerReferences matching the GameServer
	secFetched := &corev1.Secret{ObjectMeta: metav1.ObjectMeta{Name: secName, Namespace: ns}}
	if err := k8sClient.Get(ctx, client.ObjectKeyFromObject(secFetched), secFetched); err != nil {
		t.Fatalf("get secret: %v", err)
	}
	secFetched.OwnerReferences = []metav1.OwnerReference{{
		APIVersion: "gameplane.local/v1alpha1",
		Kind:       "GameServer",
		Name:       c.Name,
		UID:        c.UID,
	}}
	if err := k8sClient.Update(ctx, secFetched); err != nil {
		t.Fatalf("patch secret with ownerReference: %v", err)
	}

	// Now reconcileTunnel should succeed
	eventually(t, func() (bool, string) {
		var gs2 gameplanev1alpha1.GameServer
		if err := k8sClient.Get(ctx, client.ObjectKeyFromObject(&c), &gs2); err != nil {
			return false, "re-fetch: " + err.Error()
		}
		if err := r.reconcileTunnel(ctx, &gs2, tmpl, true); err != nil {
			return false, "reconcileTunnel after fix: " + err.Error()
		}
		return true, ""
	})

	// Verify the condition now reports DeploymentNotReady instead of TunnelCredentialRefused
	var cfinal gameplanev1alpha1.GameServer
	if err := k8sClient.Get(ctx, client.ObjectKeyFromObject(&c), &cfinal); err != nil {
		t.Fatalf("final fetch: %v", err)
	}
	conds := computeTunnelConditions(&cfinal, tunnelPlan{wantTunnel: true}, nil)
	var found bool
	for _, cond := range conds {
		if cond.Type == "TunnelReady" {
			if cond.Reason != "DeploymentNotReady" {
				t.Fatalf("after credential fix, want TunnelReady reason=DeploymentNotReady, got %s", cond.Reason)
			}
			found = true
			break
		}
	}
	if !found {
		t.Fatal("TunnelReady condition not found after credential fix")
	}
}
