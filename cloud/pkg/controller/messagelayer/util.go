package messagelayer

import (
	"fmt"
	"strings"

	"github.com/kubeedge/beehive/pkg/common/log"
	"github.com/kubeedge/beehive/pkg/core/model"
	"github.com/kubeedge/kubeedge/cloud/pkg/controller/config"
	controller "github.com/kubeedge/kubeedge/cloud/pkg/controller/constants"
	"github.com/kubeedge/kubeedge/common/constants"
)

// BuildResource return a string as "beehive/pkg/core/model".Message.Router.Resource
func BuildResource(nodeID, namespace, resourceType, resourceID string) (resource string, err error) {
	if namespace == "" || resourceType == "" {
		if !config.EdgeSiteEnabled && nodeID == "" {
			err = fmt.Errorf("required parameter are not set (node id, namespace or resource type)")
		} else {
			err = fmt.Errorf("required parameter are not set (namespace or resource type)")
		}
		return
	}

	resource = fmt.Sprintf("%s%s%s%s%s%s%s", controller.ResourceNode, constants.ResourceSep, nodeID, constants.ResourceSep, namespace, constants.ResourceSep, resourceType)
	if config.EdgeSiteEnabled {
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
	if config.EdgeSiteEnabled {
		length = controller.EdgeSiteResourceNamespaceIndex
	}
	if len(sli) <= length {
		return "", fmt.Errorf("namespace not found")
	}
	var res string
	var index uint8
	if config.EdgeSiteEnabled {
		res = sli[controller.EdgeSiteResourceNamespaceIndex]
		index = controller.EdgeSiteResourceNamespaceIndex
	} else {
		res = sli[controller.ResourceNamespaceIndex]
		index = controller.ResourceNamespaceIndex
	}
	log.LOGGER.Debugf("The namespace is %s, %d", res, index)
	return res, nil
}

// GetResourceType from "beehive/pkg/core/model".Model.Router.Resource
func GetResourceType(msg model.Message) (string, error) {
	sli := strings.Split(msg.GetResource(), constants.ResourceSep)
	length := controller.ResourceResourceTypeIndex
	if config.EdgeSiteEnabled {
		length = controller.EdgeSiteResourceResourceTypeIndex
	}
	if len(sli) <= length {
		return "", fmt.Errorf("resource type not found")
	}

	var res string
	var index uint8
	if config.EdgeSiteEnabled {
		res = sli[controller.EdgeSiteResourceResourceTypeIndex]
		index = controller.EdgeSiteResourceResourceTypeIndex
	} else {
		res = sli[controller.ResourceResourceTypeIndex]
		index = controller.ResourceResourceTypeIndex
	}
	log.LOGGER.Infof("The resource type is %s, %d", res, index)

	return res, nil
}

// GetResourceName from "beehive/pkg/core/model".Model.Router.Resource
func GetResourceName(msg model.Message) (string, error) {
	sli := strings.Split(msg.GetResource(), constants.ResourceSep)
	length := controller.ResourceResourceNameIndex
	if config.EdgeSiteEnabled {
		length = controller.EdgeSiteResourceResourceNameIndex
	}
	if len(sli) <= length {
		return "", fmt.Errorf("resource name not found")
	}

	var res string
	var index uint8
	if config.EdgeSiteEnabled {
		res = sli[controller.EdgeSiteResourceResourceNameIndex]
		index = controller.EdgeSiteResourceResourceNameIndex
	} else {
		res = sli[controller.ResourceResourceNameIndex]
		index = controller.ResourceResourceNameIndex
	}
	log.LOGGER.Infof("The resource name is %s, %d", res, index)
	return res, nil
}
