package tls

import "crypto/tls"

// TLSConfig represents configuration for TLS
type TLSConfig struct {
	CertFile 	string
	KeyFile  	string
	CAFile		string
	ServerName	string
	MinVersion  uint16
	MaxVersion  uint16
	ClientAuth  tls.ClientAuthType
	CipherSuites []uint16
}

// TLSProvider interface defines methods that must be implemented by TLS providers
type TLSProvider interface {
    // GetServerConfig returns a server TLS configuration
    GetServerConfig(config *TLSConfig) (*tls.Config, error)
    
    // GetClientConfig returns a client TLS configuration 
    GetClientConfig(config *TLSConfig) (*tls.Config, error)
    
    // LoadX509KeyPair loads an X.509 key pair
    LoadX509KeyPair(certFile, keyFile string) (interface{}, error)
}