package handlers

import (
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/x509"
	"encoding/pem"
	"math/big"
	"net/http"
	"strings"
	"testing"
	"time"

	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"

	"github.com/GameplanePanel/gameplane/api/internal/controlplane"
)

func gatewayCertificateRequest(t *testing.T) clusterGatewayRequest {
	t.Helper()
	key, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	if err != nil {
		t.Fatal(err)
	}
	template := &x509.Certificate{
		SerialNumber: big.NewInt(1), NotBefore: time.Now().Add(-time.Hour), NotAfter: time.Now().Add(time.Hour),
		KeyUsage:    x509.KeyUsageDigitalSignature | x509.KeyUsageCertSign,
		ExtKeyUsage: []x509.ExtKeyUsage{x509.ExtKeyUsageClientAuth}, IsCA: true, BasicConstraintsValid: true,
	}
	der, err := x509.CreateCertificate(rand.Reader, template, template, &key.PublicKey, key)
	if err != nil {
		t.Fatal(err)
	}
	privateKey, err := x509.MarshalECPrivateKey(key)
	if err != nil {
		t.Fatal(err)
	}
	certificate := string(pem.EncodeToMemory(&pem.Block{Type: "CERTIFICATE", Bytes: der}))
	return clusterGatewayRequest{URL: "https://gateway.example.invalid:8443", CACert: certificate, ClientCert: certificate, ClientKey: string(pem.EncodeToMemory(&pem.Block{Type: "EC PRIVATE KEY", Bytes: privateKey}))}
}

func TestStandaloneGatewayConfigurationEncryptsCredentialsAndStoresOnlyReference(t *testing.T) {
	store, home, reg, keyPath := standaloneHandlerStore(t)
	if _, err := home.Clusters().Create(t.Context(), newCluster("remote", map[string]any{}, nil), metav1.CreateOptions{}); err != nil {
		t.Fatal(err)
	}
	in := gatewayCertificateRequest(t)
	router := standaloneClustersRouter(home, reg)
	configured := doClusters(t, router, http.MethodPut, "/clusters/remote/gateway", in)
	if configured.Code != http.StatusNoContent || configured.Body.Len() != 0 {
		t.Fatalf("configure: %d %s", configured.Code, configured.Body)
	}
	assertManagementCiphertext(t, store, gatewaySecretName("remote"), in.ClientKey, in.ClientCert)
	home, err := controlplane.New(t.Context(), store, keyPath)
	if err != nil {
		t.Fatal(err)
	}
	stored, err := home.Secrets(standaloneTestNamespace).Get(t.Context(), gatewaySecretName("remote"), metav1.GetOptions{})
	if err != nil {
		t.Fatal(err)
	}
	if string(stored.Data["tls.key"]) != in.ClientKey || string(stored.Data["tls.crt"]) != in.ClientCert || string(stored.Data["ca.crt"]) != in.CACert {
		t.Fatal("gateway credentials did not survive reopen")
	}
	if stored.Labels[clusterGatewayLabel] != "true" || stored.Labels[ManagedByLabel] != managedByValue {
		t.Fatal("gateway credential ownership labels missing")
	}
	registration, err := home.Clusters().Get(t.Context(), "remote", metav1.GetOptions{})
	if err != nil {
		t.Fatal(err)
	}
	gateway, found, err := unstructured.NestedMap(registration.Object, "spec", "agentGateway")
	if err != nil || !found || len(gateway) != 2 || gateway["url"] != in.URL {
		t.Fatalf("unexpected gateway registration: %+v %v", gateway, err)
	}
	ref, _, _ := unstructured.NestedString(registration.Object, "spec", "agentGateway", "tlsSecretRef", "name")
	if ref != gatewaySecretName("remote") {
		t.Fatalf("wrong credential reference: %s", ref)
	}
	var payload string
	if err := store.DB.QueryRowContext(t.Context(), `SELECT payload FROM management_objects WHERE kind = 'clusters' AND name = 'remote'`).Scan(&payload); err != nil {
		t.Fatal(err)
	}
	if strings.Contains(payload, "PRIVATE KEY") || strings.Contains(payload, "CERTIFICATE") || strings.Contains(payload, "clientKey") {
		t.Fatal("registration contains credential material")
	}
	listed := doClusters(t, router, http.MethodGet, "/clusters/", nil)
	if listed.Code != http.StatusOK || strings.Contains(listed.Body.String(), "gateway.example") || strings.Contains(listed.Body.String(), "PRIVATE KEY") {
		t.Fatalf("gateway leaked in discovery: %d %s", listed.Code, listed.Body)
	}
}

func TestGatewayRejectsNonHTTPSOriginsAndInvalidPEMWithoutWritingSecrets(t *testing.T) {
	valid := gatewayCertificateRequest(t)
	for _, badURL := range []string{"http://gateway.example.invalid", "https://user:password@gateway.example.invalid", "https://gateway.example.invalid/path", "https://gateway.example.invalid?token=private", "https://gateway.example.invalid#fragment", "https://"} {
		t.Run(badURL, func(t *testing.T) {
			_, home, reg, _ := standaloneHandlerStore(t)
			if _, err := home.Clusters().Create(t.Context(), newCluster("remote", map[string]any{}, nil), metav1.CreateOptions{}); err != nil {
				t.Fatal(err)
			}
			in := valid
			in.URL = badURL
			rr := doClusters(t, standaloneClustersRouter(home, reg), http.MethodPut, "/clusters/remote/gateway", in)
			if rr.Code != http.StatusBadRequest {
				t.Fatalf("invalid URL accepted: %d %s", rr.Code, rr.Body)
			}
			if _, err := home.Secrets(standaloneTestNamespace).Get(t.Context(), gatewaySecretName("remote"), metav1.GetOptions{}); !apierrors.IsNotFound(err) {
				t.Fatalf("invalid URL wrote credentials: %v", err)
			}
		})
	}
	for _, field := range []string{"ca", "certificate", "key"} {
		t.Run(field, func(t *testing.T) {
			_, home, reg, _ := standaloneHandlerStore(t)
			if _, err := home.Clusters().Create(t.Context(), newCluster("remote", map[string]any{}, nil), metav1.CreateOptions{}); err != nil {
				t.Fatal(err)
			}
			in := valid
			switch field {
			case "ca":
				in.CACert = "private-invalid-ca"
			case "certificate":
				in.ClientCert = "private-invalid-cert"
			case "key":
				in.ClientKey = "private-invalid-key"
			}
			rr := doClusters(t, standaloneClustersRouter(home, reg), http.MethodPut, "/clusters/remote/gateway", in)
			if rr.Code != http.StatusBadRequest || strings.Contains(rr.Body.String(), "private-invalid") {
				t.Fatalf("invalid PEM response: %d %s", rr.Code, rr.Body)
			}
			if _, err := home.Secrets(standaloneTestNamespace).Get(t.Context(), gatewaySecretName("remote"), metav1.GetOptions{}); !apierrors.IsNotFound(err) {
				t.Fatalf("invalid PEM wrote credentials: %v", err)
			}
		})
	}
}

func TestGatewayRemovalDeletesOnlyGeneratedSecretWithBothOwnershipLabels(t *testing.T) {
	for _, tc := range []struct {
		name, secretName string
		labels           map[string]string
		deleted          bool
	}{
		{"managed", gatewaySecretName("remote"), map[string]string{clusterGatewayLabel: "true", ManagedByLabel: managedByValue}, true},
		{"missing-feature-label", gatewaySecretName("remote"), map[string]string{ManagedByLabel: managedByValue}, false},
		{"gitops-secret", gatewaySecretName("remote"), map[string]string{clusterGatewayLabel: "true"}, false},
		{"external-reference", "externally-owned-gateway", map[string]string{clusterGatewayLabel: "true", ManagedByLabel: managedByValue}, false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			_, home, reg, _ := standaloneHandlerStore(t)
			_, err := home.Clusters().Create(t.Context(), newCluster("remote", map[string]any{"agentGateway": map[string]any{"url": "https://gateway.example.invalid", "tlsSecretRef": map[string]any{"name": tc.secretName}}}, nil), metav1.CreateOptions{})
			if err != nil {
				t.Fatal(err)
			}
			_, err = home.Secrets(standaloneTestNamespace).Create(t.Context(), &corev1.Secret{ObjectMeta: metav1.ObjectMeta{Name: tc.secretName, Labels: tc.labels}, StringData: map[string]string{"tls.key": "private-key-value"}}, metav1.CreateOptions{})
			if err != nil {
				t.Fatal(err)
			}
			rr := doClusters(t, standaloneClustersRouter(home, reg), http.MethodDelete, "/clusters/remote/gateway", nil)
			if rr.Code != http.StatusNoContent {
				t.Fatalf("remove: %d %s", rr.Code, rr.Body)
			}
			_, err = home.Secrets(standaloneTestNamespace).Get(t.Context(), tc.secretName, metav1.GetOptions{})
			if tc.deleted && !apierrors.IsNotFound(err) {
				t.Fatalf("managed secret survived: %v", err)
			}
			if !tc.deleted && err != nil {
				t.Fatalf("unowned secret removed: %v", err)
			}
			registration, err := home.Clusters().Get(t.Context(), "remote", metav1.GetOptions{})
			if err != nil {
				t.Fatal(err)
			}
			if _, found, err := unstructured.NestedMap(registration.Object, "spec", "agentGateway"); err != nil || found {
				t.Fatalf("gateway reference survived: %v", err)
			}
		})
	}
}
