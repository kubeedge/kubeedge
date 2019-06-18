/*
Copyright 2019 The KubeEdge Authors.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

   http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package utils

import (
	"fmt"
	"strings"

	"github.com/kubeedge/beehive/pkg/core/model"
	constants "github.com/kubeedge/kubeedge/common/constants"
	testconstants "github.com/kubeedge/kubeedge/tests/stubs/common/constants"
)

// BuildResource return a string as "beehive/pkg/core/model".Message.Router.Resource
func BuildResource(nodeID, namespace, resourceType, resourceID string) (resource string, err error) {
	if nodeID == "" || namespace == "" || resourceType == "" {
		err = fmt.Errorf("Required parameter are not set (node id, namespace or resource type)")
		return
	}
	resource = fmt.Sprintf("%s%s%s%s%s%s%s", testconstants.ResourceNode, constants.ResourceSep, nodeID, constants.ResourceSep, namespace, constants.ResourceSep, resourceType)
	if resourceID != "" {
		resource += fmt.Sprintf("%s%s", constants.ResourceSep, resourceID)
	}
	return
}

// GetNodeID from "beehive/pkg/core/model".Message.Router.Resource
func GetNodeID(msg model.Message) (string, error) {
	sli := strings.Split(msg.GetResource(), constants.ResourceSep)
	if len(sli) <= testconstants.ResourceNodeIDIndex {
		return "", fmt.Errorf("Node id not found")
	}
	return sli[testconstants.ResourceNodeIDIndex], nil
}

// GetNamespace from "beehive/pkg/core/model".Model.Router.Resource
func GetNamespace(msg model.Message) (string, error) {
	sli := strings.Split(msg.GetResource(), constants.ResourceSep)
	if len(sli) <= testconstants.ResourceNamespaceIndex {
		return "", fmt.Errorf("Namespace not found")
	}
	return sli[testconstants.ResourceNamespaceIndex], nil
}

// GetResourceType from "beehive/pkg/core/model".Model.Router.Resource
func GetResourceType(msg model.Message) (string, error) {
	sli := strings.Split(msg.GetResource(), constants.ResourceSep)
	if len(sli) <= testconstants.ResourceResourceTypeIndex {
		return "", fmt.Errorf("Resource type not found")
	}
	return sli[testconstants.ResourceResourceTypeIndex], nil
}

// GetResourceName from "beehive/pkg/core/model".Model.Router.Resource
func GetResourceName(msg model.Message) (string, error) {
	sli := strings.Split(msg.GetResource(), constants.ResourceSep)
	if len(sli) <= testconstants.ResourceResourceNameIndex {
		return "", fmt.Errorf("Resource name not found")
	}
	return sli[testconstants.ResourceResourceNameIndex], nil
}

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
