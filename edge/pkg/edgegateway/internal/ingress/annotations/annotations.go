package annotations

import (
	"github.com/imdario/mergo"
	"github.com/kubeedge/kubeedge/edge/pkg/edgegateway/internal/ingress/annotations/alias"
	"github.com/kubeedge/kubeedge/edge/pkg/edgegateway/internal/ingress/annotations/auth"
	"github.com/kubeedge/kubeedge/edge/pkg/edgegateway/internal/ingress/annotations/authreq"
	"github.com/kubeedge/kubeedge/edge/pkg/edgegateway/internal/ingress/annotations/authreqglobal"
	"github.com/kubeedge/kubeedge/edge/pkg/edgegateway/internal/ingress/annotations/authtls"
	"github.com/kubeedge/kubeedge/edge/pkg/edgegateway/internal/ingress/annotations/backendprotocol"
	"github.com/kubeedge/kubeedge/edge/pkg/edgegateway/internal/ingress/annotations/canary"
	"github.com/kubeedge/kubeedge/edge/pkg/edgegateway/internal/ingress/annotations/clientbodybuffersize"
	"github.com/kubeedge/kubeedge/edge/pkg/edgegateway/internal/ingress/annotations/connection"
	"github.com/kubeedge/kubeedge/edge/pkg/edgegateway/internal/ingress/annotations/cors"
	"github.com/kubeedge/kubeedge/edge/pkg/edgegateway/internal/ingress/annotations/customhttperrors"
	"github.com/kubeedge/kubeedge/edge/pkg/edgegateway/internal/ingress/annotations/defaultbackend"
	"github.com/kubeedge/kubeedge/edge/pkg/edgegateway/internal/ingress/annotations/fastcgi"
	"github.com/kubeedge/kubeedge/edge/pkg/edgegateway/internal/ingress/annotations/http2pushpreload"
	"github.com/kubeedge/kubeedge/edge/pkg/edgegateway/internal/ingress/annotations/influxdb"
	"github.com/kubeedge/kubeedge/edge/pkg/edgegateway/internal/ingress/annotations/ipwhitelist"
	"github.com/kubeedge/kubeedge/edge/pkg/edgegateway/internal/ingress/annotations/loadbalancing"
	"github.com/kubeedge/kubeedge/edge/pkg/edgegateway/internal/ingress/annotations/log"
	"github.com/kubeedge/kubeedge/edge/pkg/edgegateway/internal/ingress/annotations/mirror"
	"github.com/kubeedge/kubeedge/edge/pkg/edgegateway/internal/ingress/annotations/modsecurity"
	"github.com/kubeedge/kubeedge/edge/pkg/edgegateway/internal/ingress/annotations/opentracing"
	"github.com/kubeedge/kubeedge/edge/pkg/edgegateway/internal/ingress/annotations/parser"
	"github.com/kubeedge/kubeedge/edge/pkg/edgegateway/internal/ingress/annotations/portinredirect"
	"github.com/kubeedge/kubeedge/edge/pkg/edgegateway/internal/ingress/annotations/proxy"
	"github.com/kubeedge/kubeedge/edge/pkg/edgegateway/internal/ingress/annotations/proxyssl"
	"github.com/kubeedge/kubeedge/edge/pkg/edgegateway/internal/ingress/annotations/ratelimit"
	"github.com/kubeedge/kubeedge/edge/pkg/edgegateway/internal/ingress/annotations/redirect"
	"github.com/kubeedge/kubeedge/edge/pkg/edgegateway/internal/ingress/annotations/rewrite"
	"github.com/kubeedge/kubeedge/edge/pkg/edgegateway/internal/ingress/annotations/satisfy"
	"github.com/kubeedge/kubeedge/edge/pkg/edgegateway/internal/ingress/annotations/secureupstream"
	"github.com/kubeedge/kubeedge/edge/pkg/edgegateway/internal/ingress/annotations/serversnippet"
	"github.com/kubeedge/kubeedge/edge/pkg/edgegateway/internal/ingress/annotations/serviceupstream"
	"github.com/kubeedge/kubeedge/edge/pkg/edgegateway/internal/ingress/annotations/sessionaffinity"
	"github.com/kubeedge/kubeedge/edge/pkg/edgegateway/internal/ingress/annotations/snippet"
	"github.com/kubeedge/kubeedge/edge/pkg/edgegateway/internal/ingress/annotations/sslcipher"
	"github.com/kubeedge/kubeedge/edge/pkg/edgegateway/internal/ingress/annotations/sslpassthrough"
	"github.com/kubeedge/kubeedge/edge/pkg/edgegateway/internal/ingress/annotations/upstreamhashby"
	"github.com/kubeedge/kubeedge/edge/pkg/edgegateway/internal/ingress/annotations/upstreamvhost"
	"github.com/kubeedge/kubeedge/edge/pkg/edgegateway/internal/ingress/annotations/xforwardedprefix"
	"github.com/kubeedge/kubeedge/edge/pkg/edgegateway/internal/ingress/errors"
	"github.com/kubeedge/kubeedge/edge/pkg/edgegateway/internal/ingress/resolver"
	"k8s.io/klog/v2"

	apiv1 "k8s.io/api/core/v1"
	networking "k8s.io/api/networking/v1beta1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// DeniedKeyName name of the key that contains the reason to deny a location
const DeniedKeyName = "Denied"

// Ingress defines the valid annotations present in one NGINX Ingress rule
type Ingress struct {
	metav1.ObjectMeta
	BackendProtocol      string
	Aliases              []string
	BasicDigestAuth      auth.Config
	Canary               canary.Config
	CertificateAuth      authtls.Config
	ClientBodyBufferSize string
	ConfigurationSnippet string
	Connection           connection.Config
	CorsConfig           cors.Config
	CustomHTTPErrors     []int
	DefaultBackend       *apiv1.Service
	//TODO: Change this back into an error when https://github.com/imdario/mergo/issues/100 is resolved
	FastCGI            fastcgi.Config
	Denied             *string
	ExternalAuth       authreq.Config
	EnableGlobalAuth   bool
	HTTP2PushPreload   bool
	Opentracing        opentracing.Config
	Proxy              proxy.Config
	ProxySSL           proxyssl.Config
	RateLimit          ratelimit.Config
	Redirect           redirect.Config
	Rewrite            rewrite.Config
	Satisfy            string
	SecureUpstream     secureupstream.Config
	ServerSnippet      string
	ServiceUpstream    bool
	SessionAffinity    sessionaffinity.Config
	SSLPassthrough     bool
	UsePortInRedirects bool
	UpstreamHashBy     upstreamhashby.Config
	LoadBalancing      string
	UpstreamVhost      string
	Whitelist          ipwhitelist.SourceRange
	XForwardedPrefix   string
	SSLCipher          sslcipher.Config
	Logs               log.Config
	InfluxDB           influxdb.Config
	ModSecurity        modsecurity.Config
	Mirror             mirror.Config
}

// Extractor defines the annotation parsers to be used in the extraction of annotations
type Extractor struct {
	annotations map[string]parser.IngressAnnotation
}

// NewAnnotationExtractor creates a new annotations extractor
func NewAnnotationExtractor(cfg resolver.Resolver) Extractor {
	return Extractor{
		map[string]parser.IngressAnnotation{
			"Aliases":              alias.NewParser(cfg),
			"BasicDigestAuth":      auth.NewParser(auth.AuthDirectory, cfg),
			"Canary":               canary.NewParser(cfg),
			"CertificateAuth":      authtls.NewParser(cfg),
			"ClientBodyBufferSize": clientbodybuffersize.NewParser(cfg),
			"ConfigurationSnippet": snippet.NewParser(cfg),
			"Connection":           connection.NewParser(cfg),
			"CorsConfig":           cors.NewParser(cfg),
			"CustomHTTPErrors":     customhttperrors.NewParser(cfg),
			"DefaultBackend":       defaultbackend.NewParser(cfg),
			"FastCGI":              fastcgi.NewParser(cfg),
			"ExternalAuth":         authreq.NewParser(cfg),
			"EnableGlobalAuth":     authreqglobal.NewParser(cfg),
			"HTTP2PushPreload":     http2pushpreload.NewParser(cfg),
			"Opentracing":          opentracing.NewParser(cfg),
			"Proxy":                proxy.NewParser(cfg),
			"ProxySSL":             proxyssl.NewParser(cfg),
			"RateLimit":            ratelimit.NewParser(cfg),
			"Redirect":             redirect.NewParser(cfg),
			"Rewrite":              rewrite.NewParser(cfg),
			"Satisfy":              satisfy.NewParser(cfg),
			"SecureUpstream":       secureupstream.NewParser(cfg),
			"ServerSnippet":        serversnippet.NewParser(cfg),
			"ServiceUpstream":      serviceupstream.NewParser(cfg),
			"SessionAffinity":      sessionaffinity.NewParser(cfg),
			"SSLPassthrough":       sslpassthrough.NewParser(cfg),
			"UsePortInRedirects":   portinredirect.NewParser(cfg),
			"UpstreamHashBy":       upstreamhashby.NewParser(cfg),
			"LoadBalancing":        loadbalancing.NewParser(cfg),
			"UpstreamVhost":        upstreamvhost.NewParser(cfg),
			"Whitelist":            ipwhitelist.NewParser(cfg),
			"XForwardedPrefix":     xforwardedprefix.NewParser(cfg),
			"SSLCipher":            sslcipher.NewParser(cfg),
			"Logs":                 log.NewParser(cfg),
			"InfluxDB":             influxdb.NewParser(cfg),
			"BackendProtocol":      backendprotocol.NewParser(cfg),
			"ModSecurity":          modsecurity.NewParser(cfg),
			"Mirror":               mirror.NewParser(cfg),
		},
	}
}

// Extract extracts the annotations from an Ingress
func (e Extractor) Extract(ing *networking.Ingress) *Ingress {
	pia := &Ingress{
		ObjectMeta: ing.ObjectMeta,
	}

	data := make(map[string]interface{})
	for name, annotationParser := range e.annotations {
		val, err := annotationParser.Parse(ing)
		klog.V(5).InfoS("Parsing internal annotation", "name", name, "ingress", klog.KObj(ing), "value", val)
		if err != nil {
			if errors.IsMissingAnnotations(err) {
				continue
			}

			if !errors.IsLocationDenied(err) {
				continue
			}

			if name == "CertificateAuth" && data[name] == nil {
				data[name] = authtls.Config{
					AuthTLSError: err.Error(),
				}
				// avoid mapping the result from the annotation
				val = nil
			}

			_, alreadyDenied := data[DeniedKeyName]
			if !alreadyDenied {
				errString := err.Error()
				data[DeniedKeyName] = &errString
				klog.ErrorS(err, "error reading internal annotation", "name", name, "ingress", klog.KObj(ing))
				continue
			}

			klog.V(5).Error(err, "error reading internal annotation", "name", name, "ingress", klog.KObj(ing))
		}

		if val != nil {
			data[name] = val
		}
	}

	err := mergo.MapWithOverwrite(pia, data)
	if err != nil {
		klog.ErrorS(err, "unexpected error merging extracted annotations")
	}

	return pia
}