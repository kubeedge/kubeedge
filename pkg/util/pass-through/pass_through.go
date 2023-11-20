package passthrough

import (
	"regexp"
)

type passRequest string

const (
	versionRequest passRequest = "/version::get"
)

var passThroughMap = map[passRequest]bool{
	versionRequest: true,
}

// IsPassThroughPath determining whether the uri can be passed through
func IsPassThroughPath(path, verb string) bool {
	for request, b := range passThroughMap {
		reg := regexp.MustCompile(string(request))
		if reg == nil {
			continue
		}
		if reg.Match([]byte(path + "::" + verb)) {
			return b
		}
	}
	return false
}
