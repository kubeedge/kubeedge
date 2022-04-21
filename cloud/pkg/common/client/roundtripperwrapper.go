package client

import (
	"net/http"

	commontypes "github.com/kubeedge/kubeedge/common/types"
)

func AuthProxyRoundTripperWrapper(rt http.RoundTripper) http.RoundTripper {
	return &authProxyRoundTripper{rt: rt}
}

type authProxyRoundTripper struct {
	rt http.RoundTripper
}

func (rt *authProxyRoundTripper) RoundTrip(req *http.Request) (*http.Response, error) {
	val := req.Context().Value(commontypes.AuthorizationKey)
	if val != nil {
		if auth, ok := val.(string); ok {
			req.Header.Set(string(commontypes.AuthorizationKey), auth)
		}
	}
	return rt.rt.RoundTrip(req)
}
