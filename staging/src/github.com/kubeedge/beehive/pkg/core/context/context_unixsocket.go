package context

import (
	"sync"
	"time"

	"github.com/kubeedge/beehive/pkg/core/model"
)

// UnixSocketContext unixsocket struct
type UnixSocketContext struct {
	filename string
	bufsize  int
	handler  func(string) string
}

var (
	// singleton
	usContext *UnixSocketContext
	usOnce    sync.Once
)

// GetUnixSocketContext defines and returns unix socket context object
func GetUnixSocketContext() *UnixSocketContext {
	usOnce.Do(func() {
		usContext = &UnixSocketContext{}
	})
	return usContext
}

// AddModule adds module to context
func (ctx *UnixSocketContext) AddModule(module string) {

}

// AddModuleGroup adds module to module context group
func (ctx *UnixSocketContext) AddModuleGroup(module, group string) {

}

// Cleanup cleans up module
func (ctx *UnixSocketContext) Cleanup(module string) {

}

// Send async mode
func (ctx *UnixSocketContext) Send(module string, content interface{}) {

}

//Receive the message
//local module name
func (ctx *UnixSocketContext) Receive(module string) interface{} {
	return nil
}

// SendSync sends message in sync mode
// module: the destination of the message
func (ctx *UnixSocketContext) SendSync(module string, message model.Message, timeout time.Duration) (interface{}, error) {
	return nil, nil
}

// SendResp sends response
func (ctx *UnixSocketContext) SendResp(messageID string, content interface{}) {

}

// SendToGroup broadcasts the message to all of group members
func (ctx *UnixSocketContext) SendToGroup(moduleType string, content interface{}) {

}
