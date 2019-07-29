package util

import (
	"fmt"
	"net"
	"os"
	"time"
)

// UnixSocket struct
type UnixSocket struct {
	filename string
	bufsize  int
	handler  func(string) string
}

// NewUnixSocket create new socket
func NewUnixSocket(filename string, size ...int) *UnixSocket {
	size1 := 10480
	if size != nil {
		size1 = size[0]
	}
	us := UnixSocket{filename: filename, bufsize: size1}
	return &us
}

func (us *UnixSocket) createServer() {
	os.Remove(us.filename)
	addr, err := net.ResolveUnixAddr("unix", us.filename)
	if err != nil {
		panic("Cannot resolve unix addr: " + err.Error())
	}
	listener, err := net.ListenUnix("unix", addr)
	defer listener.Close()
	if err != nil {
		panic("Cannot listen to unix domain socket: " + err.Error())
	}
	fmt.Println("Listening on", listener.Addr())
	for {
		c, err := listener.Accept()
		fmt.Printf("Connected from %v", c)
		if err != nil {
			panic("Accept: " + err.Error())
		}
		go us.HandleServerConn(c)
	}

}

// HandleServerConn handler sever
func (us *UnixSocket) HandleServerConn(c net.Conn) {
	defer c.Close()
	buf := make([]byte, us.bufsize)
	nr, err := c.Read(buf)
	if err != nil {
		panic("Read: " + err.Error())
	}
	result := us.HandleServerContext(string(buf[0:nr]))
	_, err = c.Write([]byte(result))
	if err != nil {
		panic("Writes failed.")
	}
}

// SetContextHandler set handler
func (us *UnixSocket) SetContextHandler(f func(string) string) {
	us.handler = f
}

// HandleServerContext handler ctx
func (us *UnixSocket) HandleServerContext(context string) string {
	if us.handler != nil {
		return us.handler(context)
	}
	now := time.Now().String()
	return now
}

// StartServer start server
func (us *UnixSocket) StartServer() {
	us.createServer()
}

// ClientSendContext side
func (us *UnixSocket) ClientSendContext(context string) string {
	addr, err := net.ResolveUnixAddr("unix", us.filename)
	if err != nil {
		panic("Cannot resolve unix addr: " + err.Error())
	}

	c, err := net.DialUnix("unix", nil, addr)
	if err != nil {
		panic("DialUnix failed.")
	}
	_, err = c.Write([]byte(context))
	if err != nil {
		panic("Writes failed.")
	}
	buf := make([]byte, us.bufsize)
	nr, err := c.Read(buf)
	if err != nil {
		panic("Read: " + err.Error())
	}
	return string(buf[0:nr])
}

// Connect connect
func (us *UnixSocket) Connect() *net.UnixConn {
	addr, err := net.ResolveUnixAddr("unix", us.filename)
	if err != nil {
		panic("Cannot resolve unix addr: " + err.Error())
	}

	c, err := net.DialUnix("unix", nil, addr)
	if err != nil {
		panic("DialUnix failed.")
	}
	return c
}

// Send msg
func (us *UnixSocket) Send(c *net.UnixConn, context string) string {
	_, err := c.Write([]byte(context))
	if err != nil {
		panic("Writes failed.")
	}
	buf := make([]byte, us.bufsize)
	nr, err := c.Read(buf)
	if err != nil {
		panic("Read: " + err.Error())
	}
	return string(buf[0:nr])
}
