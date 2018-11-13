package context

import (
	"time"

	"github.com/kubeedge/kubeedge/beehive/pkg/core/model"
)

type ModuleContext interface {
	AddModule(module string)
	AddModuleGroup(module, group string)
	Cleanup(module string)
}

type MessageContext interface {
	// async mode
	Send(module string, message model.Message)
	Receive(module string) (model.Message, error)
	// sync mode
	SendSync(module string, message model.Message, timeout time.Duration) (model.Message, error)
	SendResp(message model.Message)
	// group broadcast
	Send2Group(moduleType string, message model.Message)
	Send2GroupSync(moduleType string, message model.Message, timeout time.Duration) error
}

// global context
type Context struct {
	moduleContext  ModuleContext
	messageContext MessageContext
}
