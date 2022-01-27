package certificate

import (
	"time"

	"github.com/edgefarm/vault-integration/pkg/certretrieval"

	"github.com/kubeedge/kubeedge/pkg/apis/componentconfig/edgecore/v1alpha1"
)

func NewVaultRetriever(config v1alpha1.EdgeHub) (*VaultRetriever, error) {
	ttlDuration, err := time.ParseDuration(config.Vault.TTL)
	if err != nil {
		return nil, err
	}

	retriever, err := certretrieval.New(certretrieval.Config{
		Tokenfile:   config.Vault.TokenFile,
		Vault:       config.Vault.Vault,
		ServerCA:    config.TLSCAFile,
		Role:        config.Vault.Role,
		Name:        config.Vault.CommonName,
		TTL:         ttlDuration,
		Force:       true,
		OutCAfile:   config.TLSCAFile,
		OutCertfile: config.TLSCertFile,
		OutKeyfile:  config.TLSPrivateKeyFile,
		// 20% buffer for validity checks
		ValidityCheckTolerance: 80,
	})
	if err != nil {
		return nil, err
	}
	return &VaultRetriever{*retriever}, nil
}

// VaultRetriever is an implementation of the certificate.CertificateRetriever interface
// that integrates with Hashicorp Vault. The actual implementation just delegates
// to the certRetrieval library, apart from configuration handling. The library
// will handle the communication with Vault
type VaultRetriever struct {
	certRetrieval certretrieval.CertRetrieval
}

func (vr *VaultRetriever) RetrieveCertificate() error {
	return vr.certRetrieval.Retrieve()
}
