package handlers

import (
	"context"
	"crypto/tls"
	"crypto/x509"
	"encoding/json"
	"errors"
	"net/http"
	"net/url"

	"github.com/GameplanePanel/gameplane/api/internal/kube"
	"github.com/go-chi/chi/v5"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/client-go/util/retry"

	"github.com/GameplanePanel/gameplane/api/internal/httperr"
)

const clusterGatewayLabel = "gameplane.local/agent-gateway-credentials"

type clusterGatewayRequest struct {
	URL        string `json:"url"`
	CACert     string `json:"caCert"`
	ClientCert string `json:"clientCert"`
	ClientKey  string `json:"clientKey"`
}

func gatewaySecretName(name string) string { return "cluster-" + name + "-gateway" }

func (h clustersHandler) configureGateway(w http.ResponseWriter, req *http.Request) {
	name := chi.URLParam(req, "name")
	if name == h.reg.DefaultID() || !dnsLabelRE.MatchString(name) {
		httperr.WriteCode(w, req, http.StatusBadRequest, errors.New("select a registered remote cluster"))
		return
	}
	var in clusterGatewayRequest
	if err := json.NewDecoder(req.Body).Decode(&in); err != nil {
		httperr.WriteCode(w, req, http.StatusBadRequest, errors.New("invalid gateway configuration"))
		return
	}
	u, err := url.Parse(in.URL)
	if err != nil || u.Scheme != "https" || u.Hostname() == "" || u.User != nil || u.RawQuery != "" || u.Fragment != "" || (u.Path != "" && u.Path != "/") || u.Opaque != "" {
		httperr.WriteCode(w, req, http.StatusBadRequest, errors.New("gateway URL must be an HTTPS origin"))
		return
	}
	if _, err := tls.X509KeyPair([]byte(in.ClientCert), []byte(in.ClientKey)); err != nil {
		httperr.WriteCode(w, req, http.StatusBadRequest, errors.New("invalid gateway client certificate or key"))
		return
	}
	if !x509.NewCertPool().AppendCertsFromPEM([]byte(in.CACert)) {
		httperr.WriteCode(w, req, http.StatusBadRequest, errors.New("invalid gateway CA certificate"))
		return
	}
	registration, err := h.k.Clusters().Get(req.Context(), name, metav1.GetOptions{})
	if err != nil {
		httperr.Write(w, req, err)
		return
	}
	secretName := gatewaySecretName(name)
	if err := upsertLabelledSecret(req.Context(), h.k, h.namespace, secretName, clusterGatewayLabel, map[string]string{
		"ca.crt": in.CACert, "tls.crt": in.ClientCert, "tls.key": in.ClientKey,
	}); err != nil {
		httperr.Write(w, req, err)
		return
	}
	if err := h.updateGatewayRegistration(req.Context(), registration, map[string]any{
		"url": in.URL, "tlsSecretRef": map[string]any{"name": secretName},
	}); err != nil {
		httperr.Write(w, req, err)
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

func (h clustersHandler) removeGateway(w http.ResponseWriter, req *http.Request) {
	name := chi.URLParam(req, "name")
	registration, err := h.k.Clusters().Get(req.Context(), name, metav1.GetOptions{})
	if err != nil {
		httperr.Write(w, req, err)
		return
	}
	secretName, _, _ := unstructured.NestedString(registration.Object, "spec", "agentGateway", "tlsSecretRef", "name")
	if err := h.updateGatewayRegistration(req.Context(), registration, nil); err != nil {
		httperr.Write(w, req, err)
		return
	}

	if secretName == gatewaySecretName(name) {
		if err := deleteManagedSecret(req.Context(), h.k, h.namespace, secretName, clusterGatewayLabel); err != nil && !apierrors.IsNotFound(err) {
			httperr.Write(w, req, err)
			return
		}
	}
	w.WriteHeader(http.StatusNoContent)
}

// A health status update may race configuration. Retry against fresh metadata,
// but never carry a gateway write across deletion/recreation of a registration.
func (h clustersHandler) updateGatewayRegistration(ctx context.Context, original *unstructured.Unstructured, gateway map[string]any) error {
	return retry.RetryOnConflict(retry.DefaultRetry, func() error {
		current, err := h.k.Clusters().Get(ctx, original.GetName(), metav1.GetOptions{})
		if err != nil {
			return err
		}
		if current.GetUID() != original.GetUID() {
			return apierrors.NewNotFound(kube.GVRCluster.GroupResource(), original.GetName())
		}
		if gateway == nil {
			unstructured.RemoveNestedField(current.Object, "spec", "agentGateway")
		} else if err := unstructured.SetNestedMap(current.Object, gateway, "spec", "agentGateway"); err != nil {
			return err
		}
		_, err = h.k.Clusters().Update(ctx, current, metav1.UpdateOptions{})
		return err
	})
}
