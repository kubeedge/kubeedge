package messagelayer

import (
	"fmt"
	"strings"

	"k8s.io/klog/v2"

	"github.com/kubeedge/beehive/pkg/core/model"
	controller "github.com/kubeedge/kubeedge/cloud/pkg/edgecontroller/constants"
	"github.com/kubeedge/kubeedge/common/constants"
)

// BuildResource return a string as "beehive/pkg/core/model".Message.Router.Resource
func BuildResource(nodeID, namespace, resourceType, resourceID string) (resource string, err error) {
	if namespace == "" || resourceType == "" || nodeID == "" {
		err = fmt.Errorf("required parameter are not set (node id, namespace or resource type)")
		return
	}

	resource = fmt.Sprintf("%s%s%s%s%s%s%s", controller.ResourceNode, constants.ResourceSep, nodeID, constants.ResourceSep, namespace, constants.ResourceSep, resourceType)
	if resourceID != "" {
		resource += fmt.Sprintf("%s%s", constants.ResourceSep, resourceID)
	}
	return
}

// BuildResourceForRouter return a string as "beehive/pkg/core/model".Message.Router.Resource
func BuildResourceForRouter(resourceType, resourceID string) (resource string, err error) {
	if resourceID == "" || resourceType == "" {
		err = fmt.Errorf("required parameter are not set (resourceID or resource type)")
		return
	}
	resource = fmt.Sprintf("%s%s%s", constants.ResourceTypeRule, constants.ResourceSep, resourceID)
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

	res := sli[controller.ResourceNamespaceIndex]
	index := controller.ResourceNamespaceIndex

	klog.V(4).Infof("The namespace is %s, %d", res, index)
	return res, nil
}

// GetResourceType from "beehive/pkg/core/model".Model.Router.Resource
func GetResourceType(msg model.Message) (string, error) {
	sli := strings.Split(msg.GetResource(), constants.ResourceSep)
	if len(sli) <= controller.ResourceResourceTypeIndex {
		return "", fmt.Errorf("resource type not found")
	}

	res := sli[controller.ResourceResourceTypeIndex]
	index := controller.ResourceResourceTypeIndex

	klog.V(4).Infof("The resource type is %s, %d", res, index)
	return res, nil
}

// GetResourceName from "beehive/pkg/core/model".Model.Router.Resource
func GetResourceName(msg model.Message) (string, error) {
	sli := strings.Split(msg.GetResource(), constants.ResourceSep)

	if len(sli) <= controller.ResourceResourceNameIndex {
		return "", fmt.Errorf("resource name not found")
	}

	res := sli[controller.ResourceResourceNameIndex]
	index := controller.ResourceResourceNameIndex

	klog.V(4).Infof("The resource name is %s, %d", res, index)
	return res, nil
}
