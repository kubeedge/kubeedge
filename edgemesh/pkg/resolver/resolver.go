/*
Copyright 2019 The KubeEdge Authors.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/
package resolver

import (
	"bufio"
	"bytes"
	"context"
	"net/http"
	"strings"

	"github.com/go-chassis/go-chassis/core/invocation"
	"github.com/kubeedge/beehive/pkg/common/log"
)

type Resolver interface {
	Resolve(chan []byte, chan interface{}, func(string, invocation.Invocation)) (invocation.Invocation, bool)
}

type MyResolver struct {
	Name string
}

func httpMethods() (methods []string) {
	methods = []string{"GET", "HEAD", "POST", "OPTIONS", "PUT", "DELETE", "TRACE", "CONNECT"}
	return
}

func isHTTPRequest(s string) bool {
	methods := httpMethods()
	for _, method := range methods {
		if strings.HasPrefix(s, method) {
			return true
		}
	}
	return false
}

func (resolver *MyResolver) Resolve(data chan []byte, stop chan interface{}, invCallback func(string, invocation.Invocation)) (invocation.Invocation, bool) {
	content := ""
	protocol := ""
	for {
		select {
		case d := <-data:
			strData := string(d[:])
			if protocol == "" {
				if isHTTPRequest(strData) {
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
				invCallback("http", *i)
			}
		case <-stop:
			i := invocation.Invocation{MicroServiceName: resolver.Name, Args: content}
			invCallback(protocol, i)
			return i, true
		}
		log.LOGGER.Infof("content: %s\n", content)
	}
}
