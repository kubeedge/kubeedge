package client

import (
	"crypto/tls"
	"fmt"
	"io"
	"sync"
	"time"

	"k8s.io/klog"

	"github.com/kubeedge/viaduct/pkg/api"
	"github.com/kubeedge/viaduct/pkg/conn"
	"github.com/kubeedge/viaduct/pkg/mux"
)

// protocol client
// each protocol(websocket/quic) provide Connect
type ProtocolClient interface {
	Connect() (conn.Connection, error)
}

// the common options of client
type Options struct {
	// protocol type
	Type string
	// the addr or url
	Addr string
	// client type
	ConnUse api.UseType
	// used to configure a TLS client
	TLSConfig *tls.Config
	// the message will route to Handler automatically if AutoRoute is true
	Handler mux.Handler
	// auto route flag
	AutoRoute bool
	// HandshakeTimeout is the maximum duration that the cryptographic handshake may take.
	HandshakeTimeout time.Duration
	// consumer for raw data
	Consumer io.Writer
}

// client including common options and extend options
type Client struct {
	// protocol connection
	protoConn conn.Connection
	// protocol connection look
	connLock sync.Mutex
	// common options
	Options
	// client type
	ConnUse api.UseType
	// consumer for raw data
	Consumer io.Writer
	// extend options
	ExOpts interface{}
}

// Connect try to connect remote peer
func (c *Client) Connect() (conn.Connection, error) {
	var protoClient ProtocolClient

	// get protocol client instance by type
	switch c.Type {
	case api.ProtocolTypeQuic:
		protoClient = NewQuicClient(c.Options, c.ExOpts)
	case api.ProtocolTypeWS:
		protoClient = NewWSClient(c.Options, c.ExOpts)
	default:
		klog.Errorf("bad protocol type(%v)", c.Type)
		return nil, fmt.Errorf("bad protocol type(%v)", c.Type)
	}

	// try to connect to protocol server
	c.connLock.Lock()
	protoConn, err := protoClient.Connect()
	if err != nil {
		c.connLock.Unlock()
		return protoConn, err
	}
	c.protoConn = protoConn
	c.connLock.Unlock()

	// check and route to handler
	go protoConn.ServeConn()

	return protoConn, nil
}

// close the connection
func (c *Client) Close() error {
	c.connLock.Lock()
	defer c.connLock.Unlock()

	if c.protoConn != nil {
		return c.protoConn.Close()
	}
	return nil
}
