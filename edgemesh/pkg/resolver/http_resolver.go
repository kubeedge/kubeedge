package resolver

import (
	"bufio"
	"bytes"
	"context"
	"net/http"

	"github.com/go-chassis/go-chassis/core/handler"
	"github.com/go-chassis/go-chassis/core/invocation"
	"github.com/kubeedge/beehive/pkg/common/log"
	"github.com/kubeedge/kubeedge/edgemesh/pkg/common"
)

type HTTPResolver struct{}

func (resolver *HTTPResolver) Resolve(data chan []byte, stop chan interface{}, invCallback func(string, invocation.Invocation, []string, bool)) (invocation.Invocation, bool) {
	content := ""
	protocol := ""
	for {
		select {
		case d := <-data:
			strData := string(d[:])
			if protocol == "" {
				if common.IsHTTPRequest(strData) {
					protocol = "http"
				} else {
					return invocation.Invocation{}, false
				}
			}
			content += strData
			req, err := http.ReadRequest(bufio.NewReader(bytes.NewReader([]byte(content))))
			if err == nil {
				content = ""
				req.RequestURI = ""
				i := invocation.New(context.Background())
				i.MicroServiceName = req.Host
				i.SourceServiceID = ""
				i.Protocol = "rest"
				i.Args = req
				i.Strategy = "Random"
				i.Reply = &http.Response{}
				invCallback(protocol, *i, []string{handler.Loadbalance, handler.Transport}, false)
			}
		case <-stop:
			i := invocation.Invocation{}
			invCallback(protocol, i, []string{}, true)
			return i, true
		}
		log.LOGGER.Infof("content: %s\n", content)
	}
}

//Only difference between HTTPResolver and HTTPTestResolver are the handlers they use in invCallback
type HTTPTestResolver struct{}

func (resolver *HTTPTestResolver) Resolve(data chan []byte, stop chan interface{}, invCallback func(string, invocation.Invocation, []string, bool)) (invocation.Invocation, bool) {
	content := ""
	protocol := ""
	for {
		select {
		case d := <-data:
			strData := string(d[:])
			if protocol == "" {
				if common.IsHTTPRequest(strData) {
					protocol = "http"
				} else {
					return invocation.Invocation{}, false
				}
			}
			content += strData
			req, err := http.ReadRequest(bufio.NewReader(bytes.NewReader([]byte(content))))
			if err == nil {
				content = ""
				req.RequestURI = ""
				i := invocation.New(context.Background())
				i.MicroServiceName = req.Host
				i.SourceServiceID = ""
				i.Protocol = "rest"
				i.Args = req
				i.Strategy = "Random"
				i.Reply = &http.Response{}
				invCallback(protocol, *i, []string{"httpTestHandler"}, false)
			}
		case <-stop:
			i := invocation.Invocation{}
			invCallback(protocol, i, []string{}, true)
			return i, true
		}
		log.LOGGER.Infof("content: %s\n", content)
	}
}
