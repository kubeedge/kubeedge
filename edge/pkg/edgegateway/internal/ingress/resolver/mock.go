package resolver

import (
	"errors"
	defaults "github.com/kubeedge/kubeedge/edge/pkg/edgegateway/internal/ingress/default"
	apiv1 "k8s.io/api/core/v1"
)

// Mock implements the Resolver interface
type Mock struct {
	ConfigMaps map[string]*apiv1.ConfigMap
}

// GetDefaultBackend returns the backend that must be used as default
func (m Mock) GetDefaultBackend() defaults.Backend {
	return defaults.Backend{}
}

// GetSecret searches for secrets contenating the namespace and name using a the character /
func (m Mock) GetSecret(string) (*apiv1.Secret, error) {
	return nil, nil
}

// GetAuthCertificate resolves a given secret name into an SSL certificate.
// The secret must contain 3 keys named:
//   ca.crt: contains the certificate chain used for authentication
func (m Mock) GetAuthCertificate(string) (*AuthSSLCert, error) {
	return nil, nil
}

// GetService searches for services contenating the namespace and name using a the character /
func (m Mock) GetService(string) (*apiv1.Service, error) {
	return nil, nil
}

// GetConfigMap searches for configMaps contenating the namespace and name using a the character /
func (m Mock) GetConfigMap(name string) (*apiv1.ConfigMap, error) {
	if v, ok := m.ConfigMaps[name]; ok {
		return v, nil
	}
	return nil, errors.New("no configmap")
}
