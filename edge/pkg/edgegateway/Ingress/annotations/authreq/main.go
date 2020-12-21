package authreq

import (
	"fmt"
	"github.com/kubeedge/kubeedge/edge/pkg/edgegateway/Ingress/annotations/parser"
	defaults "github.com/kubeedge/kubeedge/edge/pkg/edgegateway/Ingress/default"
	networking "k8s.io/api/networking/v1beta1"
	"k8s.io/klog"
	"regexp"
	"strings"
)

// Config returns external authentication configuration for an Ingress rule
type Config struct {
	URL string `json:"url"`
	// Host contains the hostname defined in the URL
	Host                   string            `json:"host"`
	SigninURL              string            `json:"signinUrl"`
	SigninURLRedirectParam string            `json:"signinUrlRedirectParam,omitempty"`
	Method                 string            `json:"method"`
	ResponseHeaders        []string          `json:"responseHeaders,omitempty"`
	RequestRedirect        string            `json:"requestRedirect"`
	AuthSnippet            string            `json:"authSnippet"`
	AuthCacheKey           string            `json:"authCacheKey"`
	AuthCacheDuration      []string          `json:"authCacheDuration"`
	ProxySetHeaders        map[string]string `json:"proxySetHeaders,omitempty"`
}

// DefaultCacheDuration is the fallback value if no cache duration is provided
const DefaultCacheDuration = "200 202 401 5m"

// Equal tests for equality between two Config types
func (e1 *Config) Equal(e2 *Config) bool {
	if e1 == e2 {
		return true
	}
	if e1 == nil || e2 == nil {
		return false
	}
	if e1.URL != e2.URL {
		return false
	}
	if e1.Host != e2.Host {
		return false
	}
	if e1.SigninURL != e2.SigninURL {
		return false
	}
	if e1.SigninURLRedirectParam != e2.SigninURLRedirectParam {
		return false
	}
	if e1.Method != e2.Method {
		return false
	}

	match := defaults.StringElementsMatch(e1.ResponseHeaders, e2.ResponseHeaders)
	if !match {
		return false
	}

	if e1.RequestRedirect != e2.RequestRedirect {
		return false
	}
	if e1.AuthSnippet != e2.AuthSnippet {
		return false
	}

	if e1.AuthCacheKey != e2.AuthCacheKey {
		return false
	}

	return defaults.StringElementsMatch(e1.AuthCacheDuration, e2.AuthCacheDuration)
}

var (
	methods         = []string{"GET", "HEAD", "POST", "PUT", "PATCH", "DELETE", "CONNECT", "OPTIONS", "TRACE"}
	headerRegexp    = regexp.MustCompile(`^[a-zA-Z\d\-_]+$`)
	statusCodeRegex = regexp.MustCompile(`^[\d]{3}$`)
	durationRegex   = regexp.MustCompile(`^[\d]+(ms|s|m|h|d|w|M|y)$`) // see http://nginx.org/en/docs/syntax.html
)

// ValidMethod checks is the provided string a valid HTTP method
func ValidMethod(method string) bool {
	if len(method) == 0 {
		return false
	}

	for _, m := range methods {
		if method == m {
			return true
		}
	}
	return false
}

// ValidHeader checks is the provided string satisfies the header's name regex
func ValidHeader(header string) bool {
	return headerRegexp.Match([]byte(header))
}

// ValidCacheDuration checks if the provided string is a valid cache duration
// spec: [code ...] [time ...];
// with: code is an http status code
//       time must match the time regex and may appear multiple times, e.g. `1h 30m`
func ValidCacheDuration(duration string) bool {
	elements := strings.Split(duration, " ")
	seenDuration := false

	for _, element := range elements {
		if len(element) == 0 {
			continue
		}
		if statusCodeRegex.Match([]byte(element)) {
			if seenDuration {
				return false // code after duration
			}
			continue
		}
		if durationRegex.Match([]byte(element)) {
			seenDuration = true
		}
	}
	return seenDuration
}

type authReq struct {
	r defaults.Resolver
}

// NewParser creates a new authentication request annotation parser
func NewParser(r defaults.Resolver) parser.IngressAnnotation {
	return authReq{r}
}

// ParseAnnotations parses the annotations contained in the ingress
// rule used to use an Config URL as source for authentication
func (a authReq) Parse(ing *networking.Ingress) (interface{}, error) {
	// Required Parameters
	urlString, err := parser.GetStringAnnotation("auth-url", ing)
	if err != nil {
		return nil, err
	}

	authURL, err := parser.StringToURL(urlString)
	if err != nil {
		return nil, defaults.InvalidContent{Name: err.Error()}
	}

	authMethod, _ := parser.GetStringAnnotation("auth-method", ing)
	if len(authMethod) != 0 && !ValidMethod(authMethod) {
		return nil, defaults.NewLocationDenied("invalid HTTP method")
	}

	// Optional Parameters
	signIn, err := parser.GetStringAnnotation("auth-signin", ing)
	if err != nil {
		klog.V(3).Info("auth-signin annotation is undefined and will not be set")
	}

	signInRedirectParam, err := parser.GetStringAnnotation("auth-signin-redirect-param", ing)
	if err != nil {
		klog.V(3).Infof("auth-signin-redirect-param annotation is undefined and will not be set")
	}

	authSnippet, err := parser.GetStringAnnotation("auth-snippet", ing)
	if err != nil {
		klog.V(3).Info("auth-snippet annotation is undefined and will not be set")
	}

	authCacheKey, err := parser.GetStringAnnotation("auth-cache-key", ing)
	if err != nil {
		klog.V(3).Info("auth-cache-key annotation is undefined and will not be set")
	}

	durstr, _ := parser.GetStringAnnotation("auth-cache-duration", ing)
	authCacheDuration, err := ParseStringToCacheDurations(durstr)
	if err != nil {
		return nil, err
	}

	responseHeaders := []string{}
	hstr, _ := parser.GetStringAnnotation("auth-response-headers", ing)
	if len(hstr) != 0 {
		harr := strings.Split(hstr, ",")
		for _, header := range harr {
			header = strings.TrimSpace(header)
			if len(header) > 0 {
				if !ValidHeader(header) {
					return nil, defaults.NewLocationDenied("invalid headers list")
				}
				responseHeaders = append(responseHeaders, header)
			}
		}
	}

	proxySetHeaderMap, err := parser.GetStringAnnotation("auth-proxy-set-headers", ing)
	if err != nil {
		klog.V(3).Info("auth-set-proxy-headers annotation is undefined and will not be set")
	}

	var proxySetHeaders map[string]string

	if proxySetHeaderMap != "" {
		proxySetHeadersMapContents, err := a.r.GetConfigMap(proxySetHeaderMap)
		if err != nil {
			return nil, defaults.NewLocationDenied(fmt.Sprintf("unable to find configMap %q", proxySetHeaderMap))
		}

		for header := range proxySetHeadersMapContents.Data {
			if !ValidHeader(header) {
				return nil, defaults.NewLocationDenied("invalid proxy-set-headers in configmap")
			}
		}

		proxySetHeaders = proxySetHeadersMapContents.Data
	}

	requestRedirect, _ := parser.GetStringAnnotation("auth-request-redirect", ing)

	return &Config{
		URL:                    urlString,
		Host:                   authURL.Hostname(),
		SigninURL:              signIn,
		SigninURLRedirectParam: signInRedirectParam,
		Method:                 authMethod,
		ResponseHeaders:        responseHeaders,
		RequestRedirect:        requestRedirect,
		AuthSnippet:            authSnippet,
		AuthCacheKey:           authCacheKey,
		AuthCacheDuration:      authCacheDuration,
		ProxySetHeaders:        proxySetHeaders,
	}, nil
}

// ParseStringToCacheDurations parses and validates the provided string
// into a list of cache durations.
// It will always return at least one duration (the default duration)
func ParseStringToCacheDurations(input string) ([]string, error) {
	authCacheDuration := []string{}
	if len(input) != 0 {
		arr := strings.Split(input, ",")
		for _, duration := range arr {
			duration = strings.TrimSpace(duration)
			if len(duration) > 0 {
				if !ValidCacheDuration(duration) {
					authCacheDuration = []string{DefaultCacheDuration}
					return authCacheDuration, defaults.NewLocationDenied(fmt.Sprintf("invalid cache duration: %s", duration))
				}
				authCacheDuration = append(authCacheDuration, duration)
			}
		}
	}

	if len(authCacheDuration) == 0 {
		authCacheDuration = append(authCacheDuration, DefaultCacheDuration)
	}
	return authCacheDuration, nil
}
