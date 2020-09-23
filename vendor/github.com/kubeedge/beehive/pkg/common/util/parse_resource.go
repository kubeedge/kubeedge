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
		return "", "", "", fmt.Errorf("Resource: %s format incorrect, or Operation: %s is not query/response", resource, operation)
	}
}

// ParseResourceMaster parses resource at master and returns cluster_id, node_id, namespace, resource_type, resource_id.
// If operation of msg is query list, return cluster_id, node_id, namespace, pod.
func ParseResourceMaster(resource string, operation string) (string, string, string, string, string, error) {
	resourceSplits := strings.Split(resource, "/")
	if len(resourceSplits) == 7 && resourceSplits[0] == "cluster" && resourceSplits[2] == "node" {
		return resourceSplits[1], resourceSplits[3], resourceSplits[4], resourceSplits[5], resourceSplits[6], nil
	} else if operation == model.QueryOperation || operation == model.ResponseOperation && len(resourceSplits) == 6 {
		return resourceSplits[1], resourceSplits[3], resourceSplits[4], resourceSplits[5], "", nil
	} else {
		return "", "", "", "", "", fmt.Errorf("Resource: %s format incorrect, or Operation: %s is not query/response", resource, operation)

	}
}
