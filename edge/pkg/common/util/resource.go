package util

import (
	"fmt"
	"strings"

	authenticationv1 "k8s.io/api/authentication/v1"

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

// TokenRequestKeyFunc keys should be nonconfidential and safe to log
func TokenRequestKeyFunc(name, namespace string, tr *authenticationv1.TokenRequest) string {
	var exp int64
	if tr.Spec.ExpirationSeconds != nil {
		exp = *tr.Spec.ExpirationSeconds
	}

	var ref authenticationv1.BoundObjectReference
	if tr.Spec.BoundObjectRef != nil {
		ref = *tr.Spec.BoundObjectRef
	}

	return fmt.Sprintf("%q/%q/%#v/%#v/%#v", name, namespace, tr.Spec.Audiences, exp, ref)
}
