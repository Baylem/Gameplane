package kubeconfig

import (
	"bytes"
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/x509"
	"encoding/pem"
	"math/big"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
	clientcmdapi "k8s.io/client-go/tools/clientcmd/api"
)

func testConfig() *clientcmdapi.Config {
	return &clientcmdapi.Config{
		Clusters: map[string]*clientcmdapi.Cluster{
			"selected": {Server: "https://selected.example:6443"},
			"other":    {Server: "https://other.example:6443"},
		},
		AuthInfos: map[string]*clientcmdapi.AuthInfo{
			"selected": {Token: "selected-token"},
			"other":    {Token: "other-token"},
		},
		Contexts: map[string]*clientcmdapi.Context{
			"selected": {Cluster: "selected", AuthInfo: "selected", Namespace: "games"},
			"other":    {Cluster: "other", AuthInfo: "other"},
		},
		CurrentContext: "selected",
	}
}

func configBytes(t *testing.T, cfg *clientcmdapi.Config) []byte {
	t.Helper()
	data, err := clientcmd.Write(*cfg)
	if err != nil {
		t.Fatalf("encode kubeconfig: %v", err)
	}
	return data
}

// Removing any credential guard, or checking only the current context, must fail.
func TestRESTConfigRejectsProcessLocalCredentials(t *testing.T) {
	for _, entry := range []string{"selected", "other", "unreferenced"} {
		for _, tc := range []struct {
			field string
			set   func(*clientcmdapi.Cluster, *clientcmdapi.AuthInfo)
		}{
			{"certificate-authority", func(c *clientcmdapi.Cluster, _ *clientcmdapi.AuthInfo) { c.CertificateAuthority = "/private/ca.pem" }},
			{"client-certificate", func(_ *clientcmdapi.Cluster, a *clientcmdapi.AuthInfo) { a.ClientCertificate = "/private/cert.pem" }},
			{"client-key", func(_ *clientcmdapi.Cluster, a *clientcmdapi.AuthInfo) { a.ClientKey = "/private/key.pem" }},
			{"tokenFile", func(_ *clientcmdapi.Cluster, a *clientcmdapi.AuthInfo) { a.TokenFile = "/private/token" }},
			{"exec", func(_ *clientcmdapi.Cluster, a *clientcmdapi.AuthInfo) {
				a.Exec = &clientcmdapi.ExecConfig{Command: "private-command", APIVersion: "client.authentication.k8s.io/v1", InteractiveMode: clientcmdapi.NeverExecInteractiveMode}
			}},
			{"auth-provider", func(_ *clientcmdapi.Cluster, a *clientcmdapi.AuthInfo) {
				a.AuthProvider = &clientcmdapi.AuthProviderConfig{Name: "private-provider", Config: map[string]string{"cmd-path": "private-command"}}
			}},
		} {
			t.Run(entry+"/"+tc.field, func(t *testing.T) {
				cfg := testConfig()
				if entry == "unreferenced" {
					cfg.Clusters[entry] = &clientcmdapi.Cluster{Server: "https://unused.example"}
					cfg.AuthInfos[entry] = &clientcmdapi.AuthInfo{}
				}
				tc.set(cfg.Clusters[entry], cfg.AuthInfos[entry])
				result, err := RESTConfig(configBytes(t, cfg))
				if err == nil || result != nil {
					t.Fatalf("expected rejection with no config, got config=%v error=%v", result, err)
				}
				if !strings.Contains(err.Error(), tc.field) {
					t.Fatalf("expected policy rejection for %s, got %v", tc.field, err)
				}
				if strings.Contains(err.Error(), "private") {
					t.Fatalf("rejection exposed credential details: %v", err)
				}
			})
		}
	}
}

func TestRESTConfigRejectsTokenFilesBeforeReading(t *testing.T) {
	path := filepath.Join(t.TempDir(), "token")
	if err := os.WriteFile(path, []byte("process-secret"), 0600); err != nil {
		t.Fatal(err)
	}
	// Both readable and nonexistent files must produce the same policy rejection,
	// rather than loading credentials or returning a filesystem error.
	for _, tokenPath := range []string{path, path + "-missing"} {
		cfg := testConfig()
		cfg.AuthInfos["selected"].Token = ""
		cfg.AuthInfos["selected"].TokenFile = tokenPath
		result, err := RESTConfig(configBytes(t, cfg))
		if err == nil || result != nil || !strings.Contains(err.Error(), "tokenFile") {
			t.Fatalf("expected tokenFile policy rejection, got config=%v error=%v", result, err)
		}
		if strings.Contains(err.Error(), tokenPath) || strings.Contains(err.Error(), "process-secret") {
			t.Fatalf("rejection exposed process-local information: %v", err)
		}
	}
}

func TestRESTConfigPreservesCurrentContextAndEmbeddedToken(t *testing.T) {
	cfg := testConfig()
	cfg.CurrentContext = "other"
	result, err := RESTConfig(configBytes(t, cfg))
	if err != nil {
		t.Fatal(err)
	}
	if result.Host != "https://other.example:6443" || result.BearerToken != "other-token" {
		t.Fatalf("current context was not preserved: host=%q token=%q", result.Host, result.BearerToken)
	}
	if result.BearerTokenFile != "" || result.ExecProvider != nil || result.AuthProvider != nil {
		t.Fatal("embedded config retained process-local authentication")
	}
}

func TestRESTConfigAcceptsEmbeddedCertificates(t *testing.T) {
	key, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	if err != nil {
		t.Fatal(err)
	}
	cert := &x509.Certificate{SerialNumber: big.NewInt(1), NotBefore: time.Now().Add(-time.Hour), NotAfter: time.Now().Add(time.Hour), IsCA: true, BasicConstraintsValid: true, KeyUsage: x509.KeyUsageCertSign | x509.KeyUsageDigitalSignature}
	der, err := x509.CreateCertificate(rand.Reader, cert, cert, &key.PublicKey, key)
	if err != nil {
		t.Fatal(err)
	}
	keyDER, err := x509.MarshalECPrivateKey(key)
	if err != nil {
		t.Fatal(err)
	}
	certPEM := pem.EncodeToMemory(&pem.Block{Type: "CERTIFICATE", Bytes: der})
	keyPEM := pem.EncodeToMemory(&pem.Block{Type: "EC PRIVATE KEY", Bytes: keyDER})
	cfg := testConfig()
	cfg.Clusters["selected"].CertificateAuthorityData = certPEM
	cfg.AuthInfos["selected"] = &clientcmdapi.AuthInfo{ClientCertificateData: certPEM, ClientKeyData: keyPEM}
	result, err := RESTConfig(configBytes(t, cfg))
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(result.CAData, certPEM) || !bytes.Equal(result.CertData, certPEM) || !bytes.Equal(result.KeyData, keyPEM) {
		t.Fatal("embedded TLS credentials were not preserved")
	}
	if _, err := rest.TransportFor(result); err != nil {
		t.Fatalf("embedded credentials cannot create transport: %v", err)
	}
}

func TestRESTConfigInvalidConfig(t *testing.T) {
	for _, data := range [][]byte{[]byte("not valid yaml"), configBytes(t, &clientcmdapi.Config{CurrentContext: "missing"})} {
		if cfg, err := RESTConfig(data); err == nil || cfg != nil {
			t.Fatalf("expected invalid config rejection, got config=%v error=%v", cfg, err)
		}
	}
}
