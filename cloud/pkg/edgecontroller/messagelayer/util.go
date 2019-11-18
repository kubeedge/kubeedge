package messagelayer

import (
	"fmt"
	"strings"

	"k8s.io/klog"

	"github.com/kubeedge/beehive/pkg/core/model"
	"github.com/kubeedge/kubeedge/cloud/pkg/edgecontroller/config"
	controller "github.com/kubeedge/kubeedge/cloud/pkg/edgecontroller/constants"
	"github.com/kubeedge/kubeedge/common/constants"
)

// BuildResource return a string as "beehive/pkg/core/model".Message.Router.Resource
func BuildResource(nodeID, namespace, resourceType, resourceID string) (resource string, err error) {
	if namespace == "" || resourceType == "" {
		if !config.Get().EdgeSiteEnabled && nodeID == "" {
			err = fmt.Errorf("required parameter are not set (node id, namespace or resource type)")
		} else {
			err = fmt.Errorf("required parameter are not set (namespace or resource type)")
		}
		return
	}

	resource = fmt.Sprintf("%s%s%s%s%s%s%s", controller.ResourceNode, constants.ResourceSep, nodeID, constants.ResourceSep, namespace, constants.ResourceSep, resourceType)
	if config.Get().EdgeSiteEnabled {
		resource = fmt.Sprintf("%s%s%s", namespace, constants.ResourceSep, resourceType)
	}
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
	length := controller.ResourceNamespaceIndex
	if config.Get().EdgeSiteEnabled {
		length = controller.EdgeSiteResourceNamespaceIndex
	}
	if len(sli) <= length {
		return "", fmt.Errorf("namespace not found")
	}
	var res string
	var index uint8
	if config.Get().EdgeSiteEnabled {
		res = sli[controller.EdgeSiteResourceNamespaceIndex]
		index = controller.EdgeSiteResourceNamespaceIndex
	} else {
		res = sli[controller.ResourceNamespaceIndex]
		index = controller.ResourceNamespaceIndex
	}
	klog.V(4).Infof("The namespace is %s, %d", res, index)
	return res, nil
}

// GetResourceType from "beehive/pkg/core/model".Model.Router.Resource
func GetResourceType(msg model.Message) (string, error) {
	sli := strings.Split(msg.GetResource(), constants.ResourceSep)
	length := controller.ResourceResourceTypeIndex
	if config.Get().EdgeSiteEnabled {
		length = controller.EdgeSiteResourceResourceTypeIndex
	}
	if len(sli) <= length {
		return "", fmt.Errorf("resource type not found")
	}

	var res string
	var index uint8
	if config.Get().EdgeSiteEnabled {
		res = sli[controller.EdgeSiteResourceResourceTypeIndex]
		index = controller.EdgeSiteResourceResourceTypeIndex
	} else {
		res = sli[controller.ResourceResourceTypeIndex]
		index = controller.ResourceResourceTypeIndex
	}
	klog.Infof("The resource type is %s, %d", res, index)

	return res, nil
}

// GetResourceName from "beehive/pkg/core/model".Model.Router.Resource
func GetResourceName(msg model.Message) (string, error) {
	sli := strings.Split(msg.GetResource(), constants.ResourceSep)
	length := controller.ResourceResourceNameIndex
	if config.Get().EdgeSiteEnabled {
		length = controller.EdgeSiteResourceResourceNameIndex
	}
	if len(sli) <= length {
		return "", fmt.Errorf("resource name not found")
	}

	var res string
	var index uint8
	if config.Get().EdgeSiteEnabled {
		res = sli[controller.EdgeSiteResourceResourceNameIndex]
		index = controller.EdgeSiteResourceResourceNameIndex
	} else {
		res = sli[controller.ResourceResourceNameIndex]
		index = controller.ResourceResourceNameIndex
	}
	klog.Infof("The resource name is %s, %d", res, index)
	return res, nil
}
