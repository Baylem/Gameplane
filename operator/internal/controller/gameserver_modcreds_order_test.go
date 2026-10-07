package controller

import (
	"reflect"
	"testing"
)

func TestModCredentialPodTemplateIsStable(t *testing.T) {
	creds := resolvedModCreds{providers: map[string]string{
		"zeta": "zeta-secret", "alpha": "alpha-secret", "middle": "middle-secret",
	}}
	volumes := modCredsVolumes(creds)
	mounts := modCredVolumeMounts(creds)
	for i, name := range []string{"alpha", "middle", "zeta"} {
		if volumes[i].Name != "mod-creds-"+name || mounts[i].Name != volumes[i].Name ||
			volumes[i].Secret.SecretName != name+"-secret" {
			t.Fatalf("credential %d has inconsistent volume/mount order", i)
		}
	}
	// Repeated reconciles must not change either ordered PodSpec slice.
	for range 100 {
		if !reflect.DeepEqual(volumes, modCredsVolumes(creds)) || !reflect.DeepEqual(mounts, modCredVolumeMounts(creds)) {
			t.Fatal("unchanged credentials changed the pod template")
		}
	}
}
