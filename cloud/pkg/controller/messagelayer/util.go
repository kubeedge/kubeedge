package messagelayer

import (
	"fmt"
	"strings"

	"github.com/kubeedge/beehive/pkg/core/model"
	controller "github.com/kubeedge/kubeedge/cloud/pkg/controller/constants"
	"github.com/kubeedge/kubeedge/common/constants"
)

// BuildResource return a string as "beehive/pkg/core/model".Message.Router.Resource
func BuildResource(nodeID, namespace, resourceType, resourceID string) (resource string, err error) {
	if nodeID == "" || namespace == "" || resourceType == "" {
		err = fmt.Errorf("required parameter are not set (node id, namespace or resource type)")
		return
	}
	resource = fmt.Sprintf("%s%s%s%s%s%s%s", controller.ResourceNode, constants.ResourceSep, nodeID, constants.ResourceSep, namespace, constants.ResourceSep, resourceType)
	if resourceID != "" {
		resource += fmt.Sprintf("%s%s", constants.ResourceSep, resourceID)
	}
	return
}

// GetNodeID from "beehive/pkg/core/model".Message.Router.Resource
func GetNodeID(msg model.Message) (string, error) {
	sli := strings.Split(msg.GetResource(), constants.ResourceSep)
	if len(sli) <= controller.ResourceNodeIDIndex {
		return "", fmt.Errorf("node id not found")
	}
	return sli[controller.ResourceNodeIDIndex], nil
}

// GetNamespace from "beehive/pkg/core/model".Model.Router.Resource
func GetNamespace(msg model.Message) (string, error) {
	sli := strings.Split(msg.GetResource(), constants.ResourceSep)
	if len(sli) <= controller.ResourceNamespaceIndex {
		return "", fmt.Errorf("namespace not found")
	}
	return sli[controller.ResourceNamespaceIndex], nil
}

// GetResourceType from "beehive/pkg/core/model".Model.Router.Resource
func GetResourceType(msg model.Message) (string, error) {
	sli := strings.Split(msg.GetResource(), constants.ResourceSep)
	if len(sli) <= controller.ResourceResourceTypeIndex {
		return "", fmt.Errorf("resource type not found")
	}
	return sli[controller.ResourceResourceTypeIndex], nil
}

// GetResourceName from "beehive/pkg/core/model".Model.Router.Resource
func GetResourceName(msg model.Message) (string, error) {
	sli := strings.Split(msg.GetResource(), constants.ResourceSep)
	if len(sli) <= controller.ResourceResourceNameIndex {
		return "", fmt.Errorf("resource name not found")
	}
	return sli[controller.ResourceResourceNameIndex], nil
}
