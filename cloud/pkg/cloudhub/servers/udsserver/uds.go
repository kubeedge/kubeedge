package udsserver

import (
	"fmt"
	"net"
	"os"
	"strings"

	"k8s.io/klog/v2"
)

const (
	// DefaultBufferSize represents default buffer size
	DefaultBufferSize = 10480
)

// UnixDomainSocket struct
type UnixDomainSocket struct {
	filename   string
	buffersize int
	handler    func(string) string
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

// parseEndpoint parses endpoint
func parseEndpoint(ep string) (string, string, error) {
	if strings.HasPrefix(strings.ToLower(ep), "unix://") || strings.HasPrefix(strings.ToLower(ep), "tcp://") {
		s := strings.SplitN(ep, "://", 2)
		if s[1] != "" {
			return s[0], s[1], nil
		}
	}
	return "", "", fmt.Errorf("invalid endpoint: %v", ep)
}

// SetContextHandler set handler for server
func (us *UnixDomainSocket) SetContextHandler(f func(string) string) {
	us.handler = f
}

// StartServer start for server
func (us *UnixDomainSocket) StartServer() error {
	proto, addr, err := parseEndpoint(us.filename)
	if err != nil {
		klog.Errorf("failed to parseEndpoint: %v", err)
		return err
	}
	if proto == "unix" {
		addr = "/" + addr
		if err := os.Remove(addr); err != nil && !os.IsNotExist(err) { //nolint: vetshadow
			klog.Errorf("failed to remove addr: %v", err)
			return err
		}
	}

	// Listen
	listener, err := net.Listen(proto, addr)
	if err != nil {
		klog.Errorf("failed to listen addr: %v", err)
		return err
	}
	defer listener.Close()
	klog.Infof("listening on: %v", listener.Addr())

	for {
		c, err := listener.Accept()
		if err != nil {
			klog.Errorf("accept to error: %v", err)
			continue
		}
		go us.handleServerConn(c)
	}
}

// handleServerConn handler for server
func (us *UnixDomainSocket) handleServerConn(c net.Conn) {
	defer c.Close()
	buf := make([]byte, us.buffersize)
	nr, err := c.Read(buf)
	if err != nil {
		klog.Errorf("failed to read buffer: %v", err)
		return
	}
	result := us.handleServerContext(string(buf[0:nr]))
	_, err = c.Write([]byte(result))
	if err != nil {
		klog.Errorf("failed to write buffer: %v", err)
	}
}

// HandleServerContext handler for server
func (us *UnixDomainSocket) handleServerContext(context string) string {
	if us.handler != nil {
		return us.handler(context)
	}
	return ""
}
