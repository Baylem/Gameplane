// Package kubeconfig loads self-contained remote-cluster credentials.
package kubeconfig

import (
	"fmt"

	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
	clientcmdapi "k8s.io/client-go/tools/clientcmd/api"
)

// RESTConfig parses a remote kubeconfig without allowing credentials from the
// host filesystem or authentication plugins. All entries are checked, including
// entries outside the current context, before client-go builds a REST config.
func RESTConfig(data []byte) (*rest.Config, error) {
	// Load only decodes the supplied bytes; unlike ClientConfig, it does not
	// read token files or prepare executable authentication.
	cfg, err := clientcmd.Load(data)
	if err != nil {
		return nil, fmt.Errorf("decode kubeconfig: %w", err)
	}
	if err := validateEmbeddedCredentials(cfg); err != nil {
		return nil, err
	}
	result, err := clientcmd.NewNonInteractiveClientConfig(*cfg, "", &clientcmd.ConfigOverrides{}, nil).ClientConfig()
	if err != nil {
		return nil, fmt.Errorf("build kubeconfig client: %w", err)
	}
	return result, nil
}

func validateEmbeddedCredentials(cfg *clientcmdapi.Config) error {
	for _, cluster := range cfg.Clusters {
		if cluster != nil && cluster.CertificateAuthority != "" {
			return unsupportedCredential("certificate-authority")
		}
	}
	for _, auth := range cfg.AuthInfos {
		if auth == nil {
			continue
		}
		switch {
		case auth.TokenFile != "":
			return unsupportedCredential("tokenFile")
		case auth.ClientCertificate != "":
			return unsupportedCredential("client-certificate")
		case auth.ClientKey != "":
			return unsupportedCredential("client-key")
		case auth.Exec != nil:
			return unsupportedCredential("exec")
		case auth.AuthProvider != nil:
			return unsupportedCredential("auth-provider")
		}
	}
	return nil
}

func unsupportedCredential(field string) error {
	// Do not echo paths, commands, config entry names, or credential values.
	return fmt.Errorf("kubeconfig field %s is not supported; use embedded credentials", field)
}
