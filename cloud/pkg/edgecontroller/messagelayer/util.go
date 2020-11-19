package messagelayer

import (
	"fmt"
	"strings"

	"k8s.io/klog/v2"

	"github.com/kubeedge/beehive/pkg/core/model"
	"github.com/kubeedge/kubeedge/cloud/pkg/edgecontroller/config"
	controller "github.com/kubeedge/kubeedge/cloud/pkg/edgecontroller/constants"
	"github.com/kubeedge/kubeedge/common/constants"
)

// BuildResource return a string as "beehive/pkg/core/model".Message.Router.Resource
func BuildResource(nodeID, namespace, resourceType, resourceID string) (resource string, err error) {
	if namespace == "" || resourceType == "" {
		if !config.Config.EdgeSiteEnable && nodeID == "" {
			err = fmt.Errorf("required parameter are not set (node id, namespace or resource type)")
		} else {
			err = fmt.Errorf("required parameter are not set (namespace or resource type)")
		}
		return
	}

	resource = fmt.Sprintf("%s%s%s%s%s%s%s", controller.ResourceNode, constants.ResourceSep, nodeID, constants.ResourceSep, namespace, constants.ResourceSep, resourceType)
	if config.Config.EdgeSiteEnable {
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
	if config.Config.EdgeSiteEnable {
		length = controller.EdgeSiteResourceNamespaceIndex
	}
	if len(sli) <= length {
		return "", fmt.Errorf("namespace not found")
	}
	var res string
	var index uint8
	if config.Config.EdgeSiteEnable {
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
	if config.Config.EdgeSiteEnable {
		length = controller.EdgeSiteResourceResourceTypeIndex
	}
	if len(sli) <= length {
		return "", fmt.Errorf("resource type not found")
	}

	var res string
	var index uint8
	if config.Config.EdgeSiteEnable {
		res = sli[controller.EdgeSiteResourceResourceTypeIndex]
		index = controller.EdgeSiteResourceResourceTypeIndex
	} else {
		res = sli[controller.ResourceResourceTypeIndex]
		index = controller.ResourceResourceTypeIndex
	}
	klog.V(4).Infof("The resource type is %s, %d", res, index)
	return res, nil
}

// GetResourceName from "beehive/pkg/core/model".Model.Router.Resource
func GetResourceName(msg model.Message) (string, error) {
	sli := strings.Split(msg.GetResource(), constants.ResourceSep)
	length := controller.ResourceResourceNameIndex
	if config.Config.EdgeSiteEnable {
		length = controller.EdgeSiteResourceResourceNameIndex
	}
	if len(sli) <= length {
		return "", fmt.Errorf("resource name not found")
	}

	var res string
	var index uint8
	if config.Config.EdgeSiteEnable {
		res = sli[controller.EdgeSiteResourceResourceNameIndex]
		index = controller.EdgeSiteResourceResourceNameIndex
	} else {
		res = sli[controller.ResourceResourceNameIndex]
		index = controller.ResourceResourceNameIndex
	}
	klog.V(4).Infof("The resource name is %s, %d", res, index)
	return res, nil
}
