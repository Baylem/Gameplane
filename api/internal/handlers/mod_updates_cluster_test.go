package handlers

import (
	"context"
	"errors"
	"net/http"
	"testing"

	"github.com/ValgulNecron/gameplane/api/internal/kube"
	"github.com/go-chi/chi/v5"
)

type clusterModLister struct {
	gotCluster string
	localCalls int
}

func (l *clusterModLister) GetJSON(context.Context, string, string, string, any) error {
	l.localCalls++
	return errors.New("local agent must not be queried")
}
func (l *clusterModLister) GetJSONForCluster(_ context.Context, cluster, name, namespace, path string, out any) error {
	l.gotCluster = cluster
	if name != "alpha" || namespace != "gameplane-games" || path != "/mods" {
		return errors.New("unexpected target")
	}
	*out.(*[]installedMod) = []installedMod{}
	return nil
}

func TestModUpdatesRoutesAgentAndTemplateToRemoteCluster(t *testing.T) {
	clients := kube.NewRegistry("local")
	// The home cluster intentionally has no server/template. A fallback fails.
	clients.Set("local", fakeKubeClient())
	clients.Set("remote", updatesFixtureKube())
	lister := &clusterModLister{}
	router := chi.NewRouter()
	MountModUpdatesWithRegistry(router, clients, fakeSet{p: &fakeVersionsProvider{}}, lister)
	response := do(t, router, http.MethodGet, "/servers/alpha/mods/updates?cluster=remote", nil)
	if response.Code != http.StatusOK {
		t.Fatalf("status = %d: %s", response.Code, response.Body)
	}
	if lister.gotCluster != "remote" || lister.localCalls != 0 {
		t.Fatalf("agent selection = %+v", lister)
	}
}

func TestModUpdatesRemoteCannotUseLocalOnlyAgent(t *testing.T) {
	clients := kube.NewRegistry("local")
	clients.Set("local", updatesFixtureKube())
	clients.Set("remote", updatesFixtureKube())
	router := chi.NewRouter()
	MountModUpdatesWithRegistry(router, clients, fakeSet{p: &fakeVersionsProvider{}}, &fakeModLister{})
	response := do(t, router, http.MethodGet, "/servers/alpha/mods/updates?cluster=remote", nil)
	if response.Code != http.StatusServiceUnavailable {
		t.Fatalf("status = %d", response.Code)
	}
}
