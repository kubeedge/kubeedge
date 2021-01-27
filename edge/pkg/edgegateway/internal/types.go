package ingress

import (
	"github.com/kubeedge/kubeedge/edge/pkg/edgegateway/internal/ingress/annotations"
	"github.com/kubeedge/kubeedge/edge/pkg/edgegateway/internal/ingress/annotations/auth"
	"github.com/kubeedge/kubeedge/edge/pkg/edgegateway/internal/ingress/annotations/authreq"
	"github.com/kubeedge/kubeedge/edge/pkg/edgegateway/internal/ingress/annotations/authtls"
	"github.com/kubeedge/kubeedge/edge/pkg/edgegateway/internal/ingress/annotations/connection"
	"github.com/kubeedge/kubeedge/edge/pkg/edgegateway/internal/ingress/annotations/cors"
	"github.com/kubeedge/kubeedge/edge/pkg/edgegateway/internal/ingress/annotations/fastcgi"
	"github.com/kubeedge/kubeedge/edge/pkg/edgegateway/internal/ingress/annotations/influxdb"
	"github.com/kubeedge/kubeedge/edge/pkg/edgegateway/internal/ingress/annotations/ipwhitelist"
	"github.com/kubeedge/kubeedge/edge/pkg/edgegateway/internal/ingress/annotations/log"
	"github.com/kubeedge/kubeedge/edge/pkg/edgegateway/internal/ingress/annotations/mirror"
	"github.com/kubeedge/kubeedge/edge/pkg/edgegateway/internal/ingress/annotations/modsecurity"
	"github.com/kubeedge/kubeedge/edge/pkg/edgegateway/internal/ingress/annotations/opentracing"
	"github.com/kubeedge/kubeedge/edge/pkg/edgegateway/internal/ingress/annotations/proxy"
	"github.com/kubeedge/kubeedge/edge/pkg/edgegateway/internal/ingress/annotations/proxyssl"
	"github.com/kubeedge/kubeedge/edge/pkg/edgegateway/internal/ingress/annotations/ratelimit"
	"github.com/kubeedge/kubeedge/edge/pkg/edgegateway/internal/ingress/annotations/redirect"
	"github.com/kubeedge/kubeedge/edge/pkg/edgegateway/internal/ingress/annotations/rewrite"
	apiv1 "k8s.io/api/core/v1"
	networking "k8s.io/api/networking/v1beta1"
	"k8s.io/apimachinery/pkg/util/intstr"
	"reflect"
)

// Configuration holds the definition of all the parts required to describe all
// ingresses reachable by the ingress controller (using a filter by namespace)
type Configuration struct {
	// Backends are a list of backends used by all the Ingress rules in the
	// ingress controller. This list includes the default backend
	Backends []*Backend `json:"backends,omitempty"`
	// Servers save the website config
	Servers []*Server `json:"servers,omitempty"`
	// TCPEndpoints contain endpoints for tcp streams handled by this backend
	// +optional
	TCPEndpoints []L4Service `json:"tcpEndpoints,omitempty"`
	// UDPEndpoints contain endpoints for udp streams handled by this backend
	// +optional
	UDPEndpoints []L4Service `json:"udpEndpoints,omitempty"`
	// PassthroughBackends contains the backends used for SSL passthrough.
	// It contains information about the associated Server Name Indication (SNI).
	// +optional
	PassthroughBackends []*SSLPassthroughBackend `json:"passthroughBackends,omitempty"`

	// BackendConfigChecksum contains the particular checksum of a Configuration object
	BackendConfigChecksum string `json:"BackendConfigChecksum,omitempty"`

	// ConfigurationChecksum contains the particular checksum of a Configuration object
	ConfigurationChecksum string `json:"configurationChecksum,omitempty"`
}

// Backend describes one or more remote server/s (endpoints) associated with a service
// +k8s:deepcopy-gen=true
type Backend struct {
	// Name represents an unique apiv1.Service name formatted as <namespace>-<name>-<port>
	Name    string             `json:"name"`
	Service *apiv1.Service     `json:"service,omitempty"`
	Port    intstr.IntOrString `json:"port"`
	// SSLPassthrough indicates that Ingress controller will delegate TLS termination to the endpoints.
	SSLPassthrough bool `json:"sslPassthrough"`
	// Endpoints contains the list of endpoints currently running
	Endpoints []Endpoint `json:"endpoints,omitempty"`
	// StickySessionAffinitySession contains the StickyConfig object with stickyness configuration
	SessionAffinity SessionAffinityConfig `json:"sessionAffinityConfig"`
	// Consistent hashing by NGINX variable
	UpstreamHashBy UpstreamHashByConfig `json:"upstreamHashByConfig,omitempty"`
	// LB algorithm configuration per ingress
	LoadBalancing string `json:"load-balance,omitempty"`
	// Denotes if a backend has no server. The backend instead shares a server with another backend and acts as an
	// alternative backend.
	// This can be used to share multiple upstreams in the sam nginx server block.
	NoServer bool `json:"noServer"`
	// Policies to describe the characteristics of an alternative backend.
	// +optional
	TrafficShapingPolicy TrafficShapingPolicy `json:"trafficShapingPolicy,omitempty"`
	// Contains a list of backends without servers that are associated with this backend.
	// +optional
	AlternativeBackends []string `json:"alternativeBackends,omitempty"`
}

// TrafficShapingPolicy describes the policies to put in place when a backend has no server and is used as an
// alternative backend
// +k8s:deepcopy-gen=true
type TrafficShapingPolicy struct {
	// Weight (0-100) of traffic to redirect to the backend.
	// e.g. Weight 20 means 20% of traffic will be redirected to the backend and 80% will remain
	// with the other backend. 0 weight will not send any traffic to this backend
	Weight int `json:"weight"`
	// Header on which to redirect requests to this backend
	Header string `json:"header"`
	// HeaderValue on which to redirect requests to this backend
	HeaderValue string `json:"headerValue"`
	// HeaderPattern the header value match pattern, support exact, regex.
	HeaderPattern string `json:"headerPattern"`
	// Cookie on which to redirect requests to this backend
	Cookie string `json:"cookie"`
}

// HashInclude defines if a field should be used or not to calculate the hash
func (s Backend) HashInclude(field string, v interface{}) (bool, error) {
	return (field != "Endpoints"), nil
}

// SessionAffinityConfig describes different affinity configurations for new sessions.
// Once a session is mapped to a backend based on some affinity setting, it
// retains that mapping till the backend goes down, or the ingress controller
// restarts. Exactly one of these values will be set on the upstream, since multiple
// affinity values are incompatible. Once set, the backend makes no guarantees
// about honoring updates.
// +k8s:deepcopy-gen=true
type SessionAffinityConfig struct {
	AffinityType          string                `json:"name"`
	AffinityMode          string                `json:"mode"`
	CookieSessionAffinity CookieSessionAffinity `json:"cookieSessionAffinity"`
}

// CookieSessionAffinity defines the structure used in Affinity configured by Cookies.
// +k8s:deepcopy-gen=true
type CookieSessionAffinity struct {
	Name                    string              `json:"name"`
	Expires                 string              `json:"expires,omitempty"`
	MaxAge                  string              `json:"maxage,omitempty"`
	Locations               map[string][]string `json:"locations,omitempty"`
	Path                    string              `json:"path,omitempty"`
	SameSite                string              `json:"samesite,omitempty"`
	ConditionalSameSiteNone bool                `json:"conditional_samesite_none,omitempty"`
	ChangeOnFailure         bool                `json:"change_on_failure,omitempty"`
}

// UpstreamHashByConfig described setting from the upstream-hash-by* annotations.
type UpstreamHashByConfig struct {
	UpstreamHashBy           string `json:"upstream-hash-by,omitempty"`
	UpstreamHashBySubset     bool   `json:"upstream-hash-by-subset,omitempty"`
	UpstreamHashBySubsetSize int    `json:"upstream-hash-by-subset-size,omitempty"`
}

// Endpoint describes a kubernetes endpoint in a backend
// +k8s:deepcopy-gen=true
type Endpoint struct {
	// Address IP address of the endpoint
	Address string `json:"address"`
	// Port number of the TCP port
	Port string `json:"port"`
	// Target returns a reference to the object providing the endpoint
	Target *apiv1.ObjectReference `json:"target,omitempty"`
}

// Server describes a website
type Server struct {
	// Hostname returns the FQDN of the server
	Hostname string `json:"hostname"`
	// SSLPassthrough indicates if the TLS termination is realized in
	// the server or in the remote endpoint
	SSLPassthrough bool `json:"sslPassthrough"`
	// SSLCert describes the certificate that will be used on the server
	SSLCert *SSLCert `json:"sslCert"`
	// Locations list of URIs configured in the server.
	Locations []*Location `json:"locations,omitempty"`
	// Aliases return the alias of the server name
	Aliases []string `json:"aliases,omitempty"`
	// RedirectFromToWWW returns if a redirect to/from prefix www is required
	RedirectFromToWWW bool `json:"redirectFromToWWW,omitempty"`
	// CertificateAuth indicates the this server requires mutual authentication
	// +optional
	CertificateAuth authtls.Config `json:"certificateAuth"`
	// ProxySSL indicates the this server uses client certificate to access backends
	// +optional
	ProxySSL proxyssl.Config `json:"proxySSL"`
	// ServerSnippet returns the snippet of server
	// +optional
	ServerSnippet string `json:"serverSnippet"`
	// SSLCiphers returns list of ciphers to be enabled
	SSLCiphers string `json:"sslCiphers,omitempty"`
	// SSLPreferServerCiphers indicates that server ciphers should be preferred
	// over client ciphers when using the SSLv3 and TLS protocols.
	SSLPreferServerCiphers string `json:"sslPreferServerCiphers,omitempty"`
	// AuthTLSError contains the reason why the access to a server should be denied
	AuthTLSError string `json:"authTLSError,omitempty"`
}

// Location describes an URI inside a server.
// Also contains additional information about annotations in the Ingress.
//
// In some cases when more than one annotations is defined a particular order in the execution
// is required.
// The chain in the execution order of annotations should be:
// - Whitelist
// - RateLimit
// - BasicDigestAuth
// - ExternalAuth
// - Redirect
type Location struct {
	// Path is an extended POSIX regex as defined by IEEE Std 1003.1,
	// (i.e this follows the egrep/unix syntax, not the perl syntax)
	// matched against the path of an incoming request. Currently it can
	// contain characters disallowed from the conventional "path"
	// part of a URL as defined by RFC 3986. Paths must begin with
	// a '/'. If unspecified, the path defaults to a catch all sending
	// traffic to the backend.
	Path string `json:"path"`
	// PathType represents the type of path referred to by a HTTPIngressPath.
	PathType *networking.PathType `json:"pathType"`
	// IsDefBackend indicates if service specified in the Ingress
	// contains active endpoints or not. Returning true means the location
	// uses the default backend.
	IsDefBackend bool `json:"isDefBackend"`
	// Ingress returns the ingress from which this location was generated
	Ingress *Ingress `json:"ingress"`
	// Backend describes the name of the backend to use.
	Backend string `json:"backend"`
	// Service describes the referenced services from the ingress
	Service *apiv1.Service `json:"-"`
	// Port describes to which port from the service
	Port intstr.IntOrString `json:"port"`
	// Overwrite the Host header passed into the backend. Defaults to
	// vhost of the incoming request.
	// +optional
	UpstreamVhost string `json:"upstream-vhost"`
	// BasicDigestAuth returns authentication configuration for
	// an Ingress rule.
	// +optional
	BasicDigestAuth auth.Config `json:"basicDigestAuth,omitempty"`
	// Denied returns an error when this location cannot not be allowed
	// Requesting a denied location should return HTTP code 403.
	Denied *string `json:"denied,omitempty"`
	// CorsConfig returns the Cors Configuration for the ingress rule
	// +optional
	CorsConfig cors.Config `json:"corsConfig,omitempty"`
	// ExternalAuth indicates the access to this location requires
	// authentication using an external provider
	// +optional
	ExternalAuth authreq.Config `json:"externalAuth,omitempty"`
	// EnableGlobalAuth indicates if the access to this location requires
	// authentication using an external provider defined in controller's config
	EnableGlobalAuth bool `json:"enableGlobalAuth"`
	// HTTP2PushPreload allows to configure the HTTP2 Push Preload from backend
	// original location.
	// +optional
	HTTP2PushPreload bool `json:"http2PushPreload,omitempty"`
	// RateLimit describes a limit in the number of connections per IP
	// address or connections per second.
	// The Redirect annotation precedes RateLimit
	// +optional
	RateLimit ratelimit.Config `json:"rateLimit,omitempty"`
	// Redirect describes a temporal o permanent redirection this location.
	// +optional
	Redirect redirect.Config `json:"redirect,omitempty"`
	// Rewrite describes the redirection this location.
	// +optional
	Rewrite rewrite.Config `json:"rewrite,omitempty"`
	// Whitelist indicates only connections from certain client
	// addresses or networks are allowed.
	// +optional
	Whitelist ipwhitelist.SourceRange `json:"whitelist,omitempty"`
	// Proxy contains information about timeouts and buffer sizes
	// to be used in connections against endpoints
	// +optional
	Proxy proxy.Config `json:"proxy,omitempty"`
	// ProxySSL contains information about SSL configuration parameters
	// to be used in connections against endpoints
	// +optional
	ProxySSL proxyssl.Config `json:"proxySSL,omitempty"`
	// UsePortInRedirects indicates if redirects must specify the port
	// +optional
	UsePortInRedirects bool `json:"usePortInRedirects"`
	// ConfigurationSnippet contains additional configuration for the backend
	// to be considered in the configuration of the location
	ConfigurationSnippet string `json:"configurationSnippet"`
	// Connection contains connection header to override the default Connection header
	// to the request.
	// +optional
	Connection connection.Config `json:"connection"`
	// ClientBodyBufferSize allows for the configuration of the client body
	// buffer size for a specific location.
	// +optional
	ClientBodyBufferSize string `json:"clientBodyBufferSize,omitempty"`
	// DefaultBackend allows the use of a custom default backend for this location.
	// +optional
	DefaultBackend *apiv1.Service `json:"-"`
	// DefaultBackendUpstreamName is the upstream-formatted string for the name of
	// this location's custom default backend
	DefaultBackendUpstreamName string `json:"defaultBackendUpstreamName,omitempty"`
	// XForwardedPrefix allows to add a header X-Forwarded-Prefix to the request with the
	// original location.
	// +optional
	XForwardedPrefix string `json:"xForwardedPrefix,omitempty"`
	// Logs allows to enable or disable the nginx logs
	// By default access logs are enabled and rewrite logs are disabled
	Logs log.Config `json:"logs,omitempty"`
	// InfluxDB allows to monitor the incoming request by sending them to an influxdb database
	// +optional
	InfluxDB influxdb.Config `json:"influxDB,omitempty"`
	// BackendProtocol indicates which protocol should be used to communicate with the service
	// By default this is HTTP
	BackendProtocol string `json:"backend-protocol"`
	// FastCGI allows the ingress to act as a FastCGI client for a given location.
	// +optional
	FastCGI fastcgi.Config `json:"fastcgi,omitempty"`
	// CustomHTTPErrors specifies the error codes that should be intercepted.
	// +optional
	CustomHTTPErrors []int `json:"custom-http-errors"`
	// ModSecurity allows to enable and configure modsecurity
	// +optional
	ModSecurity modsecurity.Config `json:"modsecurity"`
	// Satisfy dictates allow access if any or all is set
	Satisfy string `json:"satisfy"`
	// Mirror allows you to mirror traffic to a "test" backend
	// +optional
	Mirror mirror.Config `json:"mirror,omitempty"`
	// Opentracing allows the global opentracing setting to be overridden for a location
	// +optional
	Opentracing opentracing.Config `json:"opentracing"`
}

// SSLPassthroughBackend describes a SSL upstream server configured
// as passthrough (no TLS termination in the ingress controller)
// The endpoints must provide the TLS termination exposing the required SSL certificate.
// The ingress controller only pipes the underlying TCP connection
type SSLPassthroughBackend struct {
	Service *apiv1.Service     `json:"-"`
	Port    intstr.IntOrString `json:"port"`
	// Backend describes the endpoints to use.
	Backend string `json:"namespace,omitempty"`
	// Hostname returns the FQDN of the server
	Hostname string `json:"hostname"`
}

// L4Service describes a L4 Ingress service.
type L4Service struct {
	// Port external port to expose
	Port int `json:"port"`
	// Backend of the service
	Backend L4Backend `json:"backend"`
	// Endpoints active endpoints of the service
	Endpoints []Endpoint `json:"endpoints,omitempty"`
	// k8s Service
	Service *apiv1.Service `json:"-"`
}

// L4Backend describes the kubernetes service behind L4 Ingress service
type L4Backend struct {
	Port      intstr.IntOrString `json:"port"`
	Name      string             `json:"name"`
	Namespace string             `json:"namespace"`
	Protocol  apiv1.Protocol     `json:"protocol"`
	// +optional
	ProxyProtocol ProxyProtocol `json:"proxyProtocol"`
}

// ProxyProtocol describes the proxy protocol configuration
type ProxyProtocol struct {
	Decode bool `json:"decode"`
	Encode bool `json:"encode"`
}

// Ingress holds the definition of an Ingress plus its annotations
type Ingress struct {
	networking.Ingress `json:"-"`
	ParsedAnnotations  *annotations.Ingress `json:"parsedAnnotations"`
}

// Equal tests for equality between two Configuration types
func (c1 *Configuration) Equal(c2 *Configuration) bool {
	if c1 == c2 {
		return true
	}
	if c1 == nil || c2 == nil {
		return false
	}

	match := compareBackends(c1.Backends, c2.Backends)
	if !match {
		return false
	}

	if len(c1.Servers) != len(c2.Servers) {
		return false
	}

	// Servers are sorted
	for idx, c1s := range c1.Servers {
		if !c1s.Equal(c2.Servers[idx]) {
			return false
		}
	}

	match = compareL4Service(c1.TCPEndpoints, c2.TCPEndpoints)
	if !match {
		return false
	}

	match = compareL4Service(c1.UDPEndpoints, c2.UDPEndpoints)
	if !match {
		return false
	}

	if len(c1.PassthroughBackends) != len(c2.PassthroughBackends) {
		return false
	}

	for _, ptb1 := range c1.PassthroughBackends {
		found := false
		for _, ptb2 := range c2.PassthroughBackends {
			if ptb1.Equal(ptb2) {
				found = true
				break
			}
		}
		if !found {
			return false
		}
	}

	if c1.BackendConfigChecksum != c2.BackendConfigChecksum {
		return false
	}

	return true
}

// Equal tests for equality between two Backend types
func (b1 *Backend) Equal(b2 *Backend) bool {
	if b1 == b2 {
		return true
	}
	if b1 == nil || b2 == nil {
		return false
	}
	if b1.Name != b2.Name {
		return false
	}
	if b1.NoServer != b2.NoServer {
		return false
	}

	if b1.Service != b2.Service {
		if b1.Service == nil || b2.Service == nil {
			return false
		}
		if b1.Service.GetNamespace() != b2.Service.GetNamespace() {
			return false
		}
		if b1.Service.GetName() != b2.Service.GetName() {
			return false
		}
	}

	if b1.Port != b2.Port {
		return false
	}
	if b1.SSLPassthrough != b2.SSLPassthrough {
		return false
	}
	if !(&b1.SessionAffinity).Equal(&b2.SessionAffinity) {
		return false
	}
	if b1.UpstreamHashBy != b2.UpstreamHashBy {
		return false
	}
	if b1.LoadBalancing != b2.LoadBalancing {
		return false
	}

	match := compareEndpoints(b1.Endpoints, b2.Endpoints)
	if !match {
		return false
	}

	if !b1.TrafficShapingPolicy.Equal(b2.TrafficShapingPolicy) {
		return false
	}

	return StringElementsMatch(b1.AlternativeBackends, b2.AlternativeBackends)
}

// Equal tests for equality between two SessionAffinityConfig types
func (sac1 *SessionAffinityConfig) Equal(sac2 *SessionAffinityConfig) bool {
	if sac1 == sac2 {
		return true
	}
	if sac1 == nil || sac2 == nil {
		return false
	}
	if sac1.AffinityType != sac2.AffinityType {
		return false
	}
	if sac1.AffinityMode != sac2.AffinityMode {
		return false
	}
	if !(&sac1.CookieSessionAffinity).Equal(&sac2.CookieSessionAffinity) {
		return false
	}

	return true
}

// Equal tests for equality between two CookieSessionAffinity types
func (csa1 *CookieSessionAffinity) Equal(csa2 *CookieSessionAffinity) bool {
	if csa1 == csa2 {
		return true
	}
	if csa1 == nil || csa2 == nil {
		return false
	}
	if csa1.Name != csa2.Name {
		return false
	}
	if csa1.Path != csa2.Path {
		return false
	}
	if csa1.Expires != csa2.Expires {
		return false
	}
	if csa1.MaxAge != csa2.MaxAge {
		return false
	}
	if csa1.SameSite != csa2.SameSite {
		return false
	}
	if csa1.ConditionalSameSiteNone != csa2.ConditionalSameSiteNone {
		return false
	}

	return true
}

//Equal checks the equality between UpstreamByConfig types
func (u1 *UpstreamHashByConfig) Equal(u2 *UpstreamHashByConfig) bool {
	if u1 == u2 {
		return true
	}
	if u1 == nil || u2 == nil {
		return false
	}
	if u1.UpstreamHashBy != u2.UpstreamHashBy {
		return false
	}
	if u1.UpstreamHashBySubset != u2.UpstreamHashBySubset {
		return false
	}
	if u1.UpstreamHashBySubsetSize != u2.UpstreamHashBySubsetSize {
		return false
	}

	return true
}

// Equal checks the equality against an Endpoint
func (e1 *Endpoint) Equal(e2 *Endpoint) bool {
	if e1 == e2 {
		return true
	}
	if e1 == nil || e2 == nil {
		return false
	}
	if e1.Address != e2.Address {
		return false
	}
	if e1.Port != e2.Port {
		return false
	}

	if e1.Target != e2.Target {
		if e1.Target == nil || e2.Target == nil {
			return false
		}
		if e1.Target.UID != e2.Target.UID {
			return false
		}
		if e1.Target.ResourceVersion != e2.Target.ResourceVersion {
			return false
		}
	}

	return true
}

// Equal checks for equality between two TrafficShapingPolicies
func (tsp1 TrafficShapingPolicy) Equal(tsp2 TrafficShapingPolicy) bool {
	if tsp1.Weight != tsp2.Weight {
		return false
	}
	if tsp1.Header != tsp2.Header {
		return false
	}
	if tsp1.HeaderValue != tsp2.HeaderValue {
		return false
	}
	if tsp1.HeaderPattern != tsp2.HeaderPattern {
		return false
	}
	if tsp1.Cookie != tsp2.Cookie {
		return false
	}

	return true
}

// Equal tests for equality between two Server types
func (s1 *Server) Equal(s2 *Server) bool {
	if s1 == s2 {
		return true
	}
	if s1 == nil || s2 == nil {
		return false
	}
	if s1.Hostname != s2.Hostname {
		return false
	}
	if s1.SSLPassthrough != s2.SSLPassthrough {
		return false
	}
	if !(s1.SSLCert).Equal(s2.SSLCert) {
		return false
	}

	if len(s1.Aliases) != len(s2.Aliases) {
		return false
	}

	for _, a1 := range s1.Aliases {
		found := false
		for _, a2 := range s2.Aliases {
			if a1 == a2 {
				found = true
				break
			}
		}
		if !found {
			return false
		}
	}

	if s1.RedirectFromToWWW != s2.RedirectFromToWWW {
		return false
	}
	if !(&s1.CertificateAuth).Equal(&s2.CertificateAuth) {
		return false
	}
	if s1.ServerSnippet != s2.ServerSnippet {
		return false
	}
	if s1.SSLCiphers != s2.SSLCiphers {
		return false
	}
	if s1.SSLPreferServerCiphers != s2.SSLPreferServerCiphers {
		return false
	}
	if s1.AuthTLSError != s2.AuthTLSError {
		return false
	}

	if len(s1.Locations) != len(s2.Locations) {
		return false
	}

	// Location are sorted
	for idx, s1l := range s1.Locations {
		if !s1l.Equal(s2.Locations[idx]) {
			return false
		}
	}

	return true
}

// Equal tests for equality between two Location types
func (l1 *Location) Equal(l2 *Location) bool {
	if l1 == l2 {
		return true
	}
	if l1 == nil || l2 == nil {
		return false
	}
	if l1.Path != l2.Path {
		return false
	}
	if l1.IsDefBackend != l2.IsDefBackend {
		return false
	}
	if l1.Backend != l2.Backend {
		return false
	}

	if l1.Service != l2.Service {
		if l1.Service == nil || l2.Service == nil {
			return false
		}
		if l1.Service.GetNamespace() != l2.Service.GetNamespace() {
			return false
		}
		if l1.Service.GetName() != l2.Service.GetName() {
			return false
		}
	}

	if l1.Port.String() != l2.Port.String() {
		return false
	}
	if !(&l1.BasicDigestAuth).Equal(&l2.BasicDigestAuth) {
		return false
	}
	if l1.Denied != l2.Denied {
		return false
	}
	if !(&l1.CorsConfig).Equal(&l2.CorsConfig) {
		return false
	}
	if !(&l1.ExternalAuth).Equal(&l2.ExternalAuth) {
		return false
	}
	if l1.EnableGlobalAuth != l2.EnableGlobalAuth {
		return false
	}
	if l1.HTTP2PushPreload != l2.HTTP2PushPreload {
		return false
	}
	if !(&l1.RateLimit).Equal(&l2.RateLimit) {
		return false
	}
	if !(&l1.Redirect).Equal(&l2.Redirect) {
		return false
	}
	if !(&l1.Rewrite).Equal(&l2.Rewrite) {
		return false
	}
	if !(&l1.Whitelist).Equal(&l2.Whitelist) {
		return false
	}
	if !(&l1.Proxy).Equal(&l2.Proxy) {
		return false
	}
	if l1.UsePortInRedirects != l2.UsePortInRedirects {
		return false
	}
	if l1.ConfigurationSnippet != l2.ConfigurationSnippet {
		return false
	}
	if l1.ClientBodyBufferSize != l2.ClientBodyBufferSize {
		return false
	}
	if l1.UpstreamVhost != l2.UpstreamVhost {
		return false
	}
	if l1.XForwardedPrefix != l2.XForwardedPrefix {
		return false
	}
	if !(&l1.Connection).Equal(&l2.Connection) {
		return false
	}
	if !(&l1.Logs).Equal(&l2.Logs) {
		return false
	}

	if !(&l1.InfluxDB).Equal(&l2.InfluxDB) {
		return false
	}

	if l1.BackendProtocol != l2.BackendProtocol {
		return false
	}

	if !(&l1.FastCGI).Equal(&l2.FastCGI) {
		return false
	}

	match := compareInts(l1.CustomHTTPErrors, l2.CustomHTTPErrors)
	if !match {
		return false
	}

	if !(&l1.ModSecurity).Equal(&l2.ModSecurity) {
		return false
	}

	if l1.Satisfy != l2.Satisfy {
		return false
	}

	if l1.DefaultBackendUpstreamName != l2.DefaultBackendUpstreamName {
		return false
	}

	if !l1.Opentracing.Equal(&l2.Opentracing) {
		return false
	}

	if !l1.Mirror.Equal(&l2.Mirror) {
		return false
	}

	return true
}

// Equal tests for equality between two SSLPassthroughBackend types
func (ptb1 *SSLPassthroughBackend) Equal(ptb2 *SSLPassthroughBackend) bool {
	if ptb1 == ptb2 {
		return true
	}
	if ptb1 == nil || ptb2 == nil {
		return false
	}
	if ptb1.Backend != ptb2.Backend {
		return false
	}
	if ptb1.Hostname != ptb2.Hostname {
		return false
	}
	if ptb1.Port != ptb2.Port {
		return false
	}

	if ptb1.Service != ptb2.Service {
		if ptb1.Service == nil || ptb2.Service == nil {
			return false
		}
		if ptb1.Service.GetNamespace() != ptb2.Service.GetNamespace() {
			return false
		}
		if ptb1.Service.GetName() != ptb2.Service.GetName() {
			return false
		}
	}

	return true
}

// Equal tests for equality between two L4Service types
func (e1 *L4Service) Equal(e2 *L4Service) bool {
	if e1 == e2 {
		return true
	}
	if e1 == nil || e2 == nil {
		return false
	}
	if e1.Port != e2.Port {
		return false
	}
	if !(&e1.Backend).Equal(&e2.Backend) {
		return false
	}

	return compareEndpoints(e1.Endpoints, e2.Endpoints)
}

// Equal tests for equality between two L4Backend types
func (l4b1 *L4Backend) Equal(l4b2 *L4Backend) bool {
	if l4b1 == l4b2 {
		return true
	}
	if l4b1 == nil || l4b2 == nil {
		return false
	}
	if l4b1.Port != l4b2.Port {
		return false
	}
	if l4b1.Name != l4b2.Name {
		return false
	}
	if l4b1.Namespace != l4b2.Namespace {
		return false
	}
	if l4b1.Protocol != l4b2.Protocol {
		return false
	}
	if l4b1.ProxyProtocol != l4b2.ProxyProtocol {
		return false
	}

	return true
}

// Equal tests for equality between two SSLCert types
func (s1 *SSLCert) Equal(s2 *SSLCert) bool {
	if s1 == s2 {
		return true
	}
	if s1 == nil || s2 == nil {
		return false
	}
	if s1.CASHA != s2.CASHA {
		return false
	}
	if s1.PemSHA != s2.PemSHA {
		return false
	}
	if !s1.ExpireTime.Equal(s2.ExpireTime) {
		return false
	}
	if s1.PemCertKey != s2.PemCertKey {
		return false
	}
	if s1.UID != s2.UID {
		return false
	}

	return StringElementsMatch(s1.CN, s2.CN)
}

var compareEndpointsFunc = func(e1, e2 interface{}) bool {
	ep1, ok := e1.(Endpoint)
	if !ok {
		return false
	}

	ep2, ok := e2.(Endpoint)
	if !ok {
		return false
	}

	return (&ep1).Equal(&ep2)
}

func compareEndpoints(a, b []Endpoint) bool {
	return Compare(a, b, compareEndpointsFunc)
}

var compareBackendsFunc = func(e1, e2 interface{}) bool {
	b1, ok := e1.(*Backend)
	if !ok {
		return false
	}

	b2, ok := e2.(*Backend)
	if !ok {
		return false
	}

	return b1.Equal(b2)
}

func compareBackends(a, b []*Backend) bool {
	return Compare(a, b, compareBackendsFunc)
}

var compareIntsFunc = func(e1, e2 interface{}) bool {
	b1, ok := e1.(int)
	if !ok {
		return false
	}

	b2, ok := e2.(int)
	if !ok {
		return false
	}

	return b1 == b2
}

func compareInts(a, b []int) bool {
	return Compare(a, b, compareIntsFunc)
}

var compareL4ServiceFunc = func(e1, e2 interface{}) bool {
	b1, ok := e1.(L4Service)
	if !ok {
		return false
	}

	b2, ok := e2.(L4Service)
	if !ok {
		return false
	}

	return (&b1).Equal(&b2)
}

func compareL4Service(a, b []L4Service) bool {
	return Compare(a, b, compareL4ServiceFunc)
}

type equalFunction func(e1, e2 interface{}) bool

// Compare checks if the parameters are iterable and contains the same elements
func Compare(listA, listB interface{}, eq equalFunction) bool {
	ok := isIterable(listA)
	if !ok {
		return false
	}

	ok = isIterable(listB)
	if !ok {
		return false
	}

	a := reflect.ValueOf(listA)
	b := reflect.ValueOf(listB)

	if a.IsNil() && b.IsNil() {
		return true
	}

	if a.IsNil() != b.IsNil() {
		return false
	}

	if a.Len() != b.Len() {
		return false
	}

	visited := make([]bool, b.Len())

	for i := 0; i < a.Len(); i++ {
		found := false
		for j := 0; j < b.Len(); j++ {
			if visited[j] {
				continue
			}

			if eq(a.Index(i).Interface(), b.Index(j).Interface()) {
				visited[j] = true
				found = true
				break
			}
		}

		if !found {
			return false
		}
	}

	return true
}

var compareStrings = func(e1, e2 interface{}) bool {
	s1, ok := e1.(string)
	if !ok {
		return false
	}

	s2, ok := e2.(string)
	if !ok {
		return false
	}

	return s1 == s2
}

// StringElementsMatch compares two string slices and returns if are equals
func StringElementsMatch(a, b []string) bool {
	return Compare(a, b, compareStrings)
}

func isIterable(obj interface{}) bool {
	switch reflect.TypeOf(obj).Kind() {
	case reflect.Slice, reflect.Array:
		return true
	default:
		return false
	}
}

