package server_test

import (
	"fmt"
	"net"
	"strings"
	"testing"
	"time"

	"github.com/kubeedge/kubeedge/edgemesh/pkg/resolver"
	"github.com/kubeedge/kubeedge/edgemesh/pkg/server"

	"github.com/go-chassis/go-chassis/core/invocation"
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
		fmt.Printf("content: %s\n", content)
	}
}

func TestStartTCP(t *testing.T) {
	//Register resolver
	r1 := &TestResolver{"http"}
	r2 := &TestResolver{"grpc"}
	resolver.RegisterResolver(r1)
	resolver.RegisterResolver(r2)
	go server.StartTCP()
	time.Sleep(3 * time.Second)
	//============================================
	conn, err := net.Dial("tcp", "localhost:8080")
	if err != nil {
		t.Errorf("connect failed, err : %v\n", err.Error())
		return
	}

	testString := "https://e.huawei.com/en/solutions/industries/manufacturing/individual-requirements/e-commerce"
	if err != nil {
		t.Errorf("read from console failed, err: %v\n", err)
	}
	trimmedInput := strings.TrimSpace(testString)
	_, err = conn.Write([]byte(trimmedInput))

	if err != nil {
		t.Errorf("write failed , err : %v\n", err)
	}
	conn.Close()
	//============================================
	time.Sleep(10 * time.Second)
}
