package util

import (
	"fmt"
	"strings"

	"github.com/kubeedge/beehive/pkg/core/model"
)

// ParseResourceEdge parses resource at edge and returns namespace, resource_type, resource_id.
// If operation of msg is query list, return namespace, pod.
func ParseResourceEdge(resource string, operation string) (string, string, string, error) {
	resourceSplits := strings.Split(resource, "/")
	if len(resourceSplits) == 3 {
		return resourceSplits[0], resourceSplits[1], resourceSplits[2], nil
	} else if operation == model.QueryOperation || operation == model.ResponseOperation && len(resourceSplits) == 2 {
		return resourceSplits[0], resourceSplits[1], "", nil
	} else {
		return "", "", "", fmt.Errorf("resource: %s format incorrect, or Operation: %s is not query/response", resource, operation)
	}
}
