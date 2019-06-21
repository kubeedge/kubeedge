package server_test

import (
	"bufio"
	"bytes"
	"fmt"
	"github.com/kubeedge/beehive/pkg/common/log"
	"io/ioutil"
	"net"
	"net/http"
	"net/url"
	"strings"
	"testing"
	"time"

	"github.com/go-chassis/go-chassis/core/handler"
	"github.com/go-chassis/go-chassis/core/invocation"
	"github.com/kubeedge/kubeedge/edgemesh/pkg/resolver"
	"github.com/kubeedge/kubeedge/edgemesh/pkg/server"
)

type TestResolver struct {
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

func testTransferHTTPRequest(req *http.Request) (invocation.Invocation, error) {
	clt := &http.Client{}
	u, err := url.Parse("http://127.0.0.1:9090")
	if err != nil {
		log.LOGGER.Errorf("Parse new url error: %v\n", err)
		return invocation.Invocation{}, err
	}
	req.URL = u
	//clear RequestURI
	req.RequestURI = ""

	resp, err := clt.Do(req)
	if resp != nil {
		defer resp.Body.Close()
	}
	if err != nil {
		log.LOGGER.Errorf("Resolve http request failed with error: %v\n", err)
		return invocation.Invocation{}, err
	}
	respBodyBytes, _ := ioutil.ReadAll(resp.Body)
	log.LOGGER.Infof("resolve http resp body: %s\n", respBodyBytes)
	return invocation.Invocation{MicroServiceName: "http", Protocol: "rest", Args: resp}, nil
}

func (resolver *TestResolver) Resolve(data chan []byte, stop chan interface{}, invCallback func(string, invocation.Invocation)) (invocation.Invocation, bool) {
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
				i, err := testTransferHTTPRequest(req)
				if err != nil {
					panic(err)
				}
				invCallback("http", i)
			}
		case <-stop:
			i := invocation.Invocation{MicroServiceName: resolver.Name, Args: content}
			invCallback(protocol, i)
			return i, true
		}
		log.LOGGER.Infof("content: %s\n", content)
	}
}

type TestHandler struct {}

//Handle
func (h *TestHandler) Handle(chain *handler.Chain, inv *invocation.Invocation, cb invocation.ResponseCallBack) {
	r := &invocation.Response{
		Err: nil,
	}
	resp := "HTTP/1.1 200\ncontent-type: text/plain; charset=utf-8\ncontent-length: 20\ndate: Wed, 12 Jun 2019 09:28:08 GMT\n\n{\"name\": \"Jack\"}"
	r.Result = []uint8(resp)
	cb(r)
}

//Name
func (h *TestHandler) Name() string {
	return "test"
}
func newTestHandler() handler.Handler {
	return &TestHandler{}
}

var serverStarted bool = false

var httpServerStarted bool = false

func StartTCPServer() {
	if serverStarted == false {
		//Initialize the resolvers
		r := &TestResolver{"http"}
		resolver.RegisterResolver(r)
		//Initialize the handlers
		handler.RegisterHandler("resolveHandler", newTestHandler)
		//Start server
		go server.StartTCP()
		serverStarted = true
		time.Sleep(1 * time.Second)
	}
}

func helloHTTP(w http.ResponseWriter, r *http.Request) {
	fmt.Fprintf(w, "Hello " + r.Method)
}

func StartHTTPServer() {
	if httpServerStarted == false {
		go func() {
			http.HandleFunc("/", helloHTTP)
			err := http.ListenAndServe(":9090", nil)
			if err != nil {
				log.LOGGER.Errorf("ListenAndServe error: %v\n", err)
			}
		}()
		httpServerStarted = true
		time.Sleep(1 * time.Second)
	}
}

func handleTCPRead(conn net.Conn, done chan string) {
	buf := make([]byte, 1024)
	_, err := conn.Read(buf)
	if err != nil {
		log.LOGGER.Infof("Error to read from TCP server: ", err)
		return
	}
	log.LOGGER.Info("TCP server response: " + string(buf[:]))

	done <- "done"
}

func TestTCPRawBytes(t *testing.T) {
	//start TCP server if not started
	StartTCPServer()

	//start http server (:9090) to be transferred request
	go StartHTTPServer()

	//connect to server
	conn, err := net.Dial("tcp", "localhost:8080")
	if err != nil {
		t.Errorf("connect failed, err : %v\n", err.Error())
		return
	}
	defer conn.Close()

	//handle connection read
	done := make(chan string)
	go handleTCPRead(conn, done)

	//write raw bytes to TCP server
	testString := "GET / HTTP/1.1\nHost: 127.0.0.1:8080\n"
	_, err = conn.Write([]byte(testString))
	if err != nil {
		t.Errorf("write failed , err : %v\n", err)
	}
	time.Sleep(2 * time.Second)
	testString = "User-Agent: Go-http-client/1.1\nAccept-Encoding: gzip\n\n"
	_, err = conn.Write([]byte(testString))
	if err != nil {
		t.Errorf("write failed , err : %v\n", err)
	}

	<- done
	time.Sleep(3 * time.Second)
}

func TestResolveHTTP(t *testing.T) {
	//start TCP server if not started
	StartTCPServer()

	//start http server (:9090) to be transferred request
	StartHTTPServer()

	//new http client
	clt := &http.Client{}

	//do GET request
	req, err := http.NewRequest("GET", "http://127.0.0.1:8080", nil)
	if err != nil {
		t.Errorf("new http GET request error: %v\n", err)
		return
	}
	resp1, err := clt.Do(req)
	if resp1 != nil {
		defer resp1.Body.Close()
	}
	if err != nil {
		t.Errorf("do http GET request failed with error: %v\n", err)
		return
	}
	log.LOGGER.Infof("GET response: %v\n", resp1)

	//do POST request
	data := url.Values{"Name":{"Mark"}, "Age":{"20"}}
	body := strings.NewReader(data.Encode())
	req, err = http.NewRequest("POST", "http://127.0.0.1:8080", body)
	if err != nil {
		t.Errorf("new http POST request error: %v\n", err)
		return
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	resp2, err := clt.Do(req)
	if resp2 != nil {
		defer resp2.Body.Close()
	}
	if err != nil {
		t.Errorf("do http POST request failed with error: %v\n", err)
		return
	}
	log.LOGGER.Infof("POST response: %v\n", resp2)

	time.Sleep(3 * time.Second)
}



