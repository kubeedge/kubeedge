package conn

import (
	"crypto/x509"
	"net"
	"net/http"
	"time"

	"github.com/kubeedge/beehive/pkg/core/model"
)

// connection states
// TODO: add connection state filed
type ConnectionState struct {
	State            string
	Headers          http.Header
	PeerCertificates []*x509.Certificate
}

// the operation set of connection
type Connection interface {
	// process message from the connection
	ServeConn()

	// SetReadDeadline sets the deadline for future Read calls
	// and any currently-blocked Read call.
	// A zero value for t means Read will not time out.
	SetReadDeadline(t time.Time) error

	// SetWriteDeadline sets the deadline for future Write calls
	// and any currently-blocked Write call.
	// Even if write times out, it may return n > 0, indicating that
	// some of the data was successfully written.
	// A zero value for t means Write will not time out.
	SetWriteDeadline(t time.Time) error

	// Read read raw data from the connection
	// you can also set raw data consumer when new client/server instance
	Read(raw []byte) (int, error)

	// Write write raw data to the connection
	// it will open a stream for raw data
	Write(raw []byte) (int, error)

	// WriteMessageAsync writes data to the connection and don't care about the response.
	WriteMessageAsync(msg *model.Message) error

	// WriteMessageSync writes data to the connection and care about the response.
	WriteMessageSync(msg *model.Message) (*model.Message, error)

	// ReadMessage reads message from the connection.
	// it will be blocked when no message received
	// if you want to use this api for message reading,
	// make sure AutoRoute be false
	ReadMessage(msg *model.Message) error

	// RemoteAddr returns the remote network address.
	RemoteAddr() net.Addr

	// LocalAddr returns the local network address.
	LocalAddr() net.Addr

	// ConnectState return the current connection state
	ConnectionState() ConnectionState

	// Close closes the connection.
	// Any blocked Read or Write operations will be unblocked and return errors.
	Close() error
}
