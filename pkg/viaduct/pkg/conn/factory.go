package conn

import (
	"io"
	"time"

	"k8s.io/klog/v2"

	"github.com/kubeedge/kubeedge/pkg/viaduct/pkg/api"
	"github.com/kubeedge/kubeedge/pkg/viaduct/pkg/mux"
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
	// OnReadTransportErr
	OnReadTransportErr func(nodeID, projectID string)
	// ReadDeadline is the idle timeout for reading from the connection.
	// When it is greater than zero, the websocket connection keeps the read
	// deadline refreshed through ping/pong so a stalled (half-open) connection
	// is detected within the deadline instead of waiting for the kernel TCP
	// timeout. It is only honored by the websocket message connection.
	ReadDeadline time.Duration
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
