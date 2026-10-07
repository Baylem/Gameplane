// Package auth implements the agent's request authenticator. Two
// modes are supported:
//
//   - mTLS (preferred): the agent listens TLS and requires the client
//     (API) to present a cert signed by --tls-client-ca. Request
//     handlers receive a verified chain. Nothing else is needed.
//
//   - Shared token (fallback for early dev): the agent accepts any
//     request bearing Authorization: Bearer <token> where <token>
//     matches the contents of --api-token-file.
//
// mTLS is the default the operator wires up at production time. The
// shared-token path exists so the agent can run on a plain port 8090
// during local kind development.
package auth

import (
	"crypto/subtle"
	"crypto/tls"
	"crypto/x509"
	"encoding/pem"
	"errors"
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"strings"
)

// Config holds the configuration for setting up the authenticator.
type Config struct {
	ClientCAFile string // mTLS CA bundle
	TokenFile    string // shared-secret fallback
}

// Authenticator implements request authentication via mTLS or shared token.
type Authenticator struct {
	mode  string
	token []byte
}

// New creates a new Authenticator from the supplied configuration.
func New(cfg Config) (*Authenticator, error) {
	switch {
	case cfg.ClientCAFile != "":
		// mTLS is enforced by the TLS layer; no token needed.
		return &Authenticator{mode: "mtls"}, nil
	case cfg.TokenFile != "":
		b, err := os.ReadFile(cfg.TokenFile)
		if err != nil {
			return nil, fmt.Errorf("read token file: %w", err)
		}
		tok := strings.TrimSpace(string(b))
		if tok == "" {
			return nil, errors.New("token file is empty")
		}
		return &Authenticator{mode: "token", token: []byte(tok)}, nil
	default:
		return nil, errors.New("no auth configured: provide --tls-client-ca or --api-token-file")
	}
}

// Middleware returns an HTTP middleware that enforces authentication.
func (a *Authenticator) Middleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
		switch a.mode {
		case "mtls":
			if req.TLS == nil || len(req.TLS.VerifiedChains) == 0 {
				http.Error(w, "client certificate required", http.StatusUnauthorized)
				return
			}
		case "token":
			got := bearer(req.Header.Get("Authorization"))
			if got == "" || subtle.ConstantTimeCompare([]byte(got), a.token) != 1 {
				http.Error(w, "unauthorized", http.StatusUnauthorized)
				return
			}
		}
		next.ServeHTTP(w, req)
	})
}

func bearer(h string) string {
	const prefix = "Bearer "
	if !strings.HasPrefix(h, prefix) {
		return ""
	}
	return strings.TrimSpace(h[len(prefix):])
}

// ServerTLS builds a tls.Config enforcing client-cert verification
// against the supplied CA bundle. Material is validated at startup and loaded
// again for each new handshake so projected Secret renewal needs no rollout.
// It advertises HTTP/2 and HTTP/1.1, matching the agent's HTTP server. Callers
// using other protocol policies must set NextProtos to their supported list.
func ServerTLS(certFile, keyFile, clientCAFile string) (*tls.Config, error) {
	paths := []string{certFile, keyFile, clientCAFile}
	cfg, err := loadServerTLS(paths, os.ReadFile)
	if err != nil {
		return nil, err
	}
	cfg.GetConfigForClient = func(*tls.ClientHelloInfo) (*tls.Config, error) {
		current, err := loadServerTLS(paths, os.ReadFile)
		if err != nil {
			// Do not fall back to cached certificates or trust after bad rotation.
			return nil, err
		}
		// Retain caller-provided TLS settings without mutating a configuration
		// already in use by another handshake.
		next := cfg.Clone()
		next.GetConfigForClient = nil
		next.Certificates = current.Certificates
		next.ClientCAs = current.ClientCAs
		next.ClientAuth = tls.RequireAndVerifyClientCert
		next.MinVersion = max(next.MinVersion, tls.VersionTLS12)
		next.SessionTicketsDisabled = true
		return next, nil
	}
	return cfg, nil
}

// loadServerTLS reads one generation of every projected volume before parsing
// any material. Resolving each visible file separately can mix a serving pair
// from one generation with a client CA from another, even when the key matches.
func loadServerTLS(paths []string, readFile func(string) ([]byte, error)) (*tls.Config, error) {
	pinned := make([]string, len(paths))
	generations := make(map[string]string)
	for i, path := range paths {
		if path == "" {
			continue
		}
		path = filepath.Clean(path)
		dir := filepath.Dir(path)
		generation, ok := generations[dir]
		if !ok {
			target, err := os.Readlink(filepath.Join(dir, "..data"))
			switch {
			case err == nil:
				generation = filepath.Join(dir, target)
			case errors.Is(err, os.ErrNotExist):
				// Regular files remain supported for non-Kubernetes deployments.
				generation = dir
			default:
				return nil, fmt.Errorf("resolve TLS Secret generation: %w", err)
			}
			generations[dir] = generation
		}
		pinned[i] = filepath.Join(generation, filepath.Base(path))
	}
	certPEM, err := readFile(pinned[0])
	if err != nil {
		return nil, fmt.Errorf("server keypair: %w", err)
	}
	keyPEM, err := readFile(pinned[1])
	if err != nil {
		return nil, fmt.Errorf("server keypair: %w", err)
	}
	cert, err := tls.X509KeyPair(certPEM, keyPEM)
	if err != nil {
		return nil, fmt.Errorf("server keypair: %w", err)
	}
	pool := x509.NewCertPool()
	if pinned[2] != "" {
		ca, err := readFile(pinned[2])
		if err != nil {
			return nil, fmt.Errorf("read client CA: %w", err)
		}
		// AppendCertsFromPEM silently skips invalid certificates. Validate every
		// certificate block first so a partially corrupt rotation fails closed.
		for rest := ca; len(rest) > 0; {
			var block *pem.Block
			block, rest = pem.Decode(rest)
			if block == nil {
				break
			}
			if block.Type != "CERTIFICATE" {
				continue
			}
			if _, err := x509.ParseCertificate(block.Bytes); err != nil {
				return nil, fmt.Errorf("parse client CA certificate: %w", err)
			}
		}
		if !pool.AppendCertsFromPEM(ca) {
			return nil, errors.New("client CA bundle contains no valid certs")
		}
	}
	return &tls.Config{
		Certificates: []tls.Certificate{cert},
		ClientCAs:    pool,
		ClientAuth:   tls.RequireAndVerifyClientCert,
		MinVersion:   tls.VersionTLS12,
		// net/http adjusts ALPN on its own clone. Set our HTTP protocols
		// explicitly so GetConfigForClient's replacement retains negotiation.
		NextProtos: []string{"h2", "http/1.1"},
		// Every new connection must verify against the current client CA.
		// TLS 1.2/1.3 session resumption would otherwise retain stale trust.
		SessionTicketsDisabled: true,
	}, nil
}
