package messagelayer

import (
	"github.com/kubeedge/beehive/pkg/core/model"
	"github.com/kubeedge/kubeedge/cloud/pkg/devicecontroller/config"
)

// MessageLayer define all functions that message layer must implement
type MessageLayer interface {
	Send(message model.Message) error
	Receive() (model.Message, error)
	Response(message model.Message) error
}

// NewMessageLayer by config, currently only context
func NewMessageLayer() (MessageLayer, error) {
	switch config.MessageLayer {
	case "context":
		return NewContextMessageLayer()
	default:
		return NewContextMessageLayer()
	}
}
