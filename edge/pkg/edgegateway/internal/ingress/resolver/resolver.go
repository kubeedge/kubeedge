package resolver

import (
	defaults "github.com/kubeedge/kubeedge/edge/pkg/edgegateway/internal/ingress/default"
	apiv1 "k8s.io/api/core/v1"
)

// Resolver is an interface that knows how to extract information from a controller
type Resolver interface {
	// GetDefaultBackend returns the backend that must be used as default
	GetDefaultBackend() defaults.Backend

	// GetConfigMap searches for configmap containing the namespace and name usting the character /
	GetConfigMap(string) (*apiv1.ConfigMap, error)

	// GetSecret searches for secrets containing the namespace and name using a the character /
	GetSecret(string) (*apiv1.Secret, error)

	// GetAuthCertificate resolves a given secret name into an SSL certificate and CRL.
	// The secret must contain 2 keys named:

	//   ca.crt: contains the certificate chain used for authentication
	//   ca.crl: contains the revocation list used for authentication
	GetAuthCertificate(string) (*AuthSSLCert, error)

	// GetService searches for services containing the namespace and name using a the character /
	GetService(string) (*apiv1.Service, error)
}

// AuthSSLCert contains the necessary information to do certificate based
// authentication of an ingress location
type AuthSSLCert struct {
	// Secret contains the name of the secret this was fetched from
	Secret string `json:"secret"`
	// CAFileName contains the path to the secrets 'ca.crt'
	CAFileName string `json:"caFilename"`
	// CASHA contains the SHA1 hash of the 'ca.crt' or combinations of (tls.crt, tls.key, tls.crt) depending on certs in secret
	CASHA string `json:"caSha"`
	// CRLFileName contains the path to the secrets 'ca.crl'
	CRLFileName string `json:"crlFileName"`
	// CRLSHA contains the SHA1 hash of the 'ca.crl' file
	CRLSHA string `json:"crlSha"`
	// PemFileName contains the path to the secrets 'tls.crt' and 'tls.key'
	PemFileName string `json:"pemFilename"`
}

// Equal tests for equality between two AuthSSLCert types
func (asslc1 *AuthSSLCert) Equal(assl2 *AuthSSLCert) bool {
	if asslc1 == assl2 {
		return true
	}
	if asslc1 == nil || assl2 == nil {
		return false
	}

	if asslc1.Secret != assl2.Secret {
		return false
	}
	if asslc1.CAFileName != assl2.CAFileName {
		return false
	}
	if asslc1.CASHA != assl2.CASHA {
		return false
	}

	if asslc1.CRLFileName != assl2.CRLFileName {
		return false
	}
	if asslc1.CRLSHA != assl2.CRLSHA {
		return false
	}

	return true
}
