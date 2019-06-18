package model

import (
	// Mapping value of json to struct member
	_ "encoding/json"
	"fmt"
	"strings"

	"github.com/kubeedge/beehive/pkg/common/log"
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
	SrcCloudHub   = "cloudhub"
	SrcController = "controller"
	SrcManager    = "edgemgr"
)

// HubInfo saves identifier information for edge hub
type HubInfo struct {
	ProjectID string
	NodeID    string
}

// UserGroupInfo struct
type UserGroupInfo struct {
	Resource  string `json:"resource"`
	Operation string `json:"operation"`
}

// Event represents message communicated between cloud hub and edge hub
type Event struct {
	Group     string        `json:"msg_group"`
	Source    string        `json:"source"`
	UserGroup UserGroupInfo `json:"user_group"`
	ID        string        `json:"msg_id"`
	ParentID  string        `json:"parent_msg_id"`
	Timestamp int64         `json:"timestamp"`
	Content   interface{}   `json:"content"`
}

// EventToMessage converts an event to a model message
func EventToMessage(event *Event) model.Message {
	var msg model.Message
	msg.BuildHeader(event.ID, event.ParentID, event.Timestamp)
	msg.BuildRouter(event.Source, event.Group, event.UserGroup.Resource, event.UserGroup.Operation)
	msg.FillBody(event.Content)
	return msg
}

// MessageToEvent converts a model message to an event
func MessageToEvent(msg *model.Message) Event {
	var event Event
	event.ID = msg.GetID()
	event.ParentID = msg.GetParentID()
	event.Timestamp = msg.GetTimestamp()
	event.Source = msg.GetSource()
	event.Group = msg.GetGroup()
	event.Content = msg.GetContent()
	event.UserGroup = UserGroupInfo{
		Resource:  msg.GetResource(),
		Operation: msg.GetOperation(),
	}
	return event
}

// NewResource constructs a resource field using resource type and ID
func NewResource(resType, resID string, info *HubInfo) string {
	var prefix string
	if info != nil {
		prefix = fmt.Sprintf("%s/%s/", "node", info.NodeID)
	}
	if resID == "" {
		return fmt.Sprintf("%s%s", prefix, resType)
	}
	return fmt.Sprintf("%s%s/%s", prefix, resType, resID)
}

// IsNodeStopped indicates if the node is stopped or running
func (event *Event) IsNodeStopped() bool {
	tokens := strings.Split(event.UserGroup.Resource, "/")
	if len(tokens) != 2 || tokens[0] != ResNode {
		return false
	}
	if event.UserGroup.Operation == OpDelete {
		return true
	}
	if event.UserGroup.Operation != OpUpdate || event.Content == nil {
		return false
	}
	body, ok := event.Content.(map[string]interface{})
	if !ok {
		log.LOGGER.Errorf("fail to decode node update message: %s, type is %T", event.GetContent(), event.Content)
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
func (event *Event) IsFromEdge() bool {
	return true
}

// IsToEdge judges if the vent should be sent to edge
func (event *Event) IsToEdge() bool {
	if event.Source != SrcManager {
		return true
	}
	resource := event.UserGroup.Resource
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
			if event.UserGroup.Operation == op && strings.Contains(resource, res) {
				return false
			}
		}
	}
	return true
}

// GetContent dumps the content to string
func (event *Event) GetContent() string {
	return fmt.Sprintf("%v", event.Content)
}
