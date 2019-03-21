package message

import (
	"github.com/kubeedge/beehive/pkg/core/model"
)

//constant defining node connection types
const (
	ResourceTypeNodeConnection = "node/connection"
	OperationNodeConnection    = "publish"
	SourceNodeConnection       = "edgehub"
)

//BuildMsg returns message object with router and content details
func BuildMsg(group, parentID, sourceName, resource, operation string, content interface{}) *model.Message {
	msg := model.NewMessage(parentID).BuildRouter(sourceName, group, resource, operation).FillBody(content)
	return msg
}
