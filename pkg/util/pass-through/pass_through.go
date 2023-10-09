package passthrough

type passRequest string

const (
	versionRequest passRequest = "/version::get"
)

var passThroughMap = map[passRequest]bool{
	versionRequest: true,
}

// IsPassThroughPath determining whether the uri can be passed through
func IsPassThroughPath(path, verb string) bool {
	return passThroughMap[passRequest(path+"::"+verb)]
}
