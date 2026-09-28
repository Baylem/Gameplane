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
	if err := r.reconcileTunnel(ctx, gs, tmpl, true); err == nil {
		t.Fatal("reconcileTunnel: want error for unowned credentials secret, got nil")
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
}
