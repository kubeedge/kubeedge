package passthrough

import (
	"slices"
	"strings"
)

var commonDiscoveryPaths = []string{"/api", "/apis", "/version", "/healthz", "/livez", "/readyz", "/openapi/v2", "/openapi/v3"}

// isDiscoveryPath determines whether the uri is a Kubernetes API discovery path.
// Discovery paths are non-resource requests that allow anonymous access to get
// API server capabilities and resource types.
//
// This is a helper function used by both IsPassThroughPath and IsDiscoveryPath
// to ensure consistent path matching logic.
func isDiscoveryPath(path string) bool {
	// Normalize path by removing trailing slash and cleaning multiple slashes
	path = strings.TrimSuffix(path, "/")
	if path == "" {
		return false
	}

	// Exact match for common discovery paths
	if slices.Contains(commonDiscoveryPaths, path) {
		return true
	}

	parts := strings.Split(path, "/")
	// path starts with /, so parts[0] is always empty
	// /api/v1 -> ["", "api", "v1"] (len 3)
	// /apis/group/version -> ["", "apis", "group", "version"] (len 4)

	// Need at least 3 parts (empty, api|apis, resource/version)
	if len(parts) < 3 {
		return false
	}

	// Match /api/v1 or /api/v1beta1
	if parts[1] == "api" && len(parts) == 3 {
		return true
	}

	// Match /apis/<group> or /apis/<group>/<version>
	if parts[1] == "apis" && (len(parts) == 3 || len(parts) == 4) {
		return true
	}

	return false
}

// IsPassThroughPath determining whether the uri can be passed through
func IsPassThroughPath(path, verb string) bool {
	// Only GET requests can be passed through for discovery paths
	if verb != "get" {
		return false
	}

	// Use the same discovery path logic as IsDiscoveryPath
	return isDiscoveryPath(path)
}
