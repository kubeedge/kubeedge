package messagelayer

import (
	"fmt"
	"strings"

	"github.com/kubeedge/beehive/pkg/common/log"
	"github.com/kubeedge/beehive/pkg/core/model"
	"github.com/kubeedge/kubeedge/edgesite/pkg/controller/constants"
)

// BuildResource return a string as "beehive/pkg/core/model".Message.Router.Resource
func BuildResource(namespace, resourceType, resourceID string) (resource string, err error) {
	if namespace == "" || resourceType == "" {
		err = fmt.Errorf("required parameter are not set (node id, namespace or resource type)")
		return
	}
	resource = fmt.Sprintf("%s%s%s", namespace, constants.ResourceSep, resourceType)
	if resourceID != "" {
		resource += fmt.Sprintf("%s%s", constants.ResourceSep, resourceID)
	}
	return
}

// GetNodeID from "beehive/pkg/core/model".Message.Router.Resource
func GetNodeID(msg model.Message) (string, error) {
	sli := strings.Split(msg.GetResource(), constants.ResourceSep)
	if len(sli) <= constants.ResourceNodeIDIndex {
		return "", fmt.Errorf("node id not found")
	}

	return sli[constants.ResourceNodeIDIndex], nil
}

// GetNamespace from "beehive/pkg/core/model".Model.Router.Resource
func GetNamespace(msg model.Message) (string, error) {
	sli := strings.Split(msg.GetResource(), constants.ResourceSep)
	if len(sli) <= constants.ResourceNamespaceIndex {
		return "", fmt.Errorf("namespace not found")
	}

	log.LOGGER.Infof("The namesapce is %s, %d", sli[constants.ResourceNamespaceIndex], constants.ResourceNamespaceIndex)
	return sli[constants.ResourceNamespaceIndex], nil
}

// GetResourceType from "beehive/pkg/core/model".Model.Router.Resource
func GetResourceType(msg model.Message) (string, error) {
	sli := strings.Split(msg.GetResource(), constants.ResourceSep)

	if len(sli) <= constants.ResourceResourceTypeIndex {
		return "", fmt.Errorf("resource type not found")
	}
	return sli[constants.ResourceResourceTypeIndex], nil
}

// GetResourceName from "beehive/pkg/core/model".Model.Router.Resource
func GetResourceName(msg model.Message) (string, error) {
	sli := strings.Split(msg.GetResource(), constants.ResourceSep)
	if len(sli) <= constants.ResourceResourceNameIndex {
		return "", fmt.Errorf("resource name not found")
	}
	return sli[constants.ResourceResourceNameIndex], nil
}
