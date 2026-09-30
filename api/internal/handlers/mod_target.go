package handlers

import (
	"net/http"

	"github.com/go-chi/chi/v5"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"

	"github.com/ValgulNecron/gameplane/api/internal/httperr"
	"github.com/ValgulNecron/gameplane/api/internal/kube"
	"github.com/ValgulNecron/gameplane/api/internal/rbac"
	"github.com/ValgulNecron/gameplane/api/internal/scope"
)

// modTarget keeps the server, template and writes on the same selected client.
// Registry provider credentials are deliberately not part of this target: those
// remain centrally configured by the API administrator.
type modTarget struct {
	k         *kube.Client
	namespace string
	server    *unstructured.Unstructured
	template  *unstructured.Unstructured
}

func loadModTarget(w http.ResponseWriter, req *http.Request, clients *kube.Registry, local *kube.Client) (*modTarget, bool) {
	cluster := scope.DefaultCluster
	k := local
	if clients == nil {
		// Legacy mounts have no way to select a remote client.
		if rejectRemoteCluster(w, req) {
			return nil, false
		}
	} else {
		var err error
		cluster, err = scope.ResolveCluster(req, clients)
		if err != nil {
			httperr.Write(w, req, err)
			return nil, false
		}
		k, _ = clients.Get(cluster)
	}
	if k == nil {
		httperr.Write(w, req, scope.ErrForbiddenCluster)
		return nil, false
	}
	ns, ok := resolveNS(w, req)
	if !ok {
		return nil, false
	}
	name := chi.URLParam(req, "name")
	gs, err := k.Dynamic.Resource(kube.GVRs["servers"]).Namespace(ns).Get(req.Context(), name, metav1.GetOptions{})
	if err != nil {
		httperr.Write(w, req, err)
		return nil, false
	}
	// Ownership fallback authorizes a UID, not a reusable server name. Check
	// before reading its template or making a provider request on its behalf.
	if err := rbac.ValidateServerIdentity(req.Context(), cluster, ns, name, string(gs.GetUID())); err != nil {
		http.NotFound(w, req)
		return nil, false
	}
	target := &modTarget{k: k, namespace: ns, server: gs}
	templateName, _, _ := unstructured.NestedString(gs.Object, "spec", "templateRef", "name")
	if templateName != "" {
		target.template, err = k.Dynamic.Resource(kube.GVRs["templates"]).Get(req.Context(), templateName, metav1.GetOptions{})
		if err != nil {
			httperr.Write(w, req, err)
			return nil, false
		}
	}
	return target, true
}
