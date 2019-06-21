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

package edgehub

import (
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"reflect"
	"strings"
	"testing"
	"time"

	"github.com/golang/mock/gomock"

	bhConfig "github.com/kubeedge/beehive/pkg/common/config"
	bhUtil "github.com/kubeedge/beehive/pkg/common/util"
	"github.com/kubeedge/beehive/pkg/core"
	"github.com/kubeedge/beehive/pkg/core/context"
	"github.com/kubeedge/beehive/pkg/core/model"
	"github.com/kubeedge/kubeedge/edge/mocks/edgehub"
	connect "github.com/kubeedge/kubeedge/edge/pkg/common/cloudconnection"
	module "github.com/kubeedge/kubeedge/edge/pkg/common/modules"
	"github.com/kubeedge/kubeedge/edge/pkg/common/util"
	_ "github.com/kubeedge/kubeedge/edge/pkg/devicetwin"
	"github.com/kubeedge/kubeedge/edge/pkg/edgehub/config"
)

const (
	CertFile = "/tmp/kubeedge/certs/edge.crt"
	KeyFile  = "/tmp/kubeedge/certs/edge.key"
)

//testServer is a fake http server created for testing
var testServer *httptest.Server

// mockAdapter is a mocked adapter implementation
var mockAdapter *edgehub.MockAdapter

//init() starts the test server and generates test certificates for testing
func init() {
	newTestServer()
	err := util.GenerateTestCertificate("/tmp/kubeedge/certs/", "edge", "edge")
	if err != nil {
		panic("Error in creating fake certificates")
	}
}

// initMocks is function to initialize mocks
func initMocks(t *testing.T) {
	mockCtrl := gomock.NewController(t)
	defer mockCtrl.Finish()
	mockAdapter = edgehub.NewMockAdapter(mockCtrl)
}

//testEdgeConfigYaml is a structure which is used to generate the test YAML file to test Edgehub config components
type testEdgeConfigYaml struct {
	Edgehub edgeHubConfigYaml `yaml:"edgehub"`
}

//edgeHubConfigYaml is a structure which is used to load the websocket and controller config to generate the test YAML file
type edgeHubConfigYaml struct {
	WSConfig  webSocketConfigYaml  `yaml:"websocket"`
	CtrConfig controllerConfigYaml `yaml:"controller"`
}

//controllerConfigYaml is a structure which is used to generate the test YAML file to test controller config components
type controllerConfigYaml struct {
	Placement string `yaml:"placement,omitempty"`
}

//webSocketConfigYaml is a structure which is used to generate the test YAML file to test WebSocket config components
type webSocketConfigYaml struct {
	URL string `yaml:"url,omitempty"`
}

//newTestServer() starts a fake server for testing
func newTestServer() {
	flag := true
	testServer = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch {
		case strings.Contains(r.RequestURI, "/proper_request"):
			w.Write([]byte("ws://127.0.0.1:20000"))
		case strings.Contains(r.RequestURI, "/bad_request"):
			w.WriteHeader(http.StatusBadRequest)
		case strings.Contains(r.RequestURI, "/wrong_url"):
			if flag {
				w.WriteHeader(http.StatusNotFound)
				flag = false
			} else {
				w.Write([]byte("ws://127.0.0.1:20000"))
			}
		}
	}))
}

// get the configuration file path
func getConfigDirectory() string {
	if config, err := bhConfig.CONFIG.GetValue("config-path").ToString(); err == nil {
		return config
	}

	if config, err := bhConfig.CONFIG.GetValue("GOARCHAIUS_CONFIG_PATH").ToString(); err == nil {
		return config
	}

	return bhUtil.GetCurrentDirectory()
}

var restoreConfig map[string]interface{}

func init() {
	restoreConfig = bhConfig.CONFIG.GetConfigurations()
}

func restoreConfigBack() {
	util.GenerateTestYaml(restoreConfig, getConfigDirectory()+"/conf", "edge")
}

//TestNewEdgeHubController() tests whether the EdgeHubController returned is correct or not
func TestNewEdgeHubController(t *testing.T) {
	tests := []struct {
		name             string
		want             *Controller
		controllerConfig config.ControllerConfig
	}{
		{"Testing if EdgeHubController is returned with correct values",
			&Controller{
				config: &config.ControllerConfig{
					Protocol:        "websocket",
					HeartbeatPeriod: 150 * time.Second,
					RefreshInterval: 15 * time.Minute,
					AuthInfosPath:   "/var/IEF/secret",
					PlacementURL:    "https://test_ip:port/v1/placement_external/message_queue",
					ProjectID:       "project_id",
					NodeID:          "node_id",
				},
				stopChan:   make(chan struct{}),
				syncKeeper: make(map[string]chan model.Message),
			},
			config.ControllerConfig{
				Protocol:        "websocket",
				HeartbeatPeriod: 150 * time.Second,
				RefreshInterval: 15 * time.Minute,
				AuthInfosPath:   "/var/IEF/secret",
				PlacementURL:    "https://test_ip:port/v1/placement_external/message_queue",
				ProjectID:       "project_id",
				NodeID:          "node_id",
			},
		}}
	for _, tt := range tests {
		edgeHubConfig := config.GetConfig()
		edgeHubConfig.CtrConfig = tt.controllerConfig
		t.Run(tt.name, func(t *testing.T) {
			got := NewEdgeHubController()
			if !reflect.DeepEqual(got.context, tt.want.context) {
				t.Errorf("NewEdgeHubController() Context= %v, want %v", got.context, tt.want.context)
			}
			if !reflect.DeepEqual(got.config, tt.want.config) {
				t.Errorf("NewEdgeHubController() Config = %v, want %v", got.config, tt.want.config)
			}
			if !reflect.DeepEqual(got.chClient, tt.want.chClient) {
				t.Errorf("NewEdgeHubController() chClient = %v, want %v", got.chClient, tt.want.chClient)
			}
			if !reflect.DeepEqual(got.keeperLock, tt.want.keeperLock) {
				t.Errorf("NewEdgeHubController() KeeperLock = %v, want %v", got.keeperLock, tt.want.keeperLock)
			}
			if !reflect.DeepEqual(got.syncKeeper, tt.want.syncKeeper) {
				t.Errorf("NewEdgeHubController() SyncKeeper = %v, want %v", got.syncKeeper, tt.want.syncKeeper)
			}
		})
	}
}

//TestInitial() tests the procurement of the cloudhub client
func TestInitial(t *testing.T) {
	controllerConfig := config.ControllerConfig{
		Protocol:        "websocket",
		HeartbeatPeriod: 150 * time.Second,
		RefreshInterval: 15 * time.Minute,
		AuthInfosPath:   "/var/IEF/secret",
		PlacementURL:    testServer.URL + "/proper_request",
		ProjectID:       "foo",
		NodeID:          "bar",
	}
	tests := []struct {
		name             string
		controller       Controller
		webSocketConfig  config.WebSocketConfig
		controllerConfig config.ControllerConfig
		ctx              *context.Context
		expectedError    error
	}{
		{"Valid input", Controller{
			config:     &controllerConfig,
			stopChan:   make(chan struct{}),
			syncKeeper: make(map[string]chan model.Message),
		}, config.WebSocketConfig{
			CertFilePath: CertFile,
			KeyFilePath:  KeyFile,
		}, config.ControllerConfig{
			PlacementURL: testServer.URL,
			ProjectID:    "foo",
			NodeID:       "bar",
		},
			context.GetContext(context.MsgCtxTypeChannel), nil},

		{"Wrong placement URL", Controller{
			config:     &controllerConfig,
			stopChan:   make(chan struct{}),
			syncKeeper: make(map[string]chan model.Message),
		}, config.WebSocketConfig{
			CertFilePath: CertFile,
			KeyFilePath:  KeyFile,
		}, config.ControllerConfig{
			Protocol:     "websocket",
			PlacementURL: testServer.URL,
			ProjectID:    "foo",
			NodeID:       "bar",
		},
			context.GetContext(context.MsgCtxTypeChannel), nil},

		{"No project Id & node Id", Controller{
			config:     &controllerConfig,
			stopChan:   make(chan struct{}),
			syncKeeper: make(map[string]chan model.Message),
		}, config.WebSocketConfig{
			CertFilePath: CertFile,
			KeyFilePath:  KeyFile,
		}, config.ControllerConfig{
			Protocol:     "websocket",
			PlacementURL: testServer.URL,
			ProjectID:    "",
			NodeID:       "",
		},
			context.GetContext(context.MsgCtxTypeChannel), nil},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			edgeHubConfig := config.GetConfig()
			edgeHubConfig.WSConfig = tt.webSocketConfig
			edgeHubConfig.CtrConfig = tt.controllerConfig
			if err := tt.controller.initial(tt.ctx); err != tt.expectedError {
				t.Errorf("EdgeHubController_initial() error = %v, expectedError %v", err, tt.expectedError)
			}
		})
	}
}

//TestAddKeepChannel() tests the addition of channel to the syncKeeper
func TestAddKeepChannel(t *testing.T) {
	tests := []struct {
		name       string
		controller Controller
		msgID      string
	}{
		{"Adding a valid keep channel", Controller{
			syncKeeper: make(map[string]chan model.Message),
		},
			"test"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := tt.controller.addKeepChannel(tt.msgID)
			if !reflect.DeepEqual(tt.controller.syncKeeper[tt.msgID], got) {
				t.Errorf("TestController_addKeepChannel() = %v, want %v", got, tt.controller.syncKeeper[tt.msgID])
			}
		})
	}
}

//TestDeleteKeepChannel() tests the deletion of channel in the syncKeeper
func TestDeleteKeepChannel(t *testing.T) {
	tests := []struct {
		name       string
		controller Controller
		msgID      string
	}{
		{"Deleting a valid keep channel",
			Controller{
				syncKeeper: make(map[string]chan model.Message),
			},
			"test"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tt.controller.addKeepChannel(tt.msgID)
			tt.controller.deleteKeepChannel(tt.msgID)
			if _, exist := tt.controller.syncKeeper[tt.msgID]; exist {
				t.Errorf("TestController_deleteKeepChannel = %v, want %v", tt.controller.syncKeeper[tt.msgID], nil)
			}
		})
	}
}

//TestIsSyncResponse() tests whether there exists a channel with the given message_id in the syncKeeper
func TestIsSyncResponse(t *testing.T) {
	tests := []struct {
		name       string
		controller Controller
		msgID      string
		want       bool
	}{
		{"Sync message response case",
			Controller{
				syncKeeper: make(map[string]chan model.Message),
			}, "test",
			true,
		},
		{"Non sync message response  case",
			Controller{
				syncKeeper: make(map[string]chan model.Message),
			}, "",
			false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.want {
				tt.controller.addKeepChannel(tt.msgID)
			}
			if got := tt.controller.isSyncResponse(tt.msgID); got != tt.want {
				t.Errorf("TestController_isSyncResponse() = %v, want %v", got, tt.want)
			}
		})
	}
}

//TestSendToKeepChannel() tests the reception of response in the syncKeep channel
func TestSendToKeepChannel(t *testing.T) {
	message := model.NewMessage("test_id")
	tests := []struct {
		name                string
		controller          Controller
		message             *model.Message
		keepChannelParentID string
		expectedError       error
	}{
		{"SyncKeeper Error Case in send to keep channel", Controller{
			context:    context.GetContext(context.MsgCtxTypeChannel),
			syncKeeper: make(map[string]chan model.Message),
		}, message,
			"wrong_id",
			fmt.Errorf("failed to get sync keeper channel, messageID:%+v", *message)},

		{"Negative Test Case without syncKeeper Error ", Controller{
			context:    context.GetContext(context.MsgCtxTypeChannel),
			syncKeeper: make(map[string]chan model.Message),
		}, model.NewMessage("test_id"),
			"test_id",
			fmt.Errorf("failed to send message to sync keep channel")},

		{"Send to keep channel with valid input", Controller{
			context:    context.GetContext(context.MsgCtxTypeChannel),
			syncKeeper: make(map[string]chan model.Message),
		}, model.NewMessage("test_id"),
			"test_id", nil},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			keep := tt.controller.addKeepChannel(tt.keepChannelParentID)
			if tt.expectedError == nil {
				receive := func() {
					<-keep
				}
				go receive()
			}
			time.Sleep(1 * time.Second)
			err := tt.controller.sendToKeepChannel(*tt.message)
			if !reflect.DeepEqual(err, tt.expectedError) {
				t.Errorf("TestController_sendToKeepChannel() error = %v, expectedError %v", err, tt.expectedError)
			}
		})
	}
}

//TestDispatch() tests whether the messages are properly dispatched to their respective modules
func TestDispatch(t *testing.T) {
	tests := []struct {
		name          string
		controller    Controller
		message       *model.Message
		expectedError error
		isResponse    bool
	}{
		{"dispatch with valid input", Controller{
			context:    context.GetContext(context.MsgCtxTypeChannel),
			syncKeeper: make(map[string]chan model.Message),
		},
			model.NewMessage("").BuildRouter(ModuleNameEdgeHub, module.TwinGroup, "", ""),
			nil, false},

		{"Error Case in dispatch", Controller{
			context:    context.GetContext(context.MsgCtxTypeChannel),
			syncKeeper: make(map[string]chan model.Message),
		},
			model.NewMessage("test").BuildRouter(ModuleNameEdgeHub, module.EdgedGroup, "", ""),
			fmt.Errorf("msg_group not found"), true},

		{"Response Case in dispatch", Controller{
			context:    context.GetContext(context.MsgCtxTypeChannel),
			syncKeeper: make(map[string]chan model.Message),
		},
			model.NewMessage("test").BuildRouter(ModuleNameEdgeHub, module.TwinGroup, "", ""),
			nil, true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.expectedError == nil && !tt.isResponse {
				receive := func() {
					keepChannel := tt.controller.addKeepChannel(tt.message.GetParentID())
					<-keepChannel
				}
				go receive()
			}
			time.Sleep(1 * time.Second)
			err := tt.controller.dispatch(*tt.message)
			if !reflect.DeepEqual(err, tt.expectedError) {
				t.Errorf("TestController_dispatch() error = %v, wantErr %v", err, tt.expectedError)
			}
		})
	}
}

//TestRouteToEdge() is used to test whether the message received from websocket is dispatched to the required modules
func TestRouteToEdge(t *testing.T) {
	initMocks(t)
	tests := []struct {
		name         string
		controller   Controller
		receiveTimes int
	}{
		{"Route to edge with proper input", Controller{
			context:    context.GetContext(context.MsgCtxTypeChannel),
			chClient:   mockAdapter,
			syncKeeper: make(map[string]chan model.Message),
			stopChan:   make(chan struct{}),
		}, 0},

		{"Receive Error in route to edge", Controller{
			context:    context.GetContext(context.MsgCtxTypeChannel),
			chClient:   mockAdapter,
			syncKeeper: make(map[string]chan model.Message),
			stopChan:   make(chan struct{}),
		}, 1},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockAdapter.EXPECT().Receive().Return(*model.NewMessage("test").BuildRouter(ModuleNameEdgeHub, module.EdgedGroup, "", ""), nil).Times(tt.receiveTimes)
			mockAdapter.EXPECT().Receive().Return(*model.NewMessage("test").BuildRouter(ModuleNameEdgeHub, module.TwinGroup, "", ""), nil).Times(tt.receiveTimes)
			mockAdapter.EXPECT().Receive().Return(*model.NewMessage(""), errors.New("Connection Refused")).Times(1)
			go tt.controller.routeToEdge()
			stop := <-tt.controller.stopChan
			if stop != struct{}{} {
				t.Errorf("TestRouteToEdge error got: %v want: %v", stop, struct{}{})
			}
		})
	}
}

//TestSendToCloud() tests whether the send to cloud functionality works properly
func TestSendToCloud(t *testing.T) {
	initMocks(t)
	msg := model.NewMessage("").BuildHeader("test_id", "", 1)
	msg.Header.Sync = true
	tests := []struct {
		name          string
		controller    Controller
		message       model.Message
		expectedError error
		waitError     bool
		mockError     error
	}{
		{"send to cloud with proper input", Controller{
			context:  context.GetContext(context.MsgCtxTypeChannel),
			chClient: mockAdapter,
			config: &config.ControllerConfig{
				Protocol:        "websocket",
				HeartbeatPeriod: 6 * time.Second,
			},
			syncKeeper: make(map[string]chan model.Message),
		}, *msg,
			nil,
			false,
			nil},

		{"Wait Error in send to cloud", Controller{
			chClient: mockAdapter,
			config: &config.ControllerConfig{
				Protocol:        "websocket",
				HeartbeatPeriod: 3 * time.Second,
			},
			syncKeeper: make(map[string]chan model.Message),
		}, *msg,
			nil,
			true,
			nil},

		{"Send Failure in send to cloud", Controller{
			chClient: mockAdapter,
			config: &config.ControllerConfig{
				HeartbeatPeriod: 3 * time.Second,
			},
			syncKeeper: make(map[string]chan model.Message),
		}, model.Message{},
			fmt.Errorf("failed to send message, error: Connection Refused"),
			false,
			errors.New("Connection Refused")},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockAdapter.EXPECT().Send(gomock.Any()).Return(tt.mockError).Times(1)
			if !tt.waitError && tt.expectedError == nil {
				go tt.controller.sendToCloud(tt.message)
				time.Sleep(1 * time.Second)
				tempChannel := tt.controller.syncKeeper["test_id"]
				tempChannel <- *model.NewMessage("test_id")
				time.Sleep(1 * time.Second)
				if _, exist := tt.controller.syncKeeper["test_id"]; exist {
					t.Errorf("SendToCloud() error in receiving message")
				}
				return
			}
			err := tt.controller.sendToCloud(tt.message)
			if !reflect.DeepEqual(err, tt.expectedError) {
				t.Errorf("SendToCloud() error = %v, wantErr %v", err, tt.expectedError)
			}
			time.Sleep(tt.controller.config.HeartbeatPeriod + 2*time.Second)
			if _, exist := tt.controller.syncKeeper["test_id"]; exist {
				t.Errorf("SendToCloud() error in waiting for timeout")
			}
		})
	}
}

//TestRouteToCloud() tests the reception of the message from the beehive framework and forwarding of that message to cloud
func TestRouteToCloud(t *testing.T) {
	initMocks(t)
	testContext := context.GetContext(context.MsgCtxTypeChannel)
	tests := []struct {
		name       string
		controller Controller
	}{
		{"Route to cloud with valid input", Controller{
			context:  testContext,
			chClient: mockAdapter,
			stopChan: make(chan struct{}),
		}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockAdapter.EXPECT().Send(gomock.Any()).Return(errors.New("Connection Refused")).AnyTimes()
			go tt.controller.routeToCloud()
			time.Sleep(2 * time.Second)
			core.Register(&EdgeHub{})
			testContext.AddModule(ModuleNameEdgeHub)
			msg := model.NewMessage("").BuildHeader("test_id", "", 1)
			testContext.Send(ModuleNameEdgeHub, *msg)
			stopChan := <-tt.controller.stopChan
			if stopChan != struct{}{} {
				t.Errorf("Error in route to cloud")
			}
		})
	}
}

//TestKeepalive() tests whether ping message sent to the cloud at regular intervals happens properly
func TestKeepalive(t *testing.T) {
	initMocks(t)
	tests := []struct {
		name       string
		controller Controller
	}{
		{"Heartbeat failure Case", Controller{
			config: &config.ControllerConfig{
				Protocol:     "websocket",
				PlacementURL: testServer.URL + "/proper_request",
				ProjectID:    "foo",
				NodeID:       "bar",
			},
			chClient: mockAdapter,
			stopChan: make(chan struct{}),
		}},
	}
	edgeHubConfig := config.GetConfig()
	edgeHubConfig.WSConfig = config.WebSocketConfig{
		CertFilePath: CertFile,
		KeyFilePath:  KeyFile,
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockAdapter.EXPECT().Send(gomock.Any()).Return(nil).Times(1)
			mockAdapter.EXPECT().Send(gomock.Any()).Return(errors.New("Connection Refused")).Times(1)
			go tt.controller.keepalive()
			got := <-tt.controller.stopChan
			if got != struct{}{} {
				t.Errorf("TestKeepalive() StopChan = %v, want %v", got, struct{}{})
			}
		})
	}
}

//TestPubConnectInfo() checks the connection information sent to the required group
func TestPubConnectInfo(t *testing.T) {
	initMocks(t)
	testContext := context.GetContext(context.MsgCtxTypeChannel)
	tests := []struct {
		name        string
		controller  Controller
		isConnected bool
		content     string
	}{
		{"Cloud connected case", Controller{
			context:    testContext,
			stopChan:   make(chan struct{}),
			syncKeeper: make(map[string]chan model.Message),
		},
			true,
			connect.CloudConnected},
		{"Cloud disconnected case", Controller{
			context:    testContext,
			stopChan:   make(chan struct{}),
			syncKeeper: make(map[string]chan model.Message),
		},
			false,
			connect.CloudDisconnected},
	}
	for _, tt := range tests {
		modules := core.GetModules()
		for name, module := range modules {
			testContext.AddModule(name)
			testContext.AddModuleGroup(name, module.Group())
		}
		t.Run(tt.name, func(t *testing.T) {
			tt.controller.pubConnectInfo(tt.isConnected)
			t.Run("TestMessageContent", func(t *testing.T) {
				msg, err := testContext.Receive(module.TwinGroup)
				if err != nil {
					t.Errorf("Error in receiving message from twin group: %v", err)
				} else if msg.Content != tt.content {
					t.Errorf("TestPubConnectInfo() Content of message received in twin group : %v, want: %v", msg.Content, tt.content)
				}
			})
		})
	}
}

//TestPostUrlRequst() tests the request sent to the placement URL and its corresponding response
func TestPostUrlRequst(t *testing.T) {
	tests := []struct {
		name          string
		controller    Controller
		client        *http.Client
		want          string
		expectedError error
	}{
		{"post URL request with valid input ", Controller{
			config: &config.ControllerConfig{
				Protocol:     "websocket",
				PlacementURL: testServer.URL + "/proper_request",
				ProjectID:    "foo",
				NodeID:       "bar",
			},
		}, &http.Client{},
			"ws://127.0.0.1:20000/foo/bar/events", nil},

		{"post URL request with invalid input", Controller{
			config: &config.ControllerConfig{
				Protocol:     "websocket",
				PlacementURL: testServer.URL + "/bad_request",
				ProjectID:    "foo",
				NodeID:       "bar",
			},
		}, &http.Client{}, "", fmt.Errorf("bad request")},

		{"post URL request with wrong URL", Controller{
			config: &config.ControllerConfig{
				Protocol:     "websocket",
				PlacementURL: testServer.URL + "/wrong_url",
				ProjectID:    "foo",
				NodeID:       "bar",
			},
		}, &http.Client{}, "ws://127.0.0.1:20000/foo/bar/events", nil},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := tt.controller.postURLRequst(tt.client)
			if !reflect.DeepEqual(err, tt.expectedError) {
				t.Errorf("Controller.postUrlRequst() error = %v, expectedError %v", err, tt.expectedError)
				return
			}
			if got != tt.want {
				t.Errorf("Controller.postUrlRequst() = %v, want %v", got, tt.want)
			}
		})
	}
}

//TestGetCloudHubUrlWithoutPlacement() tests the procurement of the cloudHub URL when no placement server is present
func TestGetCloudHubUrlWithoutPlacement(t *testing.T) {
	if err := util.GenerateTestYaml(testEdgeConfigYaml{edgeHubConfigYaml{
		webSocketConfigYaml{
			URL: "wss://0.0.0.0:10000/foo/bar/events",
		},
		controllerConfigYaml{
			Placement: "false",
		},
	},
	}, getConfigDirectory()+"/conf", "edge"); err != nil {
		t.Error("Unable to generate test YAML file: ", err)
	}

	tests := []struct {
		name            string
		controller      Controller
		webSocketConfig config.WebSocketConfig
		want            string
		expectedError   error
	}{
		{"Get valid cloudhub URL: without placement server", Controller{
			config: &config.ControllerConfig{
				Protocol:     "websocket",
				PlacementURL: testServer.URL + "/proper_request",
				ProjectID:    "foo",
				NodeID:       "bar",
			},
		}, config.WebSocketConfig{
			CertFilePath: CertFile,
			KeyFilePath:  KeyFile,
		}, "wss://0.0.0.0:10000/foo/bar/events", nil,
		},
	}
	// time to let config be synced again
	time.Sleep(10 * time.Second)
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			edgeHubConfig := config.GetConfig()
			edgeHubConfig.WSConfig = tt.webSocketConfig
			got, err := tt.controller.getCloudHubURL()
			if !reflect.DeepEqual(err, tt.expectedError) {
				t.Errorf("Controller.getCloudHubUrl() error = %v, expectedError %v", err, tt.expectedError)
			}
			if got != tt.want {
				t.Errorf("Controller.getCloudHubUrl() = %v, want %v", got, tt.want)
			}

		})
	}
	restoreConfigBack()
	// time to let config be synced again
	time.Sleep(10 * time.Second)
}

//TestGetCloudHubUrlWithPlacement() tests the procurement of the cloudHub URL from the placement server
func TestGetCloudHubUrlWithPlacement(t *testing.T) {
	if err := util.GenerateTestYaml(testEdgeConfigYaml{edgeHubConfigYaml{
		webSocketConfigYaml{
			URL: "wss://0.0.0.0:10000/foo/bar/events",
		},
		controllerConfigYaml{
			Placement: "true",
		},
	},
	}, getConfigDirectory()+"/conf", "edge"); err != nil {
		t.Error("Unable to generate test YAML file: ", err)
	}

	tests := []struct {
		name            string
		controller      Controller
		webSocketConfig config.WebSocketConfig
		want            string
		expectedError   error
	}{
		{"Get valid cloudhub URL: with placement server", Controller{
			config: &config.ControllerConfig{
				Protocol:     "websocket",
				PlacementURL: testServer.URL + "/proper_request",
				ProjectID:    "foo",
				NodeID:       "bar",
			},
		}, config.WebSocketConfig{
			CertFilePath: CertFile,
			KeyFilePath:  KeyFile,
		}, "ws://127.0.0.1:20000/foo/bar/events", nil,
		},
		{"Invalid cloudhub URL: with placement server", Controller{
			config: &config.ControllerConfig{
				Protocol:     "websocket",
				PlacementURL: testServer.URL + "/bad_request",
				ProjectID:    "foo",
				NodeID:       "bar",
			},
		}, config.WebSocketConfig{
			CertFilePath: CertFile,
			KeyFilePath:  KeyFile,
		}, "", fmt.Errorf("failed to new https client for placement, error: bad request"),
		},
		{"Wrong certificate paths: with placement server", Controller{
			config: &config.ControllerConfig{
				Protocol:     "websocket",
				PlacementURL: testServer.URL + "/proper_request",
				ProjectID:    "foo",
				NodeID:       "bar",
			},
		}, config.WebSocketConfig{
			CertFilePath: "/wrong_path/edge.crt",
			KeyFilePath:  "/wrong_path/edge.key",
		}, "", fmt.Errorf("failed to new https client for placement, error: open /wrong_path/edge.crt: no such file or directory"),
		},
	}
	// time to let config be synced again
	time.Sleep(10 * time.Second)
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			edgeHubConfig := config.GetConfig()
			edgeHubConfig.WSConfig = tt.webSocketConfig
			got, err := tt.controller.getCloudHubURL()
			if !reflect.DeepEqual(err, tt.expectedError) {
				t.Errorf("Controller.getCloudHubUrl() error = %v, expectedError %v", err, tt.expectedError)
			}
			if got != tt.want {
				t.Errorf("Controller.getCloudHubUrl() = %v, want %v", got, tt.want)
			}
		})
	}
	restoreConfigBack()
	// time to let config be synced again
	time.Sleep(10 * time.Second)
	defer func() {
		err := os.RemoveAll("/tmp/kubeedge/")
		if err != nil {
			fmt.Println("Error in Removing temporary files created for testing: ", err)
		}
	}()
}
