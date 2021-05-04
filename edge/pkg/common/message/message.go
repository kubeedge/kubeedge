package message

import (
	"github.com/kubeedge/beehive/pkg/core/model"
)

//constant defining node connection types
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

	ResourceGroupName = "resource"
	TwinGroupName     = "twin"
	FuncGroupName     = "func"
	UserGroupName     = "user"
)

//BuildMsg returns message object with router and content details
func BuildMsg(group, parentID, sourceName, resource, operation string, content interface{}) *model.Message {
	msg := model.NewMessage(parentID).BuildRouter(sourceName, group, resource, operation).FillBody(content)
	return msg
}
