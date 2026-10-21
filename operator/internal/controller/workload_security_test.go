package controller

import (
	"testing"

	gameplanev1alpha1 "github.com/GameplanePanel/gameplane/operator/api/v1alpha1"
	corev1 "k8s.io/api/core/v1"
)

func TestRestrictedTemplateIdentity(t *testing.T) {
	for _, test := range []struct {
		name         string
		uid, gid, fs *int64
		valid        bool
	}{
		{"absent", nil, nil, nil, false},
		{"root", ptrTo(int64(0)), nil, ptrTo(int64(1000)), false},
		{"missing-volume-group", ptrTo(int64(1000)), nil, nil, false},
		{"root-group", ptrTo(int64(1000)), ptrTo(int64(0)), ptrTo(int64(1000)), false},
		{"compatible", ptrTo(int64(1000)), ptrTo(int64(1000)), ptrTo(int64(1000)), true},
		{"default-primary-group", ptrTo(int64(1000)), nil, ptrTo(int64(1000)), true},
	} {
		t.Run(test.name, func(t *testing.T) {
			tmpl := &gameplanev1alpha1.GameTemplate{}
			tmpl.Spec.Security = &gameplanev1alpha1.GameSecuritySpec{RunAsUser: test.uid, RunAsGroup: test.gid, FSGroup: test.fs}
			if err := validateRestrictedTemplate(tmpl); (err == nil) != test.valid {
				t.Fatalf("valid=%t: %v", test.valid, err)
			}
		})
	}
}

func TestWorkloadTokenOnlyMountedInAgent(t *testing.T) {
	tmpl := &gameplanev1alpha1.GameTemplate{}
	tmpl.Spec.Security = &gameplanev1alpha1.GameSecuritySpec{RunAsUser: ptrTo(int64(1000)), FSGroup: ptrTo(int64(1000))}
	gs := &gameplanev1alpha1.GameServer{}
	pod := corev1.PodSpec{SecurityContext: gamePodSecurityContext(tmpl), Containers: []corev1.Container{{Name: "game"}, buildAgentContainer(gs, tmpl, nil, "agent", "", "")}, InitContainers: []corev1.Container{buildConfigInitContainer("", tmpl)}}
	applyWorkloadSecurity(&pod, tmpl, true, 3600)
	if pod.AutomountServiceAccountToken == nil || *pod.AutomountServiceAccountToken {
		t.Fatal("automatic credentials exposed")
	}
	for _, c := range append(pod.Containers, pod.InitContainers...) {
		sc := c.SecurityContext
		if sc == nil || sc.RunAsNonRoot == nil || !*sc.RunAsNonRoot || sc.AllowPrivilegeEscalation == nil || *sc.AllowPrivilegeEscalation || sc.SeccompProfile == nil || sc.SeccompProfile.Type != corev1.SeccompProfileTypeRuntimeDefault || sc.Capabilities == nil || len(sc.Capabilities.Drop) != 1 || sc.Capabilities.Drop[0] != "ALL" || len(sc.Capabilities.Add) != 0 {
			t.Fatalf("%s not Restricted: %+v", c.Name, sc)
		}
		mounted := false
		for _, m := range c.VolumeMounts {
			if m.Name == agentTokenVolumeName {
				mounted = true
				if !m.ReadOnly || m.MountPath != "/var/run/secrets/kubernetes.io/serviceaccount" {
					t.Fatalf("unsafe token mount: %+v", m)
				}
			}
		}
		if mounted != (c.Name == "agent") {
			t.Fatalf("%s token mount=%t", c.Name, mounted)
		}
	}
	var projection *corev1.ProjectedVolumeSource
	for _, v := range pod.Volumes {
		if v.Name == agentTokenVolumeName {
			projection = v.Projected
		}
	}
	if projection == nil || len(projection.Sources) != 3 || projection.Sources[0].ServiceAccountToken == nil || *projection.Sources[0].ServiceAccountToken.ExpirationSeconds != 3600 {
		t.Fatalf("missing rotating token projection: %+v", projection)
	}
}
