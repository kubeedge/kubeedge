package message

import (
	"github.com/kubeedge/kubeedge/beehive/pkg/core/model"
)

const (
	ResourceTypeNodeConnection = "node/connection"
	OperationNodeConnection    = "publish"
	SourceNodeConnection       = "edgehub"
)

func BuildMsg(group, parentID, sourceName, resource, operation string, content interface{}) *model.Message {
	msg := model.NewMessage(parentID).BuildRouter(sourceName, group, resource, operation).FillBody(content)
	return msg
}
