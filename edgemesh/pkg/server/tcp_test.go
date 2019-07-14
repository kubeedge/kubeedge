package server_test

import (
	"container/list"
	"fmt"
	"io/ioutil"
	"net"
	"net/http"
	"net/url"
	"strings"
	"testing"
	"time"

	"github.com/go-chassis/go-chassis/core/handler"
	"github.com/go-chassis/go-chassis/core/invocation"
	edgemeshHandler "github.com/kubeedge/kubeedge/edgemesh/pkg/handler"
	"github.com/kubeedge/kubeedge/edgemesh/pkg/resolver"
	"github.com/kubeedge/kubeedge/edgemesh/pkg/server"
)

type TestResolver struct {
	Name string
}

func (resolver *TestResolver) Resolve(data chan []byte, stop chan interface{}, invCallback func(string, invocation.Invocation, []string, bool)) (invocation.Invocation, bool) {
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
			invCallback(protocol, i, []string{}, true)
			return i, true
		}
		fmt.Printf("content: %s\n", content)
	}
}

var isTCPServerStarted bool = false

func TestStartTCP(t *testing.T) {
	//Register resolver
	r1 := &TestResolver{"http"}
	r2 := &TestResolver{"grpc"}
	resolver.RegisterResolver(r1)
	resolver.RegisterResolver(r2)
	defer func() {
		resolver.ResolverChain = list.New()
	}()
	if !isTCPServerStarted {
		go server.StartTCP()
		isTCPServerStarted = true
	}
	time.Sleep(3 * time.Second)
	//============================================
	conn, err := net.Dial("tcp", "localhost:8080")
	if err != nil {
		t.Errorf("connect failed, err : %v\n", err.Error())
		return
	}

	testString := "https://e.huawei.com/en/solutions/industries/manufacturing/individual-requirements/e-commerce"
	trimmedInput := strings.TrimSpace(testString)
	_, err = conn.Write([]byte(trimmedInput))
	if err != nil {
		t.Errorf("write failed , err : %v\n", err)
	}
	conn.Close()
	//============================================
	time.Sleep(10 * time.Second)
}

func assertStringEqual(t *testing.T, a, b string) {
	if a != b {
		t.Errorf("Not Equal. %s %s", a, b)
	}
}

func helloHTTP(w http.ResponseWriter, r *http.Request) {
	fmt.Fprintf(w, "Hello "+r.Method)
}

func TestResolveHTTP(t *testing.T) {
	//Register HTTP test resolver
	r := &resolver.HTTPTestResolver{}
	resolver.RegisterResolver(r)
	defer func() {
		resolver.ResolverChain = list.New()
	}()
	//Initialize test handler
	handler.RegisterHandler("httpTestHandler", edgemeshHandler.NewHTTPTestHandler)
	//start TCP server
	if !isTCPServerStarted {
		go server.StartTCP()
		isTCPServerStarted = true
	}
	//start an HTTP server where test handler transfers requests
	go func() {
		http.HandleFunc("/", helloHTTP)
		err := http.ListenAndServe(":8888", nil)
		if err != nil {
			t.Errorf("ListenAndServe error: %v\n", err)
		}
	}()
	time.Sleep(3 * time.Second)

	//new http client
	clt := &http.Client{}

	//do GET request
	req, err := http.NewRequest("GET", "http://127.0.0.1:8080", nil)
	if err != nil {
		t.Errorf("new http GET request error: %v", err)
		return
	}
	getResp, err := clt.Do(req)
	if getResp != nil {
		defer getResp.Body.Close()
	}
	if err != nil {
		t.Errorf("do http GET request failed with error: %v", err)
		return
	}
	getRespBody, _ := ioutil.ReadAll(getResp.Body)
	assertStringEqual(t, string(getRespBody), "Hello GET")

	//do POST request
	data := url.Values{"FirstName": {"Edge"}, "LastName": {"Mesh"}}
	body := strings.NewReader(data.Encode())
	req, err = http.NewRequest("POST", "http://127.0.0.1:8080", body)
	if err != nil {
		t.Errorf("new http POST request error: %v", err)
		return
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	postResp, err := clt.Do(req)
	if postResp != nil {
		defer postResp.Body.Close()
	}
	if err != nil {
		t.Errorf("do http POST request failed with error: %v", err)
		return
	}
	postRespBody, _ := ioutil.ReadAll(postResp.Body)
	assertStringEqual(t, string(postRespBody), "Hello POST")

	time.Sleep(3 * time.Second)
}
