package conn

import (
	"io"

	"k8s.io/klog"

	"github.com/kubeedge/viaduct/pkg/api"
	"github.com/kubeedge/viaduct/pkg/mux"
)

// connection options
type ConnectionOptions struct {
	// the protocol type that the connection based on
	ConnType string
	// connection or session object for each kind of protocol
	Base interface{}
	// control lane
	CtrlLane interface{}
	// connect stat
	State *ConnectionState
	// the message route to
	Handler mux.Handler
	// package type
	// only used by websocket mode
	ConnUse api.UseType
	// consumer for raw data
	Consumer io.Writer
	// auto route into entries
	AutoRoute bool
}

// get connection interface by ConnTye
func NewConnection(opts *ConnectionOptions) Connection {
	switch opts.ConnType {
	case api.ProtocolTypeQuic:
		return NewQuicConn(opts)
	case api.ProtocolTypeWS:
		return NewWSConn(opts)
	}
	klog.Errorf("bad connection type(%s)", opts.ConnType)
	return nil
}
