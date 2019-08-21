package app

import (
	"net"

	"k8s.io/klog"
)

const (
	// DefaultBufferSize represents default buffer size
	DefaultBufferSize = 10480
)

// UnixDomainSocket struct
type UnixDomainSocket struct {
	filename   string
	buffersize int
}

// NewUnixDomainSocket create new socket
func NewUnixDomainSocket(filename string, buffersize ...int) *UnixDomainSocket {
	size := DefaultBufferSize
	if buffersize != nil {
		size = buffersize[0]
	}
	us := UnixDomainSocket{filename: filename, buffersize: size}
	return &us
}

// Connect for client
func (us *UnixDomainSocket) Connect() (net.Conn, error) {
	// parse
	proto, addr, err := parseEndpoint(us.filename)
	if err != nil {
		klog.Errorf("failed to parseEndpoint: %v", err)
		return nil, err
	}

	// dial
	c, err := net.Dial(proto, addr)
	if err != nil {
		klog.Errorf("failed to dial: %v", err)
		return nil, err
	}
	return c, nil
}

// Send msg for client
func (us *UnixDomainSocket) Send(c net.Conn, context string) (string, error) {
	// send msg
	_, err := c.Write([]byte(context))
	if err != nil {
		klog.Errorf("failed to write buffer: %v", err)
		return "", err
	}

	// read response
	buf := make([]byte, us.buffersize)
	nr, err := c.Read(buf)
	if err != nil {
		klog.Errorf("failed to read buffer: %v", err)
		return "", err
	}
	return string(buf[0:nr]), nil
}
