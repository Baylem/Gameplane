package auth

import (
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/tls"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/pem"
	"io"
	"math/big"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"
	"time"
)

type rotationCA struct {
	cert *x509.Certificate
	key  *ecdsa.PrivateKey
	pem  []byte
}

func newRotationCA(t *testing.T, serial int64) rotationCA {
	t.Helper()
	key, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	if err != nil {
		t.Fatal(err)
	}
	template := &x509.Certificate{SerialNumber: big.NewInt(serial), Subject: pkix.Name{CommonName: "rotation CA"},
		NotBefore: time.Now().Add(-time.Hour), NotAfter: time.Now().Add(time.Hour), IsCA: true,
		BasicConstraintsValid: true, KeyUsage: x509.KeyUsageCertSign | x509.KeyUsageDigitalSignature}
	der, err := x509.CreateCertificate(rand.Reader, template, template, &key.PublicKey, key)
	if err != nil {
		t.Fatal(err)
	}
	cert, err := x509.ParseCertificate(der)
	if err != nil {
		t.Fatal(err)
	}
	return rotationCA{cert, key, pem.EncodeToMemory(&pem.Block{Type: "CERTIFICATE", Bytes: der})}
}

func (ca rotationCA) issue(t *testing.T, serial int64, usage x509.ExtKeyUsage) ([]byte, []byte, tls.Certificate) {
	t.Helper()
	key, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	if err != nil {
		t.Fatal(err)
	}
	template := &x509.Certificate{SerialNumber: big.NewInt(serial), DNSNames: []string{"localhost"},
		NotBefore: time.Now().Add(-time.Hour), NotAfter: time.Now().Add(time.Hour),
		KeyUsage: x509.KeyUsageDigitalSignature, ExtKeyUsage: []x509.ExtKeyUsage{usage}}
	der, err := x509.CreateCertificate(rand.Reader, template, ca.cert, &key.PublicKey, ca.key)
	if err != nil {
		t.Fatal(err)
	}
	keyDER, err := x509.MarshalECPrivateKey(key)
	if err != nil {
		t.Fatal(err)
	}
	certPEM := pem.EncodeToMemory(&pem.Block{Type: "CERTIFICATE", Bytes: der})
	keyPEM := pem.EncodeToMemory(&pem.Block{Type: "EC PRIVATE KEY", Bytes: keyDER})
	cert, err := tls.X509KeyPair(certPEM, keyPEM)
	if err != nil {
		t.Fatal(err)
	}
	return certPEM, keyPEM, cert
}

func writeRotationBundle(t *testing.T, dir string, cert, key, ca []byte) {
	t.Helper()
	if err := os.MkdirAll(dir, 0o700); err != nil {
		t.Fatal(err)
	}
	for name, data := range map[string][]byte{"tls.crt": cert, "tls.key": key, "ca.crt": ca} {
		if err := os.WriteFile(filepath.Join(dir, name), data, 0o600); err != nil {
			t.Fatal(err)
		}
	}
}

func rotationServer(t *testing.T, cfg *tls.Config) *httptest.Server {
	t.Helper()
	srv := httptest.NewUnstartedServer(okHandler())
	srv.TLS = cfg
	srv.StartTLS()
	t.Cleanup(srv.Close)
	return srv
}

func rotationClient(cert tls.Certificate, roots []byte, version uint16) *http.Client {
	pool := x509.NewCertPool()
	pool.AppendCertsFromPEM(roots)
	return &http.Client{Timeout: 5 * time.Second, Transport: &http.Transport{DisableKeepAlives: true,
		TLSClientConfig: &tls.Config{Certificates: []tls.Certificate{cert}, RootCAs: pool, ServerName: "localhost",
			MinVersion: version, MaxVersion: version, ClientSessionCache: tls.NewLRUClientSessionCache(8)}}}
}

func rotationRequest(t *testing.T, client *http.Client, srv *httptest.Server) (*tls.ConnectionState, error) {
	t.Helper()
	response, err := client.Get(srv.URL)
	if err != nil {
		return nil, err
	}
	defer response.Body.Close()
	if _, err := io.Copy(io.Discard, response.Body); err != nil {
		t.Fatal(err)
	}
	if response.StatusCode != http.StatusOK {
		t.Fatalf("status %d", response.StatusCode)
	}
	return response.TLS, nil
}

func TestServerTLS_HandshakeRenewalAndCARotation(t *testing.T) {
	for _, version := range []uint16{tls.VersionTLS12, tls.VersionTLS13} {
		t.Run(tls.VersionName(version), func(t *testing.T) {
			ca1, ca2 := newRotationCA(t, 100), newRotationCA(t, 200)
			cert1, key1, _ := ca1.issue(t, 1, x509.ExtKeyUsageServerAuth)
			cert2, key2, _ := ca1.issue(t, 2, x509.ExtKeyUsageServerAuth)
			_, _, client1 := ca1.issue(t, 11, x509.ExtKeyUsageClientAuth)
			_, _, client2 := ca2.issue(t, 22, x509.ExtKeyUsageClientAuth)
			dir := t.TempDir()
			writeRotationBundle(t, dir, cert1, key1, ca1.pem)
			cfg, err := ServerTLS(filepath.Join(dir, "tls.crt"), filepath.Join(dir, "tls.key"), filepath.Join(dir, "ca.crt"))
			if err != nil {
				t.Fatal(err)
			}
			srv := rotationServer(t, cfg)
			oldClient := rotationClient(client1, ca1.pem, version)
			for range 2 {
				state, err := rotationRequest(t, oldClient, srv)
				if err != nil {
					t.Fatal(err)
				}
				if state.DidResume || state.PeerCertificates[0].SerialNumber.Int64() != 1 {
					t.Fatalf("unexpected initial handshake: %+v", state)
				}
			}
			writeRotationBundle(t, dir, cert2, key2, ca1.pem)
			state, err := rotationRequest(t, oldClient, srv)
			if err != nil {
				t.Fatal(err)
			}
			if state.DidResume || state.PeerCertificates[0].SerialNumber.Int64() != 2 {
				t.Fatalf("renewed certificate not served: %+v", state)
			}
			writeRotationBundle(t, dir, cert2, key2, ca2.pem)
			if _, err := rotationRequest(t, oldClient, srv); err == nil {
				t.Fatal("removed client CA still trusted, including cached sessions")
			}
			if _, err := rotationRequest(t, rotationClient(client2, ca1.pem, version), srv); err != nil {
				t.Fatalf("new client CA rejected: %v", err)
			}
			if _, err := rotationRequest(t, rotationClient(tls.Certificate{}, ca1.pem, version), srv); err == nil {
				t.Fatal("client certificate requirement weakened")
			}
		})
	}
}

func TestServerTLS_InvalidRotationFailsClosedAndRecovers(t *testing.T) {
	ca := newRotationCA(t, 100)
	cert, key, _ := ca.issue(t, 1, x509.ExtKeyUsageServerAuth)
	_, wrongKey, clientCert := ca.issue(t, 11, x509.ExtKeyUsageClientAuth)
	dir := t.TempDir()
	writeRotationBundle(t, dir, cert, key, ca.pem)
	cfg, err := ServerTLS(filepath.Join(dir, "tls.crt"), filepath.Join(dir, "tls.key"), filepath.Join(dir, "ca.crt"))
	if err != nil {
		t.Fatal(err)
	}
	srv := rotationServer(t, cfg)
	client := rotationClient(clientCert, ca.pem, tls.VersionTLS13)
	if _, err := rotationRequest(t, client, srv); err != nil {
		t.Fatal(err)
	}
	for _, tc := range []struct {
		name string
		bad  []byte
	}{
		{"ca.crt", []byte("not PEM")}, {"tls.crt", []byte("not PEM")}, {"tls.key", wrongKey}, {"ca.crt", nil}, {"tls.crt", nil}, {"tls.key", nil},
	} {
		writeRotationBundle(t, dir, cert, key, ca.pem)
		path := filepath.Join(dir, tc.name)
		if tc.bad == nil {
			if err := os.Remove(path); err != nil {
				t.Fatal(err)
			}
		} else if err := os.WriteFile(path, tc.bad, 0o600); err != nil {
			t.Fatal(err)
		}
		if _, err := rotationRequest(t, client, srv); err == nil {
			t.Fatalf("invalid %s rotation reused old TLS material", tc.name)
		}
	}
	writeRotationBundle(t, dir, cert, key, ca.pem)
	if _, err := rotationRequest(t, client, srv); err != nil {
		t.Fatalf("valid material did not recover: %v", err)
	}
}

func swapRotationProjection(t *testing.T, dir, generation string) {
	t.Helper()
	if err := os.Symlink(generation, filepath.Join(dir, "..data-new")); err != nil {
		t.Fatal(err)
	}
	if err := os.Rename(filepath.Join(dir, "..data-new"), filepath.Join(dir, "..data")); err != nil {
		t.Fatal(err)
	}
}

func TestServerTLS_ProjectedSecretPinsOneGenerationAcrossReads(t *testing.T) {
	ca1, ca2 := newRotationCA(t, 100), newRotationCA(t, 200)
	// Keep the serving keypair identical: only checking keypair consistency
	// would miss an old certificate combined with a new client CA.
	cert, key, _ := ca1.issue(t, 1, x509.ExtKeyUsageServerAuth)
	_, _, oldCert := ca1.issue(t, 11, x509.ExtKeyUsageClientAuth)
	_, _, newCert := ca2.issue(t, 22, x509.ExtKeyUsageClientAuth)
	dir := t.TempDir()
	writeRotationBundle(t, filepath.Join(dir, "generation-a"), cert, key, ca1.pem)
	writeRotationBundle(t, filepath.Join(dir, "generation-b"), cert, key, ca2.pem)
	swapRotationProjection(t, dir, "generation-a")
	for _, name := range []string{"tls.crt", "tls.key", "ca.crt"} {
		if err := os.Symlink(filepath.Join("..data", name), filepath.Join(dir, name)); err != nil {
			t.Fatal(err)
		}
	}
	paths := []string{filepath.Join(dir, "tls.crt"), filepath.Join(dir, "tls.key"), filepath.Join(dir, "ca.crt")}
	reads := 0
	pinned, err := loadServerTLS(paths, func(path string) ([]byte, error) {
		data, err := os.ReadFile(path)
		reads++
		if reads == 1 {
			swapRotationProjection(t, dir, "generation-b")
		}
		return data, err
	})
	if err != nil {
		t.Fatal(err)
	}
	srv := rotationServer(t, pinned)
	if _, err := rotationRequest(t, rotationClient(oldCert, ca1.pem, tls.VersionTLS13), srv); err != nil {
		t.Fatalf("pinned generation lost its CA: %v", err)
	}
	if _, err := rotationRequest(t, rotationClient(newCert, ca1.pem, tls.VersionTLS13), srv); err == nil {
		t.Fatal("CA from a different generation was mixed into pinned material")
	}
	// Production callback must resolve the newer projection on the next handshake.
	cfg, err := ServerTLS(paths[0], paths[1], paths[2])
	if err != nil {
		t.Fatal(err)
	}
	live := rotationServer(t, cfg)
	if _, err := rotationRequest(t, rotationClient(newCert, ca1.pem, tls.VersionTLS13), live); err != nil {
		t.Fatal(err)
	}
	swapRotationProjection(t, dir, "generation-a")
	if _, err := rotationRequest(t, rotationClient(oldCert, ca1.pem, tls.VersionTLS13), live); err != nil {
		t.Fatalf("projection rotation not reloaded: %v", err)
	}
	if _, err := rotationRequest(t, rotationClient(newCert, ca1.pem, tls.VersionTLS13), live); err == nil {
		t.Fatal("removed projected CA retained")
	}
	// Kubelet may remove the old generation after swapping ..data. An
	// interrupted read must fail rather than finish using replacement files.
	reads = 0
	partial, err := loadServerTLS(paths, func(path string) ([]byte, error) {
		data, err := os.ReadFile(path)
		reads++
		if reads == 1 {
			swapRotationProjection(t, dir, "generation-b")
			if err := os.RemoveAll(filepath.Join(dir, "generation-a")); err != nil {
				t.Fatal(err)
			}
		}
		return data, err
	})
	if err == nil || partial != nil {
		t.Fatal("removed generation fell back to mixed material")
	}
	if _, err := rotationRequest(t, rotationClient(newCert, ca1.pem, tls.VersionTLS13), live); err != nil {
		t.Fatalf("next handshake did not recover from removed projection: %v", err)
	}
	// An invalid ..data entry cannot be treated as ordinary files.
	if err := os.Remove(filepath.Join(dir, "..data")); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "..data"), []byte("invalid projection"), 0o600); err != nil {
		t.Fatal(err)
	}
	if _, err := ServerTLS(paths[0], paths[1], paths[2]); err == nil {
		t.Fatal("invalid projection accepted at startup")
	}
}
