package config

import "crypto/tls"

type Options struct {
	ServerURI         string
	Endpoint          string
	ServiceName       string
	ApolloServiceName string
	Cluster           string
	Namespace         string
	App               string
	Env               string
	Version           string
	TLSConfig         *tls.Config
	TenantName        string
	EnableSSL         bool
	APIVersion        string
	AutoDiscovery     bool
	RefreshPort       string
}
