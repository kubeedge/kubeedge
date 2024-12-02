package certs

import (
	"crypto/x509"
	"crypto/x509/pkix"
	"testing"
	"time"
)

func TestSignX509Certs(t *testing.T) {
	cah := new(x509CAHandler)
	certh := new(x509CertsHandler)

	capkw, err := cah.GenPrivateKey()
	if err != nil {
		t.Fatal(err)
	}
	cablock, err := cah.NewSelfSigned(capkw)
	if err != nil {
		t.Fatal(err)
	}

	certpkw, err := certh.GenPrivateKey()
	if err != nil {
		t.Fatal(err)
	}
	csrblock, err := certh.CreateCSR(pkix.Name{
		Country:      []string{"CN"},
		Organization: []string{"system:nodes"},
		Locality:     []string{"Hangzhou"},
		Province:     []string{"Zhejiang"},
		CommonName:   "test-node",
	}, certpkw, nil)

	opts := SignCertsOptionsWithCSR(csrblock.Bytes, cablock.Bytes, capkw.DER(),
		[]x509.ExtKeyUsage{x509.ExtKeyUsageClientAuth}, 24*time.Hour)
	certblock, err := certh.SignCerts(opts)
	if err != nil {
		t.Fatal(err)
	}
	if len(certblock.Bytes) == 0 {
		t.Fatal("cert bytes cannot be empty")
	}
}
