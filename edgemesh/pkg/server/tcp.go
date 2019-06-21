package server

import (
	"io"
	"io/ioutil"
	"net"
	"net/http"

	"github.com/go-chassis/go-chassis/core/common"
	"github.com/go-chassis/go-chassis/core/handler"
	"github.com/go-chassis/go-chassis/core/invocation"
	"github.com/kubeedge/beehive/pkg/common/config"
	"github.com/kubeedge/beehive/pkg/common/log"
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

func httpResponseToStr(resp *http.Response) string {
	respString := resp.Proto + " " + resp.Status + "\n"
	for key, values := range resp.Header {
		respString += key + ": "
		for _, v := range values {
			respString += v + ", "
		}
		respString = respString[0 : len(respString) - 2]
		respString += "\n"
	}
	b, _  := ioutil.ReadAll(resp.Body)
	respString += "\n" + string(b)
	return respString
}

func process(conn net.Conn) {
	log.LOGGER.Info("start receiving data...\n")

	buffer := make([]byte, 1024)
	d := make(chan []byte, 1024)
	s := make(chan interface{}, 1)
	restResponse := func(data *invocation.Response) error {
		if data.Err != nil {
			log.LOGGER.Errorf("error in response:%v", data.Err)
			conn.Write([]byte(data.Err.Error()))
			return data.Err
		} else {
			if data.Result != nil {
				conn.Write([]byte(httpResponseToStr(data.Result.(*http.Response))))
			}
		}
		return nil
	}
	fakeResponse := func(data *invocation.Response) error {
		defer conn.Close()
		if data.Err != nil {
			log.LOGGER.Errorf("error in response:%v", data.Err)
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
		if invocation.Protocol == "rest" {
			c, err := handler.CreateChain(common.Consumer, protocol, handler.Loadbalance, handler.Transport)
			if err != nil {
				log.LOGGER.Errorf("failed to create handlerchain:%v", err)
			}
			c.Next(&invocation, restResponse)
		} else {
			c, err := handler.CreateChain(common.Consumer, protocol)
			if err != nil {
				log.LOGGER.Errorf("failed to create handlerchain:%v", err)
			}
			c.Next(&invocation, fakeResponse)
		}
	}

	//Start resolver
	go resolver.Resolve(d, s, invocationCallback)
	for {
		num, err := conn.Read(buffer)
		if err == nil {
			log.LOGGER.Infof("buffer:\n%s\n", buffer)
			d <- buffer[:num]
		}
		if err == io.EOF {
			close(s)
			return
		}
	}
}
