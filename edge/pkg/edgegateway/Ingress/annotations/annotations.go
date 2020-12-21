package annotations

import (
	log "github.com/go-chassis/paas-lager"
	"github.com/imdario/mergo"
	"github.com/kubeedge/kubeedge/edge/pkg/edgegateway/Ingress/annotations/authreq"
	"github.com/kubeedge/kubeedge/edge/pkg/edgegateway/Ingress/annotations/authtls"
	"github.com/kubeedge/kubeedge/edge/pkg/edgegateway/Ingress/annotations/parser"
	defaults "github.com/kubeedge/kubeedge/edge/pkg/edgegateway/Ingress/default"
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
	//BasicDigestAuth      auth.Config
	//Canary               canary.Config
	CertificateAuth      authtls.Config
	ClientBodyBufferSize string
	ConfigurationSnippet string
	//Connection           connection.Config
	//CorsConfig           cors.Config
	CustomHTTPErrors     []int
	DefaultBackend       *apiv1.Service
	//TODO: Change this back into an error when https://github.com/imdario/mergo/issues/100 is resolved
	//FastCGI            fastcgi.Config
	Denied             *string
	ExternalAuth       authreq.Config
	EnableGlobalAuth   bool
	HTTP2PushPreload   bool
	//Opentracing        opentracing.Config
	//Proxy              proxy.Config
	//ProxySSL           proxyssl.Config
	//RateLimit          ratelimit.Config
	//Redirect           redirect.Config
	//Rewrite            rewrite.Config
	Satisfy            string
	//SecureUpstream     secureupstream.Config
	ServerSnippet      string
	ServiceUpstream    bool
	//SessionAffinity    sessionaffinity.Config
	SSLPassthrough     bool
	UsePortInRedirects bool
	//UpstreamHashBy     upstreamhashby.Config
	LoadBalancing      string
	UpstreamVhost      string
	//Whitelist          ipwhitelist.SourceRange
	XForwardedPrefix   string
	//SSLCipher          sslcipher.Config
	Logs               log.Config
	//InfluxDB           influxdb.Config
	//ModSecurity        modsecurity.Config
	//Mirror             mirror.Config
}

// Extractor defines the annotation parsers to be used in the extraction of annotations
type Extractor struct {
	annotations map[string]parser.IngressAnnotation
}

// NewAnnotationExtractor creates a new annotations extractor
func NewAnnotationExtractor(cfg defaults.Resolver) Extractor {
	return Extractor{
		map[string]parser.IngressAnnotation{
			//"Aliases":              alias.NewParser(cfg),
			//"BasicDigestAuth":      auth.NewParser(auth.AuthDirectory, cfg),
			//"Canary":               canary.NewParser(cfg),
			"CertificateAuth":      authtls.NewParser(cfg),
			//"ClientBodyBufferSize": clientbodybuffersize.NewParser(cfg),
			//"ConfigurationSnippet": snippet.NewParser(cfg),
			//"Connection":           connection.NewParser(cfg),
			//"CorsConfig":           cors.NewParser(cfg),
			//"CustomHTTPErrors":     customhttperrors.NewParser(cfg),
			//"DefaultBackend":       defaultbackend.NewParser(cfg),
			//"FastCGI":              fastcgi.NewParser(cfg),
			"ExternalAuth":         authreq.NewParser(cfg),
			//"EnableGlobalAuth":     authreqglobal.NewParser(cfg),
			//"HTTP2PushPreload":     http2pushpreload.NewParser(cfg),
			//"Opentracing":          opentracing.NewParser(cfg),
			//"Proxy":                proxy.NewParser(cfg),
			//"ProxySSL":             proxyssl.NewParser(cfg),
			//"RateLimit":            ratelimit.NewParser(cfg),
			//"Redirect":             redirect.NewParser(cfg),
			//"Rewrite":              rewrite.NewParser(cfg),
			//"Satisfy":              satisfy.NewParser(cfg),
			//"SecureUpstream":       secureupstream.NewParser(cfg),
			//"ServerSnippet":        serversnippet.NewParser(cfg),
			//"ServiceUpstream":      serviceupstream.NewParser(cfg),
			//"SessionAffinity":      sessionaffinity.NewParser(cfg),
			//"SSLPassthrough":       sslpassthrough.NewParser(cfg),
			//"UsePortInRedirects":   portinredirect.NewParser(cfg),
			//"UpstreamHashBy":       upstreamhashby.NewParser(cfg),
			//"LoadBalancing":        loadbalancing.NewParser(cfg),
			//"UpstreamVhost":        upstreamvhost.NewParser(cfg),
			//"Whitelist":            ipwhitelist.NewParser(cfg),
			//"XForwardedPrefix":     xforwardedprefix.NewParser(cfg),
			//"SSLCipher":            sslcipher.NewParser(cfg),
			//"Logs":                 log.NewParser(cfg),
			//"InfluxDB":             influxdb.NewParser(cfg),
			//"BackendProtocol":      backendprotocol.NewParser(cfg),
			//"ModSecurity":          modsecurity.NewParser(cfg),
			//"Mirror":               mirror.NewParser(cfg),
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
		klog.V(5).InfoS("Parsing Ingress annotation", "name", name, "ingress", klog.KObj(ing), "value", val)
		if err != nil {
			if defaults.IsMissingAnnotations(err) {
				continue
			}

			if !defaults.IsLocationDenied(err) {
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
				klog.ErrorS(err, "error reading Ingress annotation", "name", name, "ingress", klog.KObj(ing))
				continue
			}

			klog.V(5).Error(err, "error reading Ingress annotation", "name", name, "ingress", klog.KObj(ing))
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