package client

import (
	"github.com/kubeedge/beehive/pkg/core/model"
)

//BuildMsg returns message object with router and content details
func BuildMsg(group, parentID, sourceName, resource, operation string, content interface{}) *model.Message {
	msg := model.NewMessage(parentID).BuildRouter(sourceName, group, resource, operation).FillBody(content)
	return msg
}
