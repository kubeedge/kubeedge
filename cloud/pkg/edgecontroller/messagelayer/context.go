package messagelayer

import (
	"strings"

	beehiveContext "github.com/kubeedge/beehive/pkg/core/context"
	"github.com/kubeedge/beehive/pkg/core/model"
	"github.com/kubeedge/kubeedge/cloud/pkg/edgecontroller/config"
	"github.com/kubeedge/kubeedge/common/constants"
)

// MessageLayer define all functions that message layer must implement
type MessageLayer interface {
	Send(message model.Message) error
	Receive() (model.Message, error)
	Response(message model.Message) error
}

// ContextMessageLayer build on context
type ContextMessageLayer struct {
	SendModuleName     string
	SendRouterModuleName     string
	ReceiveModuleName  string
	ResponseModuleName string
}

// Send message
func (cml *ContextMessageLayer) Send(message model.Message) error {
	// if message is rule/ruleendpoint type, send to router module.
	if isRouterMsg(message) {
		beehiveContext.Send(cml.SendRouterModuleName, message)
		return nil
	}
	beehiveContext.Send(cml.SendModuleName, message)
	return nil
}

func isRouterMsg(message model.Message) bool {
	resourceArray := strings.Split(message.GetResource(), constants.ResourceSep)
	if len(resourceArray) == 2 && (resourceArray[0] == constants.ResourceTypeRule || resourceArray[0] == constants.ResourceTypeRuleEndpoint) {
		return true
	}
	return false
}

// Receive message
func (cml *ContextMessageLayer) Receive() (model.Message, error) {
	return beehiveContext.Receive(cml.ReceiveModuleName)
}

// Response message
func (cml *ContextMessageLayer) Response(message model.Message) error {
	beehiveContext.Send(cml.ResponseModuleName, message)
	return nil
}

// NewContextMessageLayer create a ContextMessageLayer
func NewContextMessageLayer() MessageLayer {
	return &ContextMessageLayer{
		SendModuleName:     string(config.Config.Context.SendModule),
		SendRouterModuleName: string(config.Config.Context.SendRouterModule),
		ReceiveModuleName:  string(config.Config.Context.ReceiveModule),
		ResponseModuleName: string(config.Config.Context.ResponseModule),
	}
}
