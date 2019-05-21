package messagelayer

import (
	"errors"
	"fmt"
	"strings"

	deviceconstants "github.com/kubeedge/kubeedge/cloud/pkg/devicecontroller/constants"
	constants "github.com/kubeedge/kubeedge/common/constants"
)

// BuildResource return a string as "beehive/pkg/core/model".Message.Router.Resource
func BuildResource(nodeID, resourceType, resourceID string) (resource string, err error) {
	if nodeID == "" || resourceType == "" {
		err = fmt.Errorf("required parameter are not set (node id, namespace or resource type)")
		return
	}
	resource = fmt.Sprintf("%s%s%s%s%s", deviceconstants.ResourceNode, constants.ResourceSep, nodeID, constants.ResourceSep, resourceType)
	if resourceID != "" {
		resource += fmt.Sprintf("%s%s", constants.ResourceSep, resourceID)
	}
	return
}

// GetDeviceID returns the ID of the device
func GetDeviceID(resource string) (string, error) {
	res := strings.Split(resource, "/")
	if len(res) >= deviceconstants.ResourceDeviceIDIndex+1 && res[deviceconstants.ResourceDeviceIndex] == deviceconstants.ResourceDevice {
		return res[deviceconstants.ResourceDeviceIDIndex], nil
	}
	return "", errors.New("failed to get device id")
}

// GetResourceType returns the resourceType of message received from edge
func GetResourceType(resource string) (string, error) {
	if strings.Contains(resource, deviceconstants.ResourceTypeTwinEdgeUpdated) {
		return deviceconstants.ResourceTypeTwinEdgeUpdated, nil
	}
	return "", errors.New("unknown resource")
}
