//go:build e2e

package e2e

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"os/exec"
	"strings"
	"testing"
	"time"

	"github.com/gopacket/gopacket/pcapgo"
	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
)

// Dedicated kind bucket on both architectures: it upgrades namespace policy,
// checks real PSA admission, and waits for kubelet to rotate an actual token.
func TestRestrictedWorkloads_NodeCaptureAndTokenRotation(t *testing.T) {
	ctx := context.Background()
	const ns, diagnosticNS, name, template = "gameplane-games", "gameplane-diagnostics", "restricted-game", "restricted-busybox"
	applyBusyboxTemplate(t, template)
	tmpl, err := envInstance.Dyn.Resource(gameTemplateGVR).Get(ctx, template, metav1.GetOptions{})
	if err != nil {
		t.Fatal(err)
	}
	_ = unstructured.SetNestedMap(tmpl.Object, map[string]any{"runAsUser": int64(1000), "runAsGroup": int64(1000), "fsGroup": int64(1000)}, "spec", "security")
	_ = unstructured.SetNestedSlice(tmpl.Object, []any{map[string]any{"path": "cfg/profile.txt", "template": "restricted-config\n"}}, "spec", "configFiles")
	if _, err = envInstance.Dyn.Resource(gameTemplateGVR).Update(ctx, tmpl, metav1.UpdateOptions{}); err != nil {
		t.Fatal(err)
	}
	applyBusyboxGameServer(t, ns, name, template)
	waitGameContainerReady(t, ns, name+"-0", 3*time.Minute)
	before, err := envInstance.K8s.CoreV1().Pods(ns).Get(ctx, name+"-0", metav1.GetOptions{})
	if err != nil {
		t.Fatal(err)
	}
	if out, err := envInstance.KubectlExec(t, ns, "pod/"+name+"-0", "sh", "-c", "echo upgrade-marker > /data/marker && sync"); err != nil {
		t.Fatalf("write data: %v %s", err, out)
	}
	helmUpgradeReuse(ctx, t, "podSecurity.enforceRestricted=true", "operator.agentTokenExpirationSeconds=600", "capture.enabled=true", "capture.nodeDiagnostics.enabled=true")
	envInstance.Eventually(t, 3*time.Minute, func() (bool, string) {
		p, err := envInstance.K8s.CoreV1().Pods(ns).Get(ctx, name+"-0", metav1.GetOptions{})
		if err != nil {
			return false, err.Error()
		}
		return p.UID != before.UID && p.Annotations["gameplane.local/workload-security"] == "restricted", "waiting for profile upgrade"
	})
	waitGameContainerReady(t, ns, name+"-0", 3*time.Minute)
	requireAgentReady(t, ns, name)
	pod, err := envInstance.K8s.CoreV1().Pods(ns).Get(ctx, name+"-0", metav1.GetOptions{})
	if err != nil {
		t.Fatal(err)
	}
	if pod.Spec.AutomountServiceAccountToken == nil || *pod.Spec.AutomountServiceAccountToken || len(pod.Spec.InitContainers) != 1 || len(pod.Spec.EphemeralContainers) != 0 {
		t.Fatalf("unexpected restricted pod: %+v", pod.Spec)
	}
	for _, c := range append(pod.Spec.Containers, pod.Spec.InitContainers...) {
		sc := c.SecurityContext
		if sc == nil || sc.RunAsNonRoot == nil || !*sc.RunAsNonRoot || sc.AllowPrivilegeEscalation == nil || *sc.AllowPrivilegeEscalation || sc.Capabilities == nil || len(sc.Capabilities.Add) != 0 || len(sc.Capabilities.Drop) != 1 || sc.Capabilities.Drop[0] != "ALL" || sc.SeccompProfile == nil || sc.SeccompProfile.Type != corev1.SeccompProfileTypeRuntimeDefault {
			t.Fatalf("%s is not Restricted", c.Name)
		}
		tokenMounted := false
		for _, mount := range c.VolumeMounts {
			if mount.Name == "agent-api-token" {
				tokenMounted = true
			}
		}
		if tokenMounted != (c.Name == "agent") {
			t.Fatalf("%s token mount=%t", c.Name, tokenMounted)
		}
	}
	if out, err := envInstance.KubectlExec(t, ns, "pod/"+name+"-0", "sh", "-c", "test ! -e /var/run/secrets/kubernetes.io/serviceaccount/token && test $(id -u) = 1000 && grep upgrade-marker /data/marker && grep restricted-config /data/cfg/profile.txt && echo writable >> /data/marker"); err != nil {
		t.Fatalf("identity/config/data/token: %v %s", err, out)
	}
	// A root container is rejected by the real admission controller.
	zero := int64(0)
	_, err = envInstance.K8s.CoreV1().Pods(ns).Create(ctx, &corev1.Pod{ObjectMeta: metav1.ObjectMeta{Name: "rejected-root"}, Spec: corev1.PodSpec{Containers: []corev1.Container{{Name: "root", Image: "busybox:1.36", SecurityContext: &corev1.SecurityContext{RunAsUser: &zero}}}}}, metav1.CreateOptions{})
	if !apierrors.IsForbidden(err) || !strings.Contains(err.Error(), "PodSecurity") {
		t.Fatalf("PSA did not reject root: %v", err)
	}
	// Hash only: never return the bearer token in CI output. This is the kind
	// host fixture inspecting kubelet's projection, not an extra game mount.
	tokenPath := "/var/lib/kubelet/pods/" + string(pod.UID) + "/volumes/kubernetes.io~projected/agent-api-token/token"
	tokenHash := func() (string, error) {
		cctx, cancel := context.WithTimeout(ctx, 10*time.Second)
		defer cancel()
		out, err := exec.CommandContext(cctx, "docker", "exec", pod.Spec.NodeName, "sha256sum", tokenPath).CombinedOutput()
		if err != nil {
			return "", fmt.Errorf("hash projected token: %w", err)
		}
		fields := strings.Fields(string(out))
		if len(fields) == 0 {
			return "", fmt.Errorf("empty token hash")
		}
		return fields[0], nil
	}
	initialHash, err := tokenHash()
	if err != nil {
		t.Fatal(err)
	}
	initialTokenObserved := time.Now()
	applyBusyboxTemplate(t, "restricted-incompatible")
	applyBusyboxGameServer(t, ns, "restricted-incompatible", "restricted-incompatible")
	envInstance.Eventually(t, time.Minute, func() (bool, string) {
		gs, err := envInstance.Dyn.Resource(gameServerGVR).Namespace(ns).Get(ctx, "restricted-incompatible", metav1.GetOptions{})
		if err != nil {
			return false, err.Error()
		}
		phase, _, _ := unstructured.NestedString(gs.Object, "status", "phase")
		if phase != "Failed" {
			return false, "incompatible template not rejected"
		}
		_, err = envInstance.K8s.AppsV1().StatefulSets(ns).Get(ctx, "restricted-incompatible", metav1.GetOptions{})
		return apierrors.IsNotFound(err), "incompatible template reached workload admission"
	})
	envInstance.BootstrapAdmin(t, adminUsername, adminPassword)
	cli := envInstance.APIClient(t, adminUsername, adminPassword)
	defer cli.Close()
	post := func(path string, body any, want int) []byte {
		resp, raw, err := cli.Post(path, body)
		if err != nil {
			t.Fatal(err)
		}
		defer func() { _ = resp.Body.Close() }()
		if resp.StatusCode != want {
			t.Fatalf("POST %s: %d %s", path, resp.StatusCode, raw)
		}
		return raw
	}
	post("/servers/"+name+":capture-enable", nil, http.StatusOK)
	post("/servers/"+name+":capture-node-enable", nil, http.StatusOK)
	envInstance.Eventually(t, time.Minute, func() (bool, string) {
		gs, err := envInstance.Dyn.Resource(gameServerGVR).Namespace(ns).Get(ctx, name, metav1.GetOptions{})
		if err != nil {
			return false, err.Error()
		}
		ready, _, _ := unstructured.NestedBool(gs.Object, "status", "capture", "ready")
		return ready, "capture not ready"
	})
	start := func(duration int) *unstructured.Unstructured {
		raw := post("/servers/"+name+":capture-start", map[string]any{"filter": "tcp port 12345", "maxDurationSeconds": duration, "maxSizeBytes": int64(4 << 20), "ttlSecondsAfterFinished": 3600}, http.StatusAccepted)
		var response struct {
			CaptureID string `json:"captureId"`
		}
		if err := json.Unmarshal(raw, &response); err != nil || response.CaptureID == "" {
			t.Fatalf("capture response: %s %v", raw, err)
		}
		nc, err := envInstance.Dyn.Resource(networkCaptureGVR).Namespace(ns).Get(ctx, response.CaptureID, metav1.GetOptions{})
		if err != nil {
			t.Fatal(err)
		}
		t.Cleanup(func() {
			_ = envInstance.Dyn.Resource(networkCaptureGVR).Namespace(ns).Delete(context.Background(), nc.GetName(), metav1.DeleteOptions{})
		})
		envInstance.Eventually(t, 2*time.Minute, func() (bool, string) {
			pods, err := envInstance.K8s.CoreV1().Pods(diagnosticNS).List(ctx, metav1.ListOptions{LabelSelector: "gameplane.local/capture-uid=" + string(nc.GetUID()) + ",gameplane.local/capture-component=worker"})
			if err != nil {
				return false, err.Error()
			}
			for _, p := range pods.Items {
				if p.Status.Phase == corev1.PodRunning {
					if p.Spec.NodeName != pod.Spec.NodeName || !p.Spec.HostPID || p.Spec.AutomountServiceAccountToken == nil || *p.Spec.AutomountServiceAccountToken {
						t.Fatal("worker node/privilege/token mismatch")
					}
					return true, ""
				}
			}
			return false, "waiting for bound worker"
		})
		return nc
	}
	nc := start(60)
	traffic := &corev1.Pod{ObjectMeta: metav1.ObjectMeta{Name: "restricted-capture-traffic"}, Spec: corev1.PodSpec{RestartPolicy: corev1.RestartPolicyNever, Containers: []corev1.Container{{Name: "traffic", Image: "busybox:1.36", Command: []string{"sh", "-c", "for i in $(seq 1 20); do nc -w 1 " + pod.Status.PodIP + " 12345 </dev/null || true; sleep 1; done"}}}}}
	if _, err := envInstance.K8s.CoreV1().Pods("gameplane-system").Create(ctx, traffic, metav1.CreateOptions{}); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		_ = envInstance.K8s.CoreV1().Pods("gameplane-system").Delete(context.Background(), traffic.Name, metav1.DeleteOptions{})
	})
	envInstance.Eventually(t, 3*time.Minute, func() (bool, string) {
		live, err := envInstance.Dyn.Resource(networkCaptureGVR).Namespace(ns).Get(ctx, nc.GetName(), metav1.GetOptions{})
		if err != nil {
			return false, err.Error()
		}
		phase, _, _ := unstructured.NestedString(live.Object, "status", "phase")
		message, _, _ := unstructured.NestedString(live.Object, "status", "message")
		return phase == "Completed", phase + ": " + message
	})
	workers, err := envInstance.K8s.CoreV1().Pods(diagnosticNS).List(ctx, metav1.ListOptions{LabelSelector: "gameplane.local/capture-component=worker"})
	if err != nil || len(workers.Items) != 0 {
		t.Fatalf("worker survived completion: %v %+v", err, workers)
	}
	resp, raw, err := cli.Get("/servers/" + name + ":capture-file?id=" + nc.GetName())
	if err != nil {
		t.Fatal(err)
	}
	_ = resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("download: %d %s", resp.StatusCode, raw)
	}
	reader, err := pcapgo.NewNgReader(bytes.NewReader(raw), pcapgo.DefaultNgReaderOptions)
	if err != nil {
		t.Fatal(err)
	}
	if _, _, err := reader.ReadPacketData(); err != nil {
		t.Fatalf("capture has no traffic: %v", err)
	}
	assertUnchanged := func() {
		live, err := envInstance.K8s.CoreV1().Pods(ns).Get(ctx, name+"-0", metav1.GetOptions{})
		if err != nil {
			t.Fatal(err)
		}
		if live.UID != pod.UID || len(live.Spec.EphemeralContainers) != 0 {
			t.Fatal("capture restarted or injected into game pod")
		}
		for i, c := range live.Status.ContainerStatuses {
			if c.RestartCount != pod.Status.ContainerStatuses[i].RestartCount {
				t.Fatal("capture restarted game/agent container")
			}
		}
	}
	assertUnchanged()
	// Revocation must drain a running host-capable worker before releasing its lock.
	second := start(300)
	post("/servers/"+name+":capture-node-disable", nil, http.StatusOK)
	envInstance.Eventually(t, time.Minute, func() (bool, string) {
		pods, err := envInstance.K8s.CoreV1().Pods(diagnosticNS).List(ctx, metav1.ListOptions{LabelSelector: "gameplane.local/capture-component=worker"})
		if err != nil {
			return false, err.Error()
		}
		return len(pods.Items) == 0, "worker survives revocation"
	})
	assertUnchanged()
	// Real kubelet rotation plus a later status patch proves client-go reloads
	// the projected file. Waiting for a heartbeat alone would miss stale tokens.
	envInstance.Eventually(t, 12*time.Minute, func() (bool, string) {
		current, err := tokenHash()
		if err != nil {
			return false, err.Error()
		}
		return current != initialHash && time.Since(initialTokenObserved) > 11*time.Minute, "waiting for rotation and original token expiry"
	})
	rotatedAt := time.Now()
	envInstance.Eventually(t, time.Minute, func() (bool, string) {
		gs, err := envInstance.Dyn.Resource(gameServerGVR).Namespace(ns).Get(ctx, name, metav1.GetOptions{})
		if err != nil {
			return false, err.Error()
		}
		s, _, _ := unstructured.NestedString(gs.Object, "status", "agent", "lastHeartbeat")
		heartbeat, err := time.Parse(time.RFC3339, s)
		return err == nil && heartbeat.After(rotatedAt), "no post-rotation heartbeat"
	})
	// Deleting the owning server garbage-collects both retained artifacts and
	// the canceled capture across namespaces through each capture's finalizer.
	if err := envInstance.Dyn.Resource(gameServerGVR).Namespace(ns).Delete(ctx, name, metav1.DeleteOptions{}); err != nil {
		t.Fatal(err)
	}
	for _, capture := range []*unstructured.Unstructured{nc, second} {
		selector := "gameplane.local/capture-uid=" + string(capture.GetUID())
		envInstance.Eventually(t, 2*time.Minute, func() (bool, string) {
			_, err := envInstance.Dyn.Resource(networkCaptureGVR).Namespace(ns).Get(ctx, capture.GetName(), metav1.GetOptions{})
			if !apierrors.IsNotFound(err) {
				return false, "capture finalizer still present"
			}
			pods, err := envInstance.K8s.CoreV1().Pods(diagnosticNS).List(ctx, metav1.ListOptions{LabelSelector: selector})
			if err != nil {
				return false, err.Error()
			}
			pvcs, err := envInstance.K8s.CoreV1().PersistentVolumeClaims(diagnosticNS).List(ctx, metav1.ListOptions{LabelSelector: selector})
			if err != nil {
				return false, err.Error()
			}
			return len(pods.Items) == 0 && len(pvcs.Items) == 0, "diagnostic storage/pods remain"
		})
	}
}
