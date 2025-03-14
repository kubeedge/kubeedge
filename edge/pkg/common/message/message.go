package message

import (
	"fmt"
	"strings"

	"github.com/kubeedge/beehive/pkg/core/model"
)

// constant defining node connection types
const (
	ResourceTypeNodeConnection = "node/connection"
	SourceNodeConnection       = "edgehub"
	OperationNodeConnection    = "connect"
	OperationSubscribe         = "subscribe"
	OperationUnsubscribe       = "unsubscribe"
	OperationMessage           = "message"
	OperationPublish           = "publish"
	OperationGetResult         = "get_result"
	OperationResponse          = "response"
	OperationKeepalive         = "keepalive"
	OperationStart             = "start"
	OperationStop              = "stop"

	ResourceGroupName = "resource"
	FuncGroupName     = "func"
	UserGroupName     = "user"
)

// BuildMsg returns message object with router and content details
func BuildMsg(group, parentID, sourceName, resource, operation string, content interface{}) *model.Message {
	msg := model.NewMessage(parentID).BuildRouter(sourceName, group, resource, operation).FillBody(content)
	return msg
}

// ParseResourceEdge parses resource at edge and returns namespace, resource_type, resource_id.
// If operation of msg is query list, return namespace, pod.
func ParseResourceEdge(resource string, operation string) (string, string, string, error) {
	resourceSplits := strings.Split(resource, "/")
	if len(resourceSplits) == 3 {
		return resourceSplits[0], resourceSplits[1], resourceSplits[2], nil
	} else if operation == model.QueryOperation || operation == model.ResponseOperation && len(resourceSplits) == 2 {
		return resourceSplits[0], resourceSplits[1], "", nil
	}
	return "", "", "", fmt.Errorf("resource: %s format incorrect, or Operation: %s is not query/response", resource, operation)
}
