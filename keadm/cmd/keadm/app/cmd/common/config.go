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

package common

import (
	"fmt"
	"io/ioutil"
	"strings"
	"time"

	"sigs.k8s.io/yaml"

	"github.com/kubeedge/kubeedge/common/constants"
)

//Write2File writes data into a file in path
func Write2File(path string, data interface{}) error {
	d, err := yaml.Marshal(data)
	if err != nil {
		return err
	}

	return ioutil.WriteFile(path, d, 0666)
}

//WriteControllerYamlFile writes controller.yaml for cloud component
func WriteControllerYamlFile(path, kubeConfig string) error {
	controllerData := CloudCoreYaml{
		EdgeController: EdgeControllerSt{
			Kube: ControllerKubeConfig{
				Master:              "http://localhost:8080",
				Namespace:           constants.DefaultKubeNamespace,
				ContentType:         constants.DefaultKubeContentType,
				QPS:                 constants.DefaultKubeQPS,
				Burst:               constants.DefaultKubeBurst,
				NodeUpdateFrequency: constants.DefaultKubeUpdateNodeFrequency * time.Second,
				KubeConfig:          kubeConfig,
			},
		},
		CloudHub: CloudHubSt{
			IPAddress:         "0.0.0.0",
			Port:              10000,
			CA:                "/etc/kubeedge/ca/rootCA.crt",
			Cert:              "/etc/kubeedge/certs/edge.crt",
			Key:               "/etc/kubeedge/certs/edge.key",
			KeepAliveInterval: 30,
			WriteTimeout:      30,
			NodeLimit:         10,
		},
		DeviceController: DeviceControllerSt{
			Kube: ControllerKubeConfig{
				Master:      "http://localhost:8080",
				Namespace:   constants.DefaultKubeNamespace,
				ContentType: constants.DefaultKubeContentType,
				QPS:         constants.DefaultKubeQPS,
				Burst:       constants.DefaultKubeBurst,
				KubeConfig:  ""},
		},
	}
	return Write2File(path, controllerData)
}

//WriteCloudModulesYamlFile writes modules.yaml for cloud component
func WriteCloudModulesYamlFile(path string) error {
	modulesData := ModulesYaml{
		Modules: ModulesSt{
			Enabled: []string{
				"devicecontroller",
				"controller",
				"cloudhub",
			},
		},
	}
	return Write2File(path, modulesData)
}

//WriteEdgeModulesYamlFile writes modules.yaml for edge component
func WriteEdgeModulesYamlFile(path string) error {
	modulesData := ModulesYaml{
		Modules: ModulesSt{
			Enabled: []string{
				"eventbus",
				"servicebus",
				"websocket",
				"metaManager",
				"edged",
				"twin",
				"edgemesh",
			},
		},
	}
	return Write2File(path, modulesData)
}

//WriteEdgeYamlFile write conf/edge.yaml for edge component
func WriteEdgeYamlFile(path string, modifiedEdgeYaml *EdgeYamlSt) error {
	edgeID := "edge-node"
	url := fmt.Sprintf("wss://0.0.0.0:10000/%s/edge-node/events", DefaultProjectID)
	runtimeType := "docker"

	if nil != modifiedEdgeYaml {
		if "" != modifiedEdgeYaml.EdgeHub.WebSocket.URL {
			url = modifiedEdgeYaml.EdgeHub.WebSocket.URL
			edgeID = strings.Split(modifiedEdgeYaml.EdgeHub.WebSocket.URL, "/")[4]
		}
		if "" != modifiedEdgeYaml.EdgeD.RuntimeType {
			runtimeType = modifiedEdgeYaml.EdgeD.RuntimeType
		}
	}

	edgeData := EdgeYamlSt{
		MQTT: MQTTConfig{
			Server:           "tcp://127.0.0.1:1883",
			InternalServer:   "tcp://127.0.0.1:1884",
			Mode:             MQTTInternalMode,
			QOS:              MQTTQoSAtMostOnce,
			Retain:           false,
			SessionQueueSize: 100,
		},
		EdgeHub: EdgeHubSt{
			WebSocket: WebSocketSt{
				URL:              url,
				CertFile:         "/etc/kubeedge/certs/edge.crt",
				KeyFile:          "/etc/kubeedge/certs/edge.key",
				HandshakeTimeout: 30,
				WriteDeadline:    15,
				ReadDeadline:     15,
			},
			Controller: ControllerSt{
				Heartbeat: 15,
				ProjectID: DefaultProjectID,
				NodeID:    edgeID,
			},
		},
		EdgeD: EdgeDSt{
			RegisterNodeNamespace:             "default",
			HostnameOverride:                  edgeID,
			NodeStatusUpdateFrequency:         10,
			ConcurrentConsumers:               5,
			DevicePluginEnabled:               false,
			GPUPluginEnabled:                  false,
			ImageGCHighThreshold:              80,
			ImageGCLowThreshold:               40,
			MaximumDeadContainersPerContainer: 1,
			DockerAddress:                     "unix:///var/run/docker.sock",
			RuntimeType:                       runtimeType,
			RuntimeEndpoint:                   "/var/run/containerd/containerd.sock",
			ImageEndpoint:                     "/var/run/containerd/containerd.sock",
			RequestTimeout:                    2,
			PodSandboxImage:                   "k8s.gcr.io/pause",
		},
		Mesh: Mesh{
			LB: LoadBalance{
				StrategyName: "RoundRobin",
			},
		},
	}
	return Write2File(path, edgeData)
}
