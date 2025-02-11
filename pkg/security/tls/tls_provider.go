package tls

import (
	"crypto/tls"
	"crypto/x509"
	"fmt"
	"os"
)

// StandardTLSProvider implements standard TLS
type StandardTLSProvider struct{}

func (p *StandardTLSProvider) GetServerConfig(config *TLSConfig) (*tls.Config, error) {
	tlsConfig := &tls.Config{
		MinVersion:   config.MinVersion,
		MaxVersion:   config.MaxVersion,
		ClientAuth:   config.ClientAuth,
		CipherSuites: config.CipherSuites,
	}

	if config.CertFile != "" && config.KeyFile != "" {
		cert, err := tls.LoadX509KeyPair(config.CertFile, config.KeyFile)
		if err != nil {
			return nil, fmt.Errorf("failed to load key pair: %v", err)
		}
		tlsConfig.Certificates = []tls.Certificate{cert}
	}

	if config.CAFile != "" {
		caCert, err := os.ReadFile(config.CAFile)
		if err != nil {
			return nil, fmt.Errorf("failed to read CA file: %v", err)
		}

		caCertPool := x509.NewCertPool()
		if !caCertPool.AppendCertsFromPEM(caCert) {
			return nil, fmt.Errorf("failed to parse CA certificate")
		}
		tlsConfig.ClientCAs = caCertPool
	}

	return tlsConfig, nil
}

func (p *StandardTLSProvider) GetClientConfig(config *TLSConfig) (*tls.Config, error) {
	tlsConfig := &tls.Config{
		ServerName:   config.ServerName,
		MinVersion:   config.MinVersion,
		MaxVersion:   config.MaxVersion,
		CipherSuites: config.CipherSuites,
	}

	if config.CertFile != "" && config.KeyFile != "" {
		cert, err := tls.LoadX509KeyPair(config.CertFile, config.KeyFile)
		if err != nil {
			return nil, fmt.Errorf("failed to load key pair: %v", err)
		}
		tlsConfig.Certificates = []tls.Certificate{cert}
	}

	if config.CAFile != "" {
		caCert, err := os.ReadFile(config.CAFile)
		if err != nil {
			return nil, fmt.Errorf("failed to read CA file: %v", err)
		}

		caCertPool := x509.NewCertPool()
		if !caCertPool.AppendCertsFromPEM(caCert) {
			return nil, fmt.Errorf("failed to parse CA certificate")
		}
		tlsConfig.RootCAs = caCertPool
	}

	return tlsConfig, nil
}

func (p *StandardTLSProvider) LoadX509KeyPair(certFile, keyFile string) (interface{}, error) {
	return tls.LoadX509KeyPair(certFile, keyFile)
}
