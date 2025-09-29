package messagelayer

import "fmt"

func BuildResourceForStreamRuleController(namespace, resourceType, resourceID string) (string, error) {
	if namespace == "" {
		namespace = "default"
	}
	if resourceID == "" || resourceType == "" {
		return "", fmt.Errorf("required parameter are not set (resourceID or resource type)")
	}
	return fmt.Sprintf("node/nodeid/%s/%s/%s", namespace, resourceType, resourceID), nil
}
