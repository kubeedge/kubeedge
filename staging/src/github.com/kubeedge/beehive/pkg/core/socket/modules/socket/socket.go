package socket

import (
	"net"
)

// SocketServer module socket server
type SocketServer struct {
	enable     bool
	name       string
	address    string
	buffSize   uint64
	socketType string
	connMax    int
	listener   net.Listener
	pipeKeeper chan struct{}
	stopChan   chan struct{}
}

// Name name
func (m *SocketServer) Name() string {
	return m.name
}

// Group group
func (m *SocketServer) Group() string {
	return m.name
}

// Start start
func (m *SocketServer) Start() {
	m.startServer()
}

// Enable enable
func (m *SocketServer) Enable() bool {
	return m.enable
}
