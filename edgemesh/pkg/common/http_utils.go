package common

import (
	"context"
	"io/ioutil"
	"net/http"
	"strings"

	"github.com/go-chassis/go-chassis/core/invocation"
	"github.com/kubeedge/beehive/pkg/common/log"
)

func httpMethods() (methods []string) {
	methods = []string{"GET", "HEAD", "POST", "OPTIONS", "PUT", "DELETE", "TRACE", "CONNECT"}
	return
}

func IsHTTPRequest(s string) bool {
	methods := httpMethods()
	for _, method := range methods {
		if strings.HasPrefix(s, method) {
			return true
		}
	}
	return false
}

func HTTPResponseToStr(resp *http.Response) (respString string) {
	if resp == nil {
		log.LOGGER.Error("http response is nil")
	} else {
		defer resp.Body.Close()
		respString = resp.Proto + " " + resp.Status + "\n"
		for key, values := range resp.Header {
			respString += key + ": "
			for _, v := range values {
				respString += v + ", "
			}
			respString = respString[0 : len(respString) - 2]
			respString += "\n"
		}
		b, err := ioutil.ReadAll(resp.Body)
		if err != nil {
			log.LOGGER.Errorf("read response body to buffer error: %v", err)
			return
		}
		respString += "\n" + string(b)
	}
	return
}

func NewInvocationForHTTPResolver(req *http.Request) invocation.Invocation {
	if req != nil {
		i := invocation.New(context.Background())
		i.MicroServiceName = req.Host
		i.SourceServiceID = ""
		i.Protocol = "rest"
		i.Args = req
		i.Strategy = "Random"
		i.Reply = &http.Response{}
		return *i
	} else {
		log.LOGGER.Error("http request is nil when constructing invocation")
		return invocation.Invocation{}
	}
}
