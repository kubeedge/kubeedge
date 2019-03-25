package conn

import (
	"github.com/kubeedge/beehive/pkg/common/log"
	"github.com/kubeedge/viaduct/pkg/api"
	"github.com/kubeedge/viaduct/pkg/mux"
)

// connection options
type ConnectionOptions struct {
	// the protocol type that the connection based on
	ConnType string
	// connection or session object for each kind of protocol
	Base interface{}
	// the message route to
	Handler mux.Handler
}

// get connection interface by ConnType
func NewConnection(opts *ConnectionOptions) Connection {
	switch opts.ConnType {
	case api.ProtocolTypeQuic:
		return NewQuicConn(opts)
	case api.ProtocolTypeWS:
		return NewWSConn(opts)
	}
	log.LOGGER.Errorf("bad connection type(%s)", opts.ConnType)
	return nil
}
