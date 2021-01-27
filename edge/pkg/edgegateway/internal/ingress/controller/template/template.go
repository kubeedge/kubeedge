package template

import (
	"encoding/json"
	"github.com/kubeedge/kubeedge/edge/pkg/edgegateway/internal/ingress/controller/config"
	"github.com/pkg/errors"
	"io/ioutil"
	"k8s.io/klog/v2"
	"os"
	"os/exec"
	"strings"
	text_template "text/template"
)

const (
	slash                      = "/"
	nonIdempotent              = "non_idempotent"
	defBufferSize              = 65535
	defAuthSigninRedirectParam = "rd"
)

// Template ...
type Template struct {
	tmpl *text_template.Template
	//fw   watch.FileWatcher
	bp *BufferPool
}

// TemplateWriter is the interface to render a template
type TemplateWriter interface {
	Write(conf config.TemplateConfig) ([]byte, error)
}


//NewTemplate returns a new Template instance or an
//error if the specified template file contains errors
func NewTemplate(file string) (*Template, error) {
	data, err := ioutil.ReadFile(file)
	if err != nil {
		return nil, errors.Wrapf(err, "unexpected error reading template %v", file)
	}

	tmpl, err := text_template.New("nginx.tmpl").Funcs(funcMap).Parse(string(data))
	if err != nil {
		return nil, err
	}

	return &Template{
		tmpl: tmpl,
		bp:   NewBufferPool(defBufferSize),
	}, nil
}

// Write populates a buffer using a template with NGINX configuration
// and the servers and upstreams created by internal rules
func (t *Template) Write(conf config.TemplateConfig) ([]byte, error) {
	tmplBuf := t.bp.Get()
	defer t.bp.Put(tmplBuf)

	outCmdBuf := t.bp.Get()
	defer t.bp.Put(outCmdBuf)

	if klog.V(3).Enabled() {
		b, err := json.Marshal(conf)
		if err != nil {
			klog.Errorf("unexpected error: %v", err)
		}
		klog.InfoS("NGINX", "configuration", string(b))
	}

	err := t.tmpl.Execute(tmplBuf, conf)
	if err != nil {
		return nil, err
	}

	// squeezes multiple adjacent empty lines to be single
	// spaced this is to avoid the use of regular expressions
	cmd := exec.Command("/ingress-controller/clean-nginx-conf.sh")
	cmd.Stdin = tmplBuf
	cmd.Stdout = outCmdBuf
	if err := cmd.Run(); err != nil {
		klog.Warningf("unexpected error cleaning template: %v", err)
		return tmplBuf.Bytes(), nil
	}

	return outCmdBuf.Bytes(), nil
}


var (
	funcMap = text_template.FuncMap{
		"empty": func(input interface{}) bool {
			check, ok := input.(string)
			if ok {
				return len(check) == 0
			}
			return true
		},
		//"escapeLiteralDollar":             escapeLiteralDollar,
		//"buildLuaSharedDictionaries":      buildLuaSharedDictionaries,
		//"luaConfigurationRequestBodySize": luaConfigurationRequestBodySize,
		//"buildLocation":                   buildLocation,
		//"buildAuthLocation":               buildAuthLocation,
		//"shouldApplyGlobalAuth":           shouldApplyGlobalAuth,
		//"buildAuthResponseHeaders":        buildAuthResponseHeaders,
		//"buildAuthProxySetHeaders":        buildAuthProxySetHeaders,
		//"buildProxyPass":                  buildProxyPass,
		//"filterRateLimits":                filterRateLimits,
		//"buildRateLimitZones":             buildRateLimitZones,
		//"buildRateLimit":                  buildRateLimit,
		//"configForLua":                    configForLua,
		//"locationConfigForLua":            locationConfigForLua,
		//"buildResolvers":                  buildResolvers,
		//"buildUpstreamName":               buildUpstreamName,
		//"isLocationInLocationList":        isLocationInLocationList,
		//"isLocationAllowed":               isLocationAllowed,
		//"buildDenyVariable":               buildDenyVariable,
		"getenv":                          os.Getenv,
		"contains":                        strings.Contains,
		"hasPrefix":                       strings.HasPrefix,
		"hasSuffix":                       strings.HasSuffix,
		"trimSpace":                       strings.TrimSpace,
		"toUpper":                         strings.ToUpper,
		"toLower":                         strings.ToLower,
		//"formatIP":                        formatIP,
		//"quote":                           quote,
		//"buildNextUpstream":               buildNextUpstream,
		//"getIngressInformation":           getIngressInformation,
		//"serverConfig": func(all config.TemplateConfig, server *ingress.Server) interface{} {
		//	return struct{ First, Second interface{} }{all, server}
		//},
		//"isValidByteSize":                    isValidByteSize,
		//"buildForwardedFor":                  buildForwardedFor,
		//"buildAuthSignURL":                   buildAuthSignURL,
		//"buildAuthSignURLLocation":           buildAuthSignURLLocation,
		//"buildOpentracing":                   buildOpentracing,
		//"proxySetHeader":                     proxySetHeader,
		//"buildInfluxDB":                      buildInfluxDB,
		//"enforceRegexModifier":               enforceRegexModifier,
		//"buildCustomErrorDeps":               buildCustomErrorDeps,
		//"buildCustomErrorLocationsPerServer": buildCustomErrorLocationsPerServer,
		//"shouldLoadModSecurityModule":        shouldLoadModSecurityModule,
		//"buildHTTPListener":                  buildHTTPListener,
		//"buildHTTPSListener":                 buildHTTPSListener,
		//"buildOpentracingForLocation":        buildOpentracingForLocation,
		//"shouldLoadOpentracingModule":        shouldLoadOpentracingModule,
		//"buildModSecurityForLocation":        buildModSecurityForLocation,
		//"buildMirrorLocations":               buildMirrorLocations,
		//"shouldLoadAuthDigestModule":         shouldLoadAuthDigestModule,
		//"shouldLoadInfluxDBModule":           shouldLoadInfluxDBModule,
		//"buildServerName":                    buildServerName,
	}
)
