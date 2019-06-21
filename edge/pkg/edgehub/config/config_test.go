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

package config

import (
	"net/http"
	"reflect"

	"testing"
	"time"

	bhConfig "github.com/kubeedge/beehive/pkg/common/config"
	bhUtil "github.com/kubeedge/beehive/pkg/common/util"
	"github.com/kubeedge/kubeedge/edge/pkg/common/util"
)

const (
	defaultProjectID = "e632aba927ea4ac2b575ec1603d56f10"
)

//testYamlGenerator is a structure which is used to generate the test YAML file to test Edgehub config components
type testYamlGenerator struct {
	Edgehub edgeHubConfigYaml `yaml:"edgehub"`
}

//testHeaderYamlGenerator is a structure which is used to generate the test YAML file to test SystemInfo config components
type testHeaderYamlGenerator struct {
	SystemInfo extendHeaderConfigYaml `yaml:"systeminfo"`
}

//webSocketConfigYaml is a structure which is used to generate the test YAML file to test WebSocket config components
type webSocketConfigYaml struct {
	URL              string `yaml:"url,omitempty"`
	CertFilePath     string `yaml:"certfile,omitempty"`
	KeyFilePath      string `yaml:"keyfile,omitempty"`
	HandshakeTimeout string `yaml:"handshake-timeout,omitempty"`
	ReadDeadline     string `yaml:"read-deadline,omitempty"`
	WriteDeadline    string `yaml:"write-deadline,omitempty"`
}

//quicConfigYaml is a structure which is used to generate the test YAML file to test WebSocket config components
type quicConfigYaml struct {
	URL              string `yaml:"url,omitempty"`
	CaFilePath       string `yaml:"cafile,omitempty"`
	CertFilePath     string `yaml:"certfile,omitempty"`
	KeyFilePath      string `yaml:"keyfile,omitempty"`
	HandshakeTimeout string `yaml:"handshake-timeout,omitempty"`
	ReadDeadline     string `yaml:"read-deadline,omitempty"`
	WriteDeadline    string `yaml:"write-deadline,omitempty"`
}

//extendHeaderConfigYaml is a structure which is used to load the architecture and DockerRootDIr config to generate the test YAML file
type extendHeaderConfigYaml struct {
	Arch          string `yaml:"architecture,omitempty"`
	DockerRootDir string `yaml:"docker_root_dir,omitempty"`
}

//controllerConfigYaml is a structure which is used to generate the test YAML file to test controller config components
type controllerConfigYaml struct {
	Protocol        string `yaml:"protocol"`
	HeartbeatPeroid string `yaml:"heartbeat,omitempty"`
	RefreshInterval string `yaml:"refresh-ak-sk-interval,omitempty"`
	CloudhubURL     string `yaml:"cloud-hub-url"`
	AuthInfosPath   string `yaml:"auth-info-files-path,omitempty"`
	PlacementURL    string `yaml:"placement-url,omitempty"`
	ProjectID       string `yaml:"project-id,omitempty"`
	NodeID          string `yaml:"node-id,omitempty"`
}

//edgeHubConfigYaml is a structure which is used to load the websocket and controller config to generate the test YAML file
type edgeHubConfigYaml struct {
	WSConfig   webSocketConfigYaml  `yaml:"websocket"`
	QuicConfig quicConfigYaml       `yaml:"quic"`
	CtrConfig  controllerConfigYaml `yaml:"controller"`
}

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

//TestGetConfig  function loads the testing config file  tests whether the edgeHubConfig variable is loaded correctly with the values from the config file
func TestGetConfig(t *testing.T) {
	//Testcases
	tests := []struct {
		name string
		test testYamlGenerator
		want *EdgeHubConfig
	}{
		//Positive Testcase with All values Provided
		{"TestGetConfig: Proper Input",
			testYamlGenerator{
				edgeHubConfigYaml{
					webSocketConfigYaml{
						URL:              "ws://127.0.0.1:20000/fake_group_id/events",
						CertFilePath:     "/tmp/edge.crt",
						KeyFilePath:      "/tmp/edge.key",
						HandshakeTimeout: "500",
						WriteDeadline:    "100",
						ReadDeadline:     "100",
					},
					quicConfigYaml{
						URL:              "127.0.0.1:10001",
						CaFilePath:       "/tmp/rootCA.crt",
						CertFilePath:     "/tmp/edge.crt",
						KeyFilePath:      "/tmp/edge.key",
						HandshakeTimeout: "500",
						WriteDeadline:    "100",
						ReadDeadline:     "100",
					},
					controllerConfigYaml{
						Protocol:        "websocket",
						HeartbeatPeroid: "150",
						RefreshInterval: "15",
						AuthInfosPath:   "/var/IEF/secret",
						PlacementURL:    "https://10.154.193.32:7444/v1/placement_external/message_queue",
						ProjectID:       defaultProjectID,
						NodeID:          "fb4ebb70-2783-42b8-b3ef-63e2fd6d242e",
					},
				},
			},
			&EdgeHubConfig{
				WSConfig: WebSocketConfig{
					URL:              "ws://127.0.0.1:20000/fake_group_id/events",
					CertFilePath:     "/tmp/edge.crt",
					KeyFilePath:      "/tmp/edge.key",
					HandshakeTimeout: 500 * time.Second,
					WriteDeadline:    100 * time.Second,
					ReadDeadline:     100 * time.Second,
					ExtendHeader:     http.Header{},
				},
				QcConfig: QuicConfig{
					URL:              "127.0.0.1:10001",
					CaFilePath:       "/tmp/rootCA.crt",
					CertFilePath:     "/tmp/edge.crt",
					KeyFilePath:      "/tmp/edge.key",
					HandshakeTimeout: 500 * time.Second,
					WriteDeadline:    100 * time.Second,
					ReadDeadline:     100 * time.Second,
				},
				CtrConfig: ControllerConfig{
					Protocol:        "websocket",
					HeartbeatPeriod: 150 * time.Second,
					RefreshInterval: 15 * time.Minute,
					AuthInfosPath:   "/var/IEF/secret",
					PlacementURL:    "https://10.154.193.32:7444/v1/placement_external/message_queue",
					ProjectID:       defaultProjectID,
					NodeID:          "fb4ebb70-2783-42b8-b3ef-63e2fd6d242e",
				},
			}},

		//Positive Testcase which uses all default values provided in code
		{"TestGetConfig: Use all default options",
			testYamlGenerator{
				edgeHubConfigYaml{
					webSocketConfigYaml{
						URL:          "ws://127.0.0.1:20000/fake_group_id/events",
						CertFilePath: "/tmp/edge.crt",
						KeyFilePath:  "/tmp/edge.key",
					},
					quicConfigYaml{
						URL:          "127.0.0.1:10001",
						CaFilePath:   "/tmp/rootCA.crt",
						CertFilePath: "/tmp/edge.crt",
						KeyFilePath:  "/tmp/edge.key",
					},
					controllerConfigYaml{
						Protocol:     "websocket",
						PlacementURL: "https://10.154.193.32:7444/v1/placement_external/message_queue",
						ProjectID:    defaultProjectID,
						NodeID:       "fb4ebb70-2783-42b8-b3ef-63e2fd6d242e",
					},
				},
			},
			&EdgeHubConfig{
				WSConfig: WebSocketConfig{
					URL:              "ws://127.0.0.1:20000/fake_group_id/events",
					CertFilePath:     "/tmp/edge.crt",
					KeyFilePath:      "/tmp/edge.key",
					HandshakeTimeout: 60 * time.Second,
					WriteDeadline:    15 * time.Second,
					ReadDeadline:     15 * time.Second,
					ExtendHeader:     http.Header{},
				},
				QcConfig: QuicConfig{
					URL:              "127.0.0.1:10001",
					CaFilePath:       "/tmp/rootCA.crt",
					CertFilePath:     "/tmp/edge.crt",
					KeyFilePath:      "/tmp/edge.key",
					HandshakeTimeout: 60 * time.Second,
					WriteDeadline:    15 * time.Second,
					ReadDeadline:     15 * time.Second,
				},
				CtrConfig: ControllerConfig{
					Protocol:        "websocket",
					AuthInfosPath:   "/var/IEF/secret",
					HeartbeatPeriod: 15 * time.Second,
					RefreshInterval: 10 * time.Minute,
					PlacementURL:    "https://10.154.193.32:7444/v1/placement_external/message_queue",
					ProjectID:       defaultProjectID,
					NodeID:          "fb4ebb70-2783-42b8-b3ef-63e2fd6d242e",
				},
			}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := util.GenerateTestYaml(tt.test, getConfigDirectory()+"/conf", "edge")
			if err != nil {
				t.Error("Unable to generate test YAML file: ", err)
			}
			// time to let config be synced again
			time.Sleep(10 * time.Second)
			err = getWebSocketConfig()
			if err != nil {
				t.Errorf("getWebSocketConfig() returns an error: %v", err)
			}
			err = getQuicConfig()
			if err != nil {
				t.Errorf("getQuicConfig() returns an error: %v", err)
			}
			err = getControllerConfig()
			if err != nil {
				t.Errorf("getControllerConfig() returns an error: %v", err)
			}
			if got := GetConfig(); !reflect.DeepEqual(*got, *tt.want) {
				t.Errorf("GetConfig() = %v, want %v", got, tt.want)
			}
		})
	}
}

//Test_getWebSocketConfig function loads the testing config file tests whether the websocket configurations are loaded correctly with the values from the config file
func Test_getWebSocketConfig(t *testing.T) {
	//Testcases
	tests := []struct {
		name    string
		test    testYamlGenerator
		wantErr bool
	}{
		//Positive Testcase with all values correctly specified
		{"Test_getWebSocketConfig1: Proper Input",
			testYamlGenerator{
				edgeHubConfigYaml{
					webSocketConfigYaml{
						URL:              "ws://127.0.0.1:20000/fake_group_id/events",
						CertFilePath:     "/tmp/edge.crt",
						KeyFilePath:      "/tmp/edge.key",
						HandshakeTimeout: "500",
						WriteDeadline:    "100",
						ReadDeadline:     "100",
					},
					quicConfigYaml{},
					controllerConfigYaml{},
				},
			},
			false},

		//Positive Testcase which uses default input for Handshake-timeout, Write Dealine & Read Deadline
		{"Test_getWebSocketConfig2: Use Default input for Handshake-timeout, Write Dealine & Read Deadline ",
			testYamlGenerator{
				edgeHubConfigYaml{
					webSocketConfigYaml{
						URL:          "ws://127.0.0.1:20000/fake_group_id/events",
						CertFilePath: "/tmp/edge.crt",
						KeyFilePath:  "/tmp/edge.key",
					},
					quicConfigYaml{},
					controllerConfigYaml{},
				},
			},
			false},

		//Positive Testcase with no URL specified
		{"Test_getWebSocketConfig3: No URL provided ",
			testYamlGenerator{
				edgeHubConfigYaml{
					webSocketConfigYaml{
						CertFilePath:     "/tmp/edge.crt",
						KeyFilePath:      "/tmp/edge.key",
						HandshakeTimeout: "500",
						WriteDeadline:    "100",
						ReadDeadline:     "100",
					},
					quicConfigYaml{},
					controllerConfigYaml{},
				},
			},
			false},

		//Negative  Testcase with no CertFile  specified
		{"Test_getWebSocketConfig4: No Cert File provided ",
			testYamlGenerator{
				edgeHubConfigYaml{
					webSocketConfigYaml{
						URL:              "ws://127.0.0.1:20000/fake_group_id/events",
						KeyFilePath:      "/tmp/edge.key",
						HandshakeTimeout: "500",
						WriteDeadline:    "100",
						ReadDeadline:     "100",
					},
					quicConfigYaml{},
					controllerConfigYaml{},
				},
			},
			true},

		//Negative  Testcase with no KeyFile  specified
		{"Test_getWebSocketConfig5: No Key File provided",
			testYamlGenerator{
				edgeHubConfigYaml{
					webSocketConfigYaml{
						URL:              "ws://127.0.0.1:20000/fake_group_id/events",
						CertFilePath:     "/tmp/edge.crt",
						HandshakeTimeout: "500",
						WriteDeadline:    "100",
						ReadDeadline:     "100",
					},
					quicConfigYaml{},
					controllerConfigYaml{},
				},
			},
			true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := util.GenerateTestYaml(tt.test, getConfigDirectory()+"/conf", "edge")
			if err != nil {
				t.Error("Unable to generate test YAML file: ", err)
			}
			// time to let config be synced again
			time.Sleep(10 * time.Second)
			if err := getWebSocketConfig(); (err != nil) != tt.wantErr {
				t.Errorf("getWebSocketConfig() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

//Test_getControllerConfig function loads the testing config file tests whether the controller configurations are loaded correctly with the values from the config file
func Test_getControllerConfig(t *testing.T) {
	//Testcases
	tests := []struct {
		name    string
		test    testYamlGenerator
		wantErr bool
	}{
		//Positive Testcase with all values correctly specified
		{"Test_getControllerConfig1: Proper input ",
			testYamlGenerator{
				edgeHubConfigYaml{
					webSocketConfigYaml{},
					quicConfigYaml{},
					controllerConfigYaml{
						Protocol:        "websocket",
						HeartbeatPeroid: "150",
						RefreshInterval: "15",
						AuthInfosPath:   "/var/IEF/secret",
						PlacementURL:    "https://10.154.193.32:7444/v1/placement_external/message_queue",
						ProjectID:       defaultProjectID,
						NodeID:          "fb4ebb70-2783-42b8-b3ef-63e2fd6d242e",
					},
				},
			},
			false},

		//Positive Testcase which uses default input for HeartbeatPeroid, RefreshInterval & AuthInfosPath
		{"Test_getControllerConfig2: 	Use default values for HeartbeatPeroid, RefreshInterval &  AuthInfosPath  ",
			testYamlGenerator{
				edgeHubConfigYaml{
					webSocketConfigYaml{},
					quicConfigYaml{},
					controllerConfigYaml{
						Protocol:     "websocket",
						PlacementURL: "https://10.154.193.32:7444/v1/placement_external/message_queue",
						ProjectID:    defaultProjectID,
						NodeID:       "fb4ebb70-2783-42b8-b3ef-63e2fd6d242e",
					},
				},
			},
			false},

		//Negative Testcase with no placementURL  provided
		{"Test_getControllerConfig3: No placementURL  provided ",
			testYamlGenerator{
				edgeHubConfigYaml{
					webSocketConfigYaml{},
					quicConfigYaml{},
					controllerConfigYaml{
						Protocol:        "websocket",
						HeartbeatPeroid: "150",
						RefreshInterval: "15",
						AuthInfosPath:   "/var/IEF/secret",
						ProjectID:       defaultProjectID,
						NodeID:          "fb4ebb70-2783-42b8-b3ef-63e2fd6d242e",
					},
				},
			},
			true},

		//Negative Testcase with no projectID  provided
		{"Test_getControllerConfig4: No projectID provided ",
			testYamlGenerator{
				edgeHubConfigYaml{
					webSocketConfigYaml{},
					quicConfigYaml{},
					controllerConfigYaml{
						Protocol:        "websocket",
						HeartbeatPeroid: "150",
						RefreshInterval: "15",
						AuthInfosPath:   "/var/IEF/secret",
						PlacementURL:    "https://10.154.193.32:7444/v1/placement_external/message_queue",
						NodeID:          "fb4ebb70-2783-42b8-b3ef-63e2fd6d242e",
					},
				},
			},
			true},

		//Negative Testcase with no node ID provided
		{"Test_getControllerConfig5: No node ID provided ",
			testYamlGenerator{
				edgeHubConfigYaml{
					webSocketConfigYaml{},
					quicConfigYaml{},
					controllerConfigYaml{
						Protocol:        "websocket",
						HeartbeatPeroid: "150",
						RefreshInterval: "15",
						AuthInfosPath:   "/var/IEF/secret",
						PlacementURL:    "https://10.154.193.32:7444/v1/placement_external/message_queue",
						ProjectID:       defaultProjectID,
					},
				},
			},
			true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := util.GenerateTestYaml(tt.test, getConfigDirectory()+"/conf", "edge")
			if err != nil {
				t.Error("Unable to generate test YAML file: ", err)
			}
			// time to let config be synced again
			time.Sleep(10 * time.Second)
			if err := getControllerConfig(); (err != nil) != tt.wantErr {
				t.Errorf("getControllerConfig() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

//Test_getExtendHeader function loads the test config file and tests whether it returns the correct http headers as specified in the config file
func Test_getExtendHeader(t *testing.T) {
	//Testcases
	tests := []struct {
		name string
		test testHeaderYamlGenerator
		want http.Header
	}{
		//Positive Testcase with proper input provided
		{"Test_getExtendHeader1: Proper Input ",
			testHeaderYamlGenerator{
				extendHeaderConfigYaml{
					Arch:          "x86",
					DockerRootDir: "/usr",
				}},
			http.Header{
				"Arch":          []string{"x86"},
				"Dockerrootdir": []string{"/usr"},
			}},

		//Positive Testcase with only architecture input  provided
		{"Test_getExtendHeader2: Only Arch provided",
			testHeaderYamlGenerator{
				extendHeaderConfigYaml{
					Arch: "x86",
				}},
			http.Header{
				"Arch": []string{"x86"},
			}},

		//Positive Testcase with only docker root directory  input  provided
		{"Test_getExtendHeader3: Only Docker root directory provided",
			testHeaderYamlGenerator{
				extendHeaderConfigYaml{
					DockerRootDir: "/usr",
				}},
			http.Header{
				"Dockerrootdir": []string{"/usr"},
			}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := util.GenerateTestYaml(tt.test, getConfigDirectory()+"/conf", "edge")
			if err != nil {
				t.Error("Unable to generate test YAML file: ", err)
			}
			// time to let config be synced again
			time.Sleep(10 * time.Second)
			if got := getExtendHeader(); !reflect.DeepEqual(got, tt.want) {
				t.Errorf("getExtendHeader() = %v, want %v", got, tt.want)
			}
		})
	}

	restoreConfigBack()
}
