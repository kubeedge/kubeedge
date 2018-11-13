package context

import (
	"sync"
	"time"

	"github.com/kubeedge/kubeedge/beehive/pkg/core/model"
)

// UnixSocket struct
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

func GetUnixSocketContext() *UnixSocketContext {
	usOnce.Do(func() {
		usContext = &UnixSocketContext{}
	})
	return usContext
}

func (ctx *UnixSocketContext) AddModule(module string) {

}

func (ctx *UnixSocketContext) AddModuleGroup(module, group string) {

}

func (ctx *UnixSocketContext) Cleanup(module string) {

}

// async mode
func (ctx *UnixSocketContext) Send(module string, content interface{}) {

}

func (ctx *UnixSocketContext) Receive(module string) interface{} {
	return nil
}

// sync mode
func (ctx *UnixSocketContext) SendSync(module string, message model.Message, timeout time.Duration) (interface{}, error) {
	return nil, nil
}

func (ctx *UnixSocketContext) SendResp(messageId string, content interface{}) {

}

// group broadcast
func (ctx *UnixSocketContext) Send2Group(moduleType string, content interface{}) {

}
