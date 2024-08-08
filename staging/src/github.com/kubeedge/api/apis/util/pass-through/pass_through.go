package passthrough

type passRequest string

const (
	versionRequest passRequest = "/version::get"
	healthRequest  passRequest = "/healthz::get" // deprecated: TODO remove this once it is gone
	liveRequest    passRequest = "/livez::get"
	readyRequest   passRequest = "/readyz::get"
)

var passThroughMap = map[passRequest]bool{
	versionRequest: true,
	healthRequest:  true,
	liveRequest:    true,
	readyRequest:   true,
}

// IsPassThroughPath determining whether the uri can be passed through
func IsPassThroughPath(path, verb string) bool {
	return passThroughMap[passRequest(path+"::"+verb)]
}
