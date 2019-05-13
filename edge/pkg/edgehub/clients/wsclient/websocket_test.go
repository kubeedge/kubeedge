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

package wsclient

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"reflect"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/gorilla/websocket"
	"github.com/kubeedge/beehive/pkg/core/model"
	"github.com/kubeedge/kubeedge/edge/pkg/common/util"
	"github.com/satori/go.uuid"
)

//init() starts the test server and generates test certificates for testing
func init() {
	newTestServer()

	err := util.GenerateTestCertificate("/tmp/", "edge", "edge")
	if err != nil {
		panic("Error in creating fake certificates")
	}
}

//Message is content object to be passed in as message object
type Message struct {
	Name string `json:"name"`
}

var testServer *httptest.Server
var upgrader websocket.Upgrader

func newTestWebSocketClient(api string, certPath string, keyPath string) *WebSocketClient {
	return &WebSocketClient{
		//webConn:  &websocket.Conn{},
		sendLock: sync.Mutex{},
		config: &WebSocketConfig{
			URL:              "ws://" + testServer.Listener.Addr().String() + "/" + api,
			CertFilePath:     certPath,
			KeyFilePath:      keyPath,
			HandshakeTimeout: 500 * time.Second,
			WriteDeadline:    100 * time.Second,
			ReadDeadline:     100 * time.Second,
			ExtendHeader:     http.Header{},
		},
	}
}

//newTestServer() starts a fake server for testing
func newTestServer() {
	testServer = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch {
		case strings.Contains(r.RequestURI, "/normal"):
			c, err := upgrader.Upgrade(w, r, nil)
			if err != nil {
				return
			}
			defer c.Close()
			m := model.Message{}
			for {
				err := c.ReadJSON(&m)
				if err != nil {
					break
				}
				err = c.WriteJSON(m)
				if err != nil {
					break
				}
			}

		case strings.Contains(r.RequestURI, "/bad_request"):
			w.WriteHeader(http.StatusBadRequest)

		case strings.Contains(r.RequestURI, "/wrong_send"):
			c, err := upgrader.Upgrade(w, r, nil)
			if err != nil {
				return
			}
			defer c.Close()
			m := model.Message{}
			for {
				err := c.ReadJSON(&m)
				if err != nil {
					break
				}
				err = c.WriteMessage(3, []byte(""))
				if err != nil {
					break
				}
			}
		}
	}))
}

//TestNewWebSocketClient tests the NewWebSocketClient function that creates the WebSocketClient object
func TestNewWebSocketClient(t *testing.T) {
	tests := []struct {
		name string
		conf *WebSocketConfig
		want *WebSocketClient
	}{
		{"TestNewWebSocketClient: ",
			&WebSocketConfig{
				URL:              "ws://" + testServer.Listener.Addr().String() + "/normal",
				CertFilePath:     "/tmp/edge.crt",
				KeyFilePath:      "/tmp/edge.key",
				HandshakeTimeout: 500 * time.Second,
				WriteDeadline:    100 * time.Second,
				ReadDeadline:     100 * time.Second,
				ExtendHeader:     http.Header{},
			},
			newTestWebSocketClient("normal", "/tmp/edge.crt", "/tmp/edge.key"),
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := NewWebSocketClient(tt.conf); !reflect.DeepEqual(got, tt.want) {
				t.Errorf("NewWebSocketClient() got = %v, want %v", got, tt.want)
			}
		})
	}
}

//TestInit tests the procurement of the WebSocketClient
func TestInit(t *testing.T) {
	tests := []struct {
		name          string
		fields        *WebSocketClient
		expectedError error
	}{
		{name: "TestInit: Success in connection ",
			fields:        newTestWebSocketClient("normal", "/tmp/edge.crt", "/tmp/edge.key"),
			expectedError: nil,
		},
		{name: "TestInit: If Certificate files not loaded properly",
			fields:        newTestWebSocketClient("normal", "/wrong_path.crt", "/wrong_path.key"),
			expectedError: fmt.Errorf("failed to load x509 key pair, error: open /wrong_path.crt: no such file or directory"),
		},
		{name: "TestInit: Error in dial call returned with valid resp object ",
			fields:        newTestWebSocketClient("bad_request", "/tmp/edge.crt", "/tmp/edge.key"),
			expectedError: fmt.Errorf("max retry count to connect Access"),
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			wcc := tt.fields
			err := wcc.Init()
			if !reflect.DeepEqual(err, tt.expectedError) {
				t.Errorf("WebSocketClient.Init() error = %v, expectedError =  %v", err, tt.expectedError)
			}
		})
	}
}

//TestUninit tests the Uninit function by trying to access the connection object
func TestUninit(t *testing.T) {
	tests := []struct {
		name   string
		fields *WebSocketClient
	}{
		{name: "TestUninit ",
			fields: newTestWebSocketClient("normal", "/tmp/edge.crt", "/tmp/edge.key"),
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			wcc := tt.fields
			wcc.Init()
			wcc.Uninit()
			err := wcc.webConn.WriteMessage(2, []byte(""))
			if err == nil {
				t.Errorf("WebSocketClient.Uninit")
			}
		})
	}
}

//TestSend checks send function by sending message to server
func TestSend(t *testing.T) {
	var msg = model.Message{Header: model.MessageHeader{ID: uuid.NewV4().String(), ParentID: "12", Timestamp: time.Now().UnixNano() / 1e6},
		Content: "test",
	}
	tests := []struct {
		name          string
		fields        *WebSocketClient
		message       model.Message
		expectedError error
	}{
		{name: "Test sending small message: ",
			fields:        newTestWebSocketClient("normal", "/tmp/edge.crt", "/tmp/edge.key"),
			message:       msg,
			expectedError: nil,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			wcc := tt.fields
			//First run init
			wcc.Init()

			if err := wcc.Send(tt.message); !reflect.DeepEqual(err, tt.expectedError) {
				t.Errorf("WebSocketClient.Send() error = %v, expectedError = %v", err, tt.expectedError)
			}
		})
	}
}

//TestReceive sends the message through send function then calls receive function to see same message is received or not
func TestReceive(t *testing.T) {
	var msg = model.Message{Header: model.MessageHeader{ID: uuid.NewV4().String(), ParentID: "12", Timestamp: time.Now().UnixNano() / 1e6},
		Content: "test",
	}
	tests := []struct {
		name          string
		fields        *WebSocketClient
		want          model.Message
		sent          model.Message
		expectedError error
	}{
		{name: "Test Receiving the send message: Success in receiving",
			fields:        newTestWebSocketClient("normal", "/tmp/edge.crt", "/tmp/edge.key"),
			want:          msg,
			sent:          msg,
			expectedError: nil,
		},
		{name: "Test Recieving the send message: Error in recieving ",
			fields:        newTestWebSocketClient("wrong_send", "/tmp/edge.crt", "/tmp/edge.key"),
			want:          model.Message{},
			sent:          model.Message{},
			expectedError: &websocket.CloseError{Code: 1006, Text: "unexpected EOF"},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			wcc := tt.fields
			//First run init
			wcc.Init()
			//Run send
			err := wcc.Send(tt.sent)

			if err != nil {
				t.Errorf("error = %v", err)
			}

			got, err := wcc.Receive()
			if !reflect.DeepEqual(err, tt.expectedError) {
				t.Errorf("WebSocketClient.Receive() error = %v, expectedError = %v", err, tt.expectedError)
				return
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("WebSocketClient.Receive() message: got = %v, want = %v", got, tt.want)
			}
			wcc.Uninit()
		})
	}
}
