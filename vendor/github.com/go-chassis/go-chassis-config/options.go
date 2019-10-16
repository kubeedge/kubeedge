package config

import "crypto/tls"

type Options struct {
	ServerURI     string
	Endpoint      string
	TLSConfig     *tls.Config
	TenantName    string
	EnableSSL     bool
	APIVersion    string
	AutoDiscovery bool
	RefreshPort   string

	Labels map[string]string
}
