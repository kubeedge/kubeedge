package messagelayer

import (
	"github.com/kubeedge/kubeedge/cloud/edgecontroller/pkg/controller/config"
	"github.com/kubeedge/kubeedge/common/beehive/pkg/core/context"
	"github.com/kubeedge/kubeedge/common/beehive/pkg/core/model"
)

// ContextMessageLayer build on context
type ContextMessageLayer struct {
	SendModuleName     string
	ReceiveModuleName  string
	ResponseModuleName string
	Context            *context.Context
}

// Send message
func (cml *ContextMessageLayer) Send(message model.Message) error {
	cml.Context.Send(cml.SendModuleName, message)
	return nil
}

// Receive message
func (cml *ContextMessageLayer) Receive() (model.Message, error) {
	return cml.Context.Receive(cml.ReceiveModuleName)
}

// Response message
func (cml *ContextMessageLayer) Response(message model.Message) error {
	cml.Context.Send(cml.ResponseModuleName, message)
	return nil
}

// NewContextMessageLayer create a ContextMessageLayer
func NewContextMessageLayer() (*ContextMessageLayer, error) {
	return &ContextMessageLayer{SendModuleName: config.ContextSendModule, ReceiveModuleName: config.ContextReceiveModule, ResponseModuleName: config.ContextResponseModule, Context: config.Context}, nil
}
