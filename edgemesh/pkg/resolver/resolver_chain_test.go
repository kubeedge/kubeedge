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

// Package resolver_test
package resolver_test

import (
	"strings"
	"testing"
	"time"

	"github.com/go-chassis/go-chassis/core/invocation"
	"github.com/kubeedge/beehive/pkg/common/log"
	"github.com/kubeedge/kubeedge/edgemesh/pkg/resolver"
)

type TestResolver struct {
	Name string
}

func (resolver *TestResolver) Resolve(data chan []byte, stop chan interface{}, invCallback func(string, invocation.Invocation)) (invocation.Invocation, bool) {
	content := ""
	protocol := ""
	for {
		select {
		case d := <-data:
			strData := string(d[:])
			if protocol == "" {
				//Only address HTTP
				if strings.HasPrefix(strData, resolver.Name) {
					protocol = resolver.Name
					content += strData
				} else {
					return invocation.Invocation{}, false
				}
			} else {
				content += strData
			}
		case <-stop:
			i := invocation.Invocation{MicroServiceName: resolver.Name, Args: content}
			invCallback(protocol, i)
			return i, true
		}
		log.LOGGER.Infof("content: %s\n", content)
	}
}

func TestResolve(t *testing.T) {
	//Register resolver
	r1 := &TestResolver{"http"}
	r2 := &TestResolver{"grpc"}
	resolver.RegisterResolver(r1)
	resolver.RegisterResolver(r2)
	invCallback := func(protocol string, inv invocation.Invocation) {
		log.LOGGER.Infof("protocol in invCallback:%v", protocol)
		log.LOGGER.Infof("content in invCallback: %v\n", inv.Args)
	}
	d := make(chan []byte, 1024)
	s := make(chan interface{}, 1)

	//Do resolver
	go func() {
		i, f := resolver.Resolve(d, s, invCallback)
		if !f {
			t.Error("resolver chain resolve error to no able to fire an existing resolver")
		} else {
			if i.MicroServiceName != "http" {
				t.Error("resolver chain resolve error when construct invocation")
			}
		}
	}()

	d <- []byte("http://support.huaweicloud.com/")
	time.Sleep(10 * time.Millisecond)
	d <- []byte("usermanual-ief/ief_01_0001.html")
	time.Sleep(10 * time.Millisecond)
	close(s)

	d = make(chan []byte, 1024)
	s = make(chan interface{}, 1)
	go func() {
		_, f := resolver.Resolve(d, s, invCallback)
		if f {
			t.Error("resolver chain resolve error to fired a no-existing resolver")
		}
	}()
	d <- []byte("quic://support.huaweicloud.com/")
	time.Sleep(10 * time.Millisecond)
	d <- []byte("usermanual-ief/ief_01_0001.html")
	time.Sleep(10 * time.Millisecond)
	close(s)

	time.Sleep(3 * time.Second)
}
