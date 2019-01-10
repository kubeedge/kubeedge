package transport

import (
	"net"
	"time"

	"github.com/256dpi/gomqtt/packet"
)

// A Conn is a connection between a client and a broker. It abstracts an
// existing underlying stream connection.
type Conn interface {
	// Send will write the packet to an internal buffer. It will either flush the
	// internal buffer immediately or asynchronously in the background when it gets
	// stale. Encoding errors are directly returned, but any network errors caught
	// while flushing the buffer asynchronously will be returned on the next call.
	//
	// Note: Only one goroutine can Send at the same time.
	Send(pkt packet.Generic, async bool) error

	// Receive will read from the underlying connection and return a fully read
	// packet. It will return an Error if there was an error while decoding or
	// reading from the underlying connection.
	//
	// Note: Only one goroutine can Receive at the same time.
	Receive() (packet.Generic, error)

	// Close will close the underlying connection and cleanup resources. It will
	// return an Error if there was an error while closing the underlying
	// connection.
	Close() error

	// SetReadLimit sets the maximum size of a packet that can be received.
	// If the limit is greater than zero, Receive will close the connection and
	// return an Error if receiving the next packet will exceed the limit.
	SetReadLimit(limit int64)

	// SetReadTimeout sets the maximum time that can pass between reads.
	// If no data is received in the set duration the connection will be closed
	// and Read returns an error.
	SetReadTimeout(timeout time.Duration)

	// LocalAddr will return the underlying connection's local net address.
	LocalAddr() net.Addr

	// RemoteAddr will return the underlying connection's remote net address.
	RemoteAddr() net.Addr
}
