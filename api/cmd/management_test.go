package main

import (
	"path/filepath"
	"testing"

	"github.com/GameplanePanel/gameplane/api/internal/db"
)

func TestStandaloneManagementIgnoresKubeconfig(t *testing.T) {
	t.Setenv("KUBECONFIG", filepath.Join(t.TempDir(), "does-not-exist"))
	t.Setenv("KUBERNETES_SERVICE_HOST", "")
	store, err := db.Open(t.Context(), "sqlite", "file::memory:?cache=shared")
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = store.Close() })
	if err := store.Migrate(t.Context()); err != nil {
		t.Fatal(err)
	}
	management, reg, err := managementClients(t.Context(), config{standalone: true, panelKeyFile: filepath.Join(t.TempDir(), "panel.key")}, store)
	if err != nil {
		t.Fatal(err)
	}
	if management.Typed != nil || management.Dynamic != nil || management.Config != nil {
		t.Fatal("standalone management acquired a Kubernetes client")
	}
	if reg.Default() != nil || len(reg.IDs()) != 0 || reg.Management() != management {
		t.Fatal("standalone registered a local workload cluster")
	}
}

func TestStandaloneRejectsLocalCredentialMinting(t *testing.T) {
	_, _, err := managementClients(t.Context(), config{standalone: true, clusterOps: true}, nil)
	if err == nil {
		t.Fatal("standalone must reject local cluster operations")
	}
}
