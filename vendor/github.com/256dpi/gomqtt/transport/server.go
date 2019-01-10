package transport

import "net"

// A Server is a local port on which incoming connections can be accepted.
type Server interface {
	// Accept will return the next available connection or block until a
	// connection becomes available, otherwise returns an Error.
	Accept() (Conn, error)

	// Close will close the underlying listener and cleanup resources. It will
	// return an Error if the underlying listener didn't close cleanly.
	Close() error

	// Addr returns the server's network address.
	Addr() net.Addr
}
