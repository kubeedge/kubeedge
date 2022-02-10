package socket

import (
	"net"
)

// Server module socket server
type Server struct {
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
func (m *Server) Name() string {
	return m.name
}

// Group group
func (m *Server) Group() string {
	return m.name
}

// Start start
func (m *Server) Start() {
	m.startServer()
}

// Enable enable
func (m *Server) Enable() bool {
	return m.enable
}
