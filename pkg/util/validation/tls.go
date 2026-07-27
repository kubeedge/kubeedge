package validation

import (
	"crypto/tls"
	"crypto/x509"
	"fmt"
	"net"
	"os"
	"time"
)

func ValidateServerTLSFiles(certFile, keyFile, host string, now time.Time) []string {
	var errs []string

	certInfo, err := os.Stat(certFile)
	if err != nil {
		return []string{fmt.Sprintf("tlsCertFile not accessible: %v", err)}
	}
	if !certInfo.Mode().IsRegular() {
		return []string{"tlsCertFile must be a regular file"}
	}

	keyInfo, err := os.Stat(keyFile)
	if err != nil {
		return []string{fmt.Sprintf("tlsPrivateKeyFile not accessible: %v", err)}
	}
	if !keyInfo.Mode().IsRegular() {
		return []string{"tlsPrivateKeyFile must be a regular file"}
	}
	if keyInfo.Mode().Perm()&0o077 != 0 {
		errs = append(errs, "tlsPrivateKeyFile must not be readable or writable by group or others")
	}

	keyPair, err := tls.LoadX509KeyPair(certFile, keyFile)
	if err != nil {
		errs = append(errs, fmt.Sprintf("invalid tls key pair: %v", err))
		return errs
	}
	if len(keyPair.Certificate) == 0 {
		errs = append(errs, "tlsCertFile does not contain a certificate")
		return errs
	}

	leaf, err := x509.ParseCertificate(keyPair.Certificate[0])
	if err != nil {
		errs = append(errs, fmt.Sprintf("failed to parse tls certificate: %v", err))
		return errs
	}
	if now.Before(leaf.NotBefore) || now.After(leaf.NotAfter) {
		errs = append(errs, "tls certificate is not currently valid")
	}

	var serverAuth bool
	for _, usage := range leaf.ExtKeyUsage {
		if usage == x509.ExtKeyUsageServerAuth {
			serverAuth = true
			break
		}
	}
	if !serverAuth {
		errs = append(errs, "tls certificate must support server auth")
	}

	if err := leaf.VerifyHostname(host); err != nil {
		if net.ParseIP(host) != nil {
			errs = append(errs, fmt.Sprintf("tls certificate must include IP SAN %s", host))
		} else {
			errs = append(errs, fmt.Sprintf("tls certificate must match hostname %s", host))
		}
	}

	return errs
}

func IsLoopbackHost(host string) bool {
	if host == "localhost" {
		return true
	}
	ip := net.ParseIP(host)
	return ip != nil && ip.IsLoopback()
}
