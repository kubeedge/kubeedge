package model

import (
	// Mapping value of json to struct member
	_ "encoding/json"
	"fmt"
	"strings"

	"k8s.io/klog"

	"github.com/kubeedge/beehive/pkg/core/model"
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

const (
	StopGetFromQueue = "stopgetfromqueue"
)

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
	tokens := strings.Split(msg.Router.Resource, "/")
	if len(tokens) != 2 || tokens[0] != ResNode {
		return false
	}
	if msg.Router.Operation == OpDelete {
		return true
	}
	if msg.Router.Operation != OpUpdate || msg.Content == nil {
		return false
	}
	body, ok := msg.Content.(map[string]interface{})
	if !ok {
		klog.Errorf("fail to decode node update message: %s, type is %T", msg.GetContent(), msg.Content)
		// it can't be determined if the node has stopped
		return false
	}
	// trust struct of json body
	action, ok := body["action"]
	if !ok || action.(string) != "stop" {
		return false
	}
	return true
}

// IsFromEdge judges if the event is sent from edge
func IsFromEdge(msg *model.Message) bool {
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
