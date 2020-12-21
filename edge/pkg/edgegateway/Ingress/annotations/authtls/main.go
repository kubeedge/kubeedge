package authtls

import (
	"github.com/kubeedge/kubeedge/edge/pkg/edgegateway/Ingress/annotations/parser"
	"github.com/kubeedge/kubeedge/edge/pkg/edgegateway/Ingress/controller"
	defaults "github.com/kubeedge/kubeedge/edge/pkg/edgegateway/Ingress/default"
	"github.com/pkg/errors"
	networking "k8s.io/api/networking/v1beta1"
	"regexp"
)

const (
	defaultAuthTLSDepth     = 1
	defaultAuthVerifyClient = "on"
)

var (
	authVerifyClientRegex = regexp.MustCompile(`on|off|optional|optional_no_ca`)
)

// Config contains the AuthSSLCert used for mutual authentication
// and the configured ValidationDepth
type Config struct {
	defaults.AuthSSLCert
	VerifyClient       string `json:"verify_client"`
	ValidationDepth    int    `json:"validationDepth"`
	ErrorPage          string `json:"errorPage"`
	PassCertToUpstream bool   `json:"passCertToUpstream"`
	AuthTLSError       string
}

// Equal tests for equality between two Config types
func (assl1 *Config) Equal(assl2 *Config) bool {
	if assl1 == assl2 {
		return true
	}
	if assl1 == nil || assl2 == nil {
		return false
	}
	if !(&assl1.AuthSSLCert).Equal(&assl2.AuthSSLCert) {
		return false
	}
	if assl1.VerifyClient != assl2.VerifyClient {
		return false
	}
	if assl1.ValidationDepth != assl2.ValidationDepth {
		return false
	}
	if assl1.ErrorPage != assl2.ErrorPage {
		return false
	}
	if assl1.PassCertToUpstream != assl2.PassCertToUpstream {
		return false
	}

	return true
}

// NewParser creates a new TLS authentication annotation parser
func NewParser(resolver defaults.Resolver) parser.IngressAnnotation {
	return authTLS{resolver}
}

type authTLS struct {
	r defaults.Resolver
}

// Parse parses the annotations contained in the ingress
// rule used to use a Certificate as authentication method
func (a authTLS) Parse(ing *networking.Ingress) (interface{}, error) {
	var err error
	config := &Config{}

	tlsauthsecret, err := parser.GetStringAnnotation("auth-tls-secret", ing)
	if err != nil {
		return &Config{}, err
	}

	_, _, err = controller.ParseNameNS(tlsauthsecret)
	if err != nil {
		return &Config{}, defaults.NewLocationDenied(err.Error())
	}

	authCert, err := a.r.GetAuthCertificate(tlsauthsecret)
	if err != nil {
		e := errors.Wrap(err, "error obtaining certificate")
		return &Config{}, defaults.LocationDenied{Reason: e}

	}
	config.AuthSSLCert = *authCert

	config.VerifyClient, err = parser.GetStringAnnotation("auth-tls-verify-client", ing)
	if err != nil || !authVerifyClientRegex.MatchString(config.VerifyClient) {
		config.VerifyClient = defaultAuthVerifyClient
	}

	config.ValidationDepth, err = parser.GetIntAnnotation("auth-tls-verify-depth", ing)
	if err != nil || config.ValidationDepth == 0 {
		config.ValidationDepth = defaultAuthTLSDepth
	}

	config.ErrorPage, err = parser.GetStringAnnotation("auth-tls-error-page", ing)
	if err != nil {
		config.ErrorPage = ""
	}

	config.PassCertToUpstream, err = parser.GetBoolAnnotation("auth-tls-pass-certificate-to-upstream", ing)
	if err != nil {
		config.PassCertToUpstream = false
	}

	return config, nil
}
