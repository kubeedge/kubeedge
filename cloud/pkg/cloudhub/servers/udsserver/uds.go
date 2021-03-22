package udsserver

import (
	"fmt"
	"net"
	"os"
	"path/filepath"
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
	if len(buffersize) != 0 {
		size = buffersize[0]
	}
	return &UnixDomainSocket{filename: filename, buffersize: size}
}

// parseEndpoint parses endpoint
func parseEndpoint(ep string) (string, string, error) {
	lep := strings.ToLower(ep)
	if strings.HasPrefix(lep, "unix://") || strings.HasPrefix(lep, "tcp://") {
		s := strings.SplitN(ep, "://", 2)
		proto, path := strings.ToLower(s[0]), strings.TrimSpace(s[1])
		if path != "" {
			return proto, path, nil
		}
	}
	return "", "", fmt.Errorf("invalid endpoint: %s", ep)
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
		addr = filepath.Join("/" + addr)
		if err := checkUnixSocket(addr); err != nil {
			klog.Errorf("failed to check unix socket addr: %v", err)
			return err
		}
		defer os.Remove(addr)
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

func checkUnixSocket(addr string) error {
	fileInfo, err := os.Stat(addr)
	if err != nil {
		if os.IsNotExist(err) {
			// ensure parent directory is created
			if err := os.MkdirAll(filepath.Dir(addr), 0770); err != nil {
				return err
			}
			return nil
		}
		return err
	}

	// addr cannot be a directory
	if fileInfo.IsDir() {
		return fmt.Errorf("%s is dir", addr)
	}

	if fileInfo.Mode()&os.ModeSocket == 0 {
		return fmt.Errorf("%s already is existed, but not unix socket", addr)
	}

	// addr already is existed and it is unix socket, remove it
	if err := os.Remove(addr); err != nil {
		return fmt.Errorf("failed to remove addr: %v", err)
	}
	return nil
}
