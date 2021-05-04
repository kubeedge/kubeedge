package model

import (

	// Mapping value of json to struct member
	_ "encoding/json"
	"fmt"
	"strings"

	"github.com/kubeedge/beehive/pkg/core/model"
	"github.com/kubeedge/kubeedge/cloud/pkg/common/modules"
	edgemessagelayer "github.com/kubeedge/kubeedge/cloud/pkg/edgecontroller/messagelayer"
)

// constants for resource types
const (
	ResNode   = "node"
	ResMember = "membership"
	ResTwin   = "twin"
	ResAuth   = "auth_info"
	ResDevice = "device"
)

// constants for resource operations
const (
	OpGet        = "get"
	OpResult     = "get_result"
	OpList       = "list"
	OpDetail     = "detail"
	OpDelta      = "delta"
	OpDoc        = "document"
	OpUpdate     = "updated"
	OpInsert     = "insert"
	OpDelete     = "deleted"
	OpConnect    = "connected"
	OpDisConnect = "disconnected"
	OpKeepalive  = "keepalive"
)

// constants for message group
const (
	GpResource = "resource"
)

// constants for message source
const (
	SrcCloudHub         = "cloudhub"
	SrcEdgeController   = "edgecontroller"
	SrcDeviceController = "devicecontroller"
	SrcManager          = "edgemgr"
)

// constants for identifier information for edge hub
const (
	ProjectID = "project_id"
	NodeID    = "node_id"
)

var cloudModuleArray = []string{
	modules.CloudHubModuleName,
	modules.CloudStreamModuleName,
	modules.DeviceControllerModuleName,
	modules.EdgeControllerModuleName,
	modules.SyncControllerModuleName,
}

// HubInfo saves identifier information for edge hub
type HubInfo struct {
	ProjectID string
	NodeID    string
}

// NewResource constructs a resource field using resource type and ID
func NewResource(resType, resID string, info *HubInfo) string {
	var prefix string
	if info != nil {
		prefix = fmt.Sprintf("%s/%s/", model.ResourceTypeNode, info.NodeID)
	}
	if resID == "" {
		return fmt.Sprintf("%s%s", prefix, resType)
	}
	return fmt.Sprintf("%s%s/%s", prefix, resType, resID)
}

// IsNodeStopped indicates if the node is stopped or running
func IsNodeStopped(msg *model.Message) bool {
	resourceType, _ := edgemessagelayer.GetResourceType(*msg)
	if resourceType != model.ResourceTypeNode {
		return false
	}

	if msg.Router.Operation == model.DeleteOperation {
		return true
	}
	return false
}

// IsFromEdge judges if the event is sent from edge
func IsFromEdge(msg *model.Message) bool {
	source := msg.Router.Source
	for _, item := range cloudModuleArray {
		if source == item {
			return false
		}
	}
	return true
}

// IsToEdge judges if the vent should be sent to edge
func IsToEdge(msg *model.Message) bool {
	if msg.Router.Source != SrcManager {
		return true
	}
	resource := msg.Router.Resource
	if strings.HasPrefix(resource, ResNode) {
		tokens := strings.Split(resource, "/")
		if len(tokens) >= 3 {
			resource = strings.Join(tokens[2:], "/")
		}
	}

	// apply special check for edge manager
	resOpMap := map[string][]string{
		ResMember: {OpGet},
		ResTwin:   {OpDelta, OpDoc, OpGet},
		ResAuth:   {OpGet},
		ResNode:   {OpDelete},
	}
	for res, ops := range resOpMap {
		for _, op := range ops {
			if msg.Router.Operation == op && strings.Contains(resource, res) {
				return false
			}
		}
	}
	return true
}

// GetContent dumps the content to string
func GetContent(msg *model.Message) string {
	return fmt.Sprintf("%v", msg.Content)
}
