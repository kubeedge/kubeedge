package common

import (
	"context"
)

// constant for provider and consumer
const (
	Provider = "Provider"
	Consumer = "Consumer"
)

const (
	// ScopeFull means service is able to access to another app's service
	ScopeFull = "full"
	// ScopeApp means service is not able to access to another app's service
	ScopeApp = "app"
)

// constant for micro service environment parameters
const (
	Env = "go-chassis_ENV"

	EnvNodeIP      = "HOSTING_SERVER_IP"
	EnvSchemaRoot  = "SCHEMA_ROOT"
	EnvProjectID   = "CSE_PROJECT_ID"
	EnvCSEEndpoint = "PAAS_CSE_ENDPOINT"
)

// constant environment keys service center, config center, monitor server addresses
const (
	CseRegistryAddress     = "CSE_REGISTRY_ADDR"
	CseConfigCenterAddress = "CSE_CONFIG_CENTER_ADDR"
	CseMonitorServer       = "CSE_MONITOR_SERVER_ADDR"
)

// env connect with "." like service_description.name and service_description.version which can not be used in k8s.
// So we can not use archaius to set env.
// To support this declaring constant for service name and version
// constant for service name and version.
const (
	ServiceName = "SERVICE_NAME"
	Version     = "VERSION"
)

// constant for microservice environment
const (
	EnvValueDev  = "development"
	EnvValueProd = "production"
)

// constant for secure socket layer parameters
const (
	SslCipherPluginKey = "cipherPlugin"
	SslVerifyPeerKey   = "verifyPeer"
	SslCipherSuitsKey  = "cipherSuits"
	SslProtocolKey     = "protocol"
	SslCaFileKey       = "caFile"
	SslCertFileKey     = "certFile"
	SslKeyFileKey      = "keyFile"
	SslCertPwdFileKey  = "certPwdFile"
	AKSKCustomCipher   = "cse.credentials.akskCustomCipher"
)

// constant for protocol types
const (
	ProtocolRest    = "rest"
	ProtocolHighway = "highway"
	LBSessionID     = "go-chassisLB"
)

// configuration placeholders
const (
	PlaceholderInternalIP = "$INTERNAL_IP"
)

// SessionNameSpaceKey metadata session namespace key
const SessionNameSpaceKey = "_Session_Namespace"

// SessionNameSpaceDefaultValue default session namespace value
const SessionNameSpaceDefaultValue = "default"

// DefaultKey default key
const DefaultKey = "default"

// DefaultValue default value
const DefaultValue = "default"

// BuildinTagApp build tag for the application
const BuildinTagApp = "app"

// BuildinTagVersion build tag version
const BuildinTagVersion = "version"

// BuildinLabelVersion build label for version
const BuildinLabelVersion = BuildinTagVersion + ":" + LatestVersion

// CallerKey caller key
const CallerKey = "caller"

const (
	// HeaderSourceName is constant for header source name
	HeaderSourceName = "x-cse-src-microservice"
)

const (
	// RestMethod is the http method for restful protocol
	RestMethod = "method"
)

// constant for default application name and version
const (
	DefaultApp        = "default"
	DefaultVersion    = "0.0.1"
	LatestVersion     = "latest"
	AllVersion        = "0+"
	DefaultStatus     = "UP"
	TESTINGStatus     = "TESTING"
	DefaultLevel      = "BACK"
	DefaultHBInterval = 30
)

//constant used
const (
	HTTP   = "http"
	HTTPS  = "https"
	JSON   = "application/json"
	Create = "CREATE"
	Update = "UPDATE"
	Delete = "DELETE"

	Client           = "client"
	File             = "File"
	DefaultTenant    = "default"
	DefaultChainName = "default"

	FileRegistry      = "File"
	DefaultUserName   = "default"
	DefaultDomainName = "default"
	DefaultProvider   = "default"

	TRUE  = "true"
	FALSE = "false"
)

// const default config for config-center
const (
	DefaultRefreshMode = 1
)

//ContextHeaderKey is the unified key of header value in context
//all protocol integrated with go chassis must set protocol header into context in this context key
type ContextHeaderKey struct{}

// NewContext transforms a metadata to context object
func NewContext(m map[string]string) context.Context {
	if m == nil {
		return context.WithValue(context.Background(), ContextHeaderKey{}, make(map[string]string, 0))
	}
	return context.WithValue(context.Background(), ContextHeaderKey{}, m)
}

// WithContext sets the KV and returns the context object
func WithContext(ctx context.Context, key, val string) context.Context {
	if ctx == nil {
		return context.WithValue(context.Background(), ContextHeaderKey{}, map[string]string{
			key: val,
		})
	}

	at, ok := ctx.Value(ContextHeaderKey{}).(map[string]string)
	if !ok {
		return context.WithValue(ctx, ContextHeaderKey{}, map[string]string{
			key: val,
		})
	}
	at[key] = val
	return ctx
}

// FromContext return the headers which should be send to provider
// through transport
func FromContext(ctx context.Context) map[string]string {
	if ctx == nil {
		return make(map[string]string, 0)
	}
	at, ok := ctx.Value(ContextHeaderKey{}).(map[string]string)
	if !ok {
		return make(map[string]string, 0)
	}
	return at
}
