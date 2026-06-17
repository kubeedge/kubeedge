package store

import (
	"net"

	"k8s.io/klog/v2"

	"github.com/kubeedge/beehive/pkg/core/model"
	"github.com/kubeedge/beehive/pkg/core/socket/wrapper"
)

// PipeInfo pipe info
type PipeInfo struct {
	pipe interface{}
}

// Channel channel
func (info *PipeInfo) Channel() chan model.Message {
	if ch, ok := info.pipe.(chan model.Message); ok {
		return ch
	}
	klog.Warning("failed to get channel")
	return nil
}

// Socket socket
func (info *PipeInfo) Socket() net.Conn {
	if socket, ok := info.pipe.(net.Conn); ok {
		return socket
	}
	klog.Warning("failed to get socket")
	return nil
}

// Wrapper wrapper
func (info *PipeInfo) Wrapper() wrapper.Conn {
	if socket, ok := info.pipe.(wrapper.Conn); ok {
		return socket
	}
	klog.Warning("failed to get conn wrapper")
	return nil
}
