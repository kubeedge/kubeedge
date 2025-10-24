package model

import (
	_ "encoding/json" // Mapping value of json to struct member
	"fmt"
	"strings"

	"github.com/kubeedge/beehive/pkg/core/model"
	"github.com/kubeedge/kubeedge/cloud/pkg/common/messagelayer"
	"github.com/kubeedge/kubeedge/cloud/pkg/common/modules"
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

// GpResource constants for message group
const (
	GpResource = "resource"
)

// constants for message source
const (
	SrcManager = "edgemgr"
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

// HubInfo stores identifier information for an edge hub.
type HubInfo struct {
	ProjectID string
	NodeID    string
}

// NewResource constructs a full resource string using resource type and ID.
// If HubInfo is provided, the resource path will be prefixed with the node information.
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

// IsNodeStopped returns true if the message indicates that a node has been stopped (deleted).
func IsNodeStopped(msg *model.Message) bool {
	resourceType, _ := messagelayer.GetResourceType(*msg)
	return resourceType == model.ResourceTypeNode && msg.Router.Operation == model.DeleteOperation
}

// IsFromEdge returns true if the message was sent from an edge component.
func IsFromEdge(msg *model.Message) bool {
	source := msg.Router.Source
	for _, item := range cloudModuleArray {
		if source == item {
			return false
		}
	}
	return true
}

// IsToEdge returns true if the message should be delivered to edge nodes.
// Some messages from the edge manager are internal and should not be sent to the edge.
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

	// apply special check for edge manager messages
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
