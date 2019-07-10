package server

import (
	"io"
	"net"
	"net/http"

	"github.com/go-chassis/go-chassis/core/common"
	"github.com/go-chassis/go-chassis/core/handler"
	"github.com/go-chassis/go-chassis/core/invocation"
	"github.com/kubeedge/beehive/pkg/common/config"
	"github.com/kubeedge/beehive/pkg/common/log"
	edgemeshCommon "github.com/kubeedge/kubeedge/edgemesh/pkg/common"
	"github.com/kubeedge/kubeedge/edgemesh/pkg/resolver"
)

func StartTCP() {
	server := config.GetString("server", "0.0.0.0")
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
	log.LOGGER.Info("start receiving data...\n")

	buffer := make([]byte, 1024)
	d := make(chan []byte, 1024)
	s := make(chan interface{}, 1)
	responseCallback := func(data *invocation.Response) error {
		if data.Err != nil {
			log.LOGGER.Errorf("error in response:%v", data.Err)
			conn.Write([]byte(data.Err.Error()))
			return data.Err
		} else {
			if data.Result != nil {
				switch data.Result.(type) {
				case *http.Response:
					conn.Write([]byte(edgemeshCommon.HTTPResponseToStr(data.Result.(*http.Response))))
				default:
					conn.Write(data.Result.([]byte))
				}
			}
		}
		return nil
	}
	invocationCallback := func(protocol string, invocation invocation.Invocation, handlerNames []string, needCloseConn bool) {
		if needCloseConn {
			defer conn.Close()
		}
		c, _ := handler.CreateChain(common.Consumer, protocol)
		for _, handlerName := range handlerNames {
			handlerToAdd, err := handler.CreateHandler(handlerName)
			if err != nil {
				log.LOGGER.Errorf("Create handler %s failed with error: %v", handlerName, err)
			}
			c.AddHandler(handlerToAdd)
		}
		c.Next(&invocation, responseCallback)
	}

	//Start resolver
	go resolver.Resolve(d, s, invocationCallback)
	for {
		num, err := conn.Read(buffer)
		if err == nil {
			log.LOGGER.Infof("buffer:\n%s\n", buffer[:num])
			d <- buffer[:num]
		}
		if err == io.EOF {
			close(s)
			return
		}
	}
}
