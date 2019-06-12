package server

import (
	"fmt"
	"io"
	"net"

	"github.com/kubeedge/kubeedge/edgemesh/pkg/resolver"

	"github.com/kubeedge/beehive/pkg/common/config"
	"github.com/kubeedge/beehive/pkg/common/log"

	"github.com/go-chassis/go-chassis/core/common"
	"github.com/go-chassis/go-chassis/core/handler"
	"github.com/go-chassis/go-chassis/core/invocation"
)

func StartTCP() {
	server := config.GetString("server", "127.0.0.1")
	port := config.GetString("port", "8080")

	log.LOGGER.Infof("start listening at %s:%s", server, port)

	listener, err := net.Listen("tcp", server+":"+port)
	if err != nil {
		log.LOGGER.Errorf("failed to start TCP server with error:%v\n", err)
		return
	}

	for {
		conn, err := listener.Accept()
		if err != nil {
			log.LOGGER.Errorf("failed to accept, err: %v\n", err)
			continue
		}

		go process(conn)
	}
}

func process(conn net.Conn) {
	fmt.Printf("start receiving data...")

	buffer := make([]byte, 1024)
	d := make(chan []byte, 1024)
	s := make(chan interface{}, 1)
	response := func(data *invocation.Response) error {
		defer conn.Close()
		if data.Err != nil {
			log.LOGGER.Errorf("error in response:v%", data.Err)
			conn.Write([]byte(data.Err.Error()))
			return data.Err
		} else {
			if data.Result != nil {
				conn.Write(data.Result.([]byte))
			}
		}
		return nil
	}
	invocationCallback := func(protocol string, invocation invocation.Invocation) {
		c, err := handler.CreateChain(common.Consumer, protocol)
		if err != nil {
			log.LOGGER.Errorf("failed to create handlerchain:v%", err)
		}
		c.Next(&invocation, response)
	}

	//Start resolver
	go resolver.Resolve(d, s, invocationCallback)
	for {
		_, err := conn.Read(buffer)
		d <- buffer
		if err == io.EOF {
			close(s)
			return
		}
	}
}
