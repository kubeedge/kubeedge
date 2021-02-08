package handlerfactory

/*
import (
	"net/http"

	utilnet "k8s.io/apimachinery/pkg/util/net"
)

const (
	maxUserAgentLength      = 1024
	userAgentTruncateSuffix = "...TRUNCATED"
)

// lazyTruncatedUserAgent implements String() string and it will
// return user-agent which may be truncated.
type lazyTruncatedUserAgent struct {
	req *http.Request
}

func (lazy *lazyTruncatedUserAgent) String() string {
	ua := "unknown"
	if lazy.req != nil {
		ua = utilnet.GetHTTPClient(lazy.req)
		if len(ua) > maxUserAgentLength {
			ua = ua[:maxUserAgentLength] + userAgentTruncateSuffix
		}
	}
	return ua
}

// LazyClientIP implements String() string and it will
// calls GetClientIP() lazily only when required.
type lazyClientIP struct {
	req *http.Request
}

func (lazy *lazyClientIP) String() string {
	if lazy.req != nil {
		if ip := utilnet.GetClientIP(lazy.req); ip != nil {
			return ip.String()
		}
	}
	return "unknown"
}
*/
