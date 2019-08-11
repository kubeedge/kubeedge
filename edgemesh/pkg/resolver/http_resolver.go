package resolver

import (
	"bufio"
	"bytes"
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
				invCallback(protocol, common.NewInvocationForHTTPResolver(req), []string{handler.Loadbalance, handler.Transport}, false)
			}
		case <-stop:
			i := invocation.Invocation{}
			invCallback(protocol, i, []string{}, true)
			return i, true
		}
		log.LOGGER.Infof("content: %s\n", content)
	}
}

//Only difference between HTTPResolver and HTTPTestResolver is the handler they use in invCallback
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
				invCallback(protocol, common.NewInvocationForHTTPResolver(req), []string{"httpTestHandler"}, false)
			}
		case <-stop:
			i := invocation.Invocation{}
			invCallback(protocol, i, []string{}, true)
			return i, true
		}
		log.LOGGER.Infof("content: %s\n", content)
	}
}
