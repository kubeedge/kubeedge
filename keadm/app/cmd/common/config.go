/*
Copyright 2019 The Kubeedge Authors.

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

	"github.com/kubeedge/kubeedge/common/constants"
	"gopkg.in/yaml.v2"
)

//Write2File writes data into a file in path
func Write2File(path string, data interface{}) error {
	d, err := yaml.Marshal(&data)
	if err != nil {
		return err
	}

	if err = ioutil.WriteFile(path, d, 0666); err != nil {
		return err
	}
	return nil
}

//WriteControllerYamlFile writes controller.yaml for cloud component
func WriteControllerYamlFile(path, kubeConfig string) error {
	controllerData := ControllerYaml{Controller: CloudControllerSt{Kube: KubeEdgeControllerConfig{Master: "http://localhost:8080", Namespace: constants.DefaultKubeNamespace,
		ContentType: constants.DefaultKubeContentType,
		QPS:         constants.DefaultKubeQPS, Burst: constants.DefaultKubeBurst, NodeUpdateFrequency: constants.DefaultKubeUpdateNodeFrequency * time.Second,
		KubeConfig: kubeConfig}},
		CloudHub: CloudHubSt{IPAddress: "0.0.0.0", Port: 10000, CA: "/etc/kubeedge/ca/rootCA.crt", Cert: "/etc/kubeedge/certs/edge.crt",
			Key: "/etc/kubeedge/certs/edge.key", KeepAliveInterval: 30, WriteTimeout: 30, NodeLimit: 10},
		DeviceController: DeviceControllerSt{Kube: KubeEdgeControllerConfig{Master: "http://localhost:8080", Namespace: constants.DefaultKubeNamespace,
			ContentType: constants.DefaultKubeContentType,
			QPS:         constants.DefaultKubeQPS, Burst: constants.DefaultKubeBurst, KubeConfig: ""}},
	}
	if err := Write2File(path, controllerData); err != nil {
		return err
	}
	return nil
}

//WriteCloudModulesYamlFile writes modules.yaml for cloud component
func WriteCloudModulesYamlFile(path string) error {
	modulesData := ModulesYaml{Modules: ModulesSt{Enabled: []string{"devicecontroller", "controller", "cloudhub"}}}
	if err := Write2File(path, modulesData); err != nil {
		return err
	}
	return nil
}

//WriteCloudLoggingYamlFile writes logging yaml for cloud component
func WriteCloudLoggingYamlFile(path string) error {
	loggingData := LoggingYaml{LoggerLevel: "INFO", EnableRsysLog: false, LogFormatText: true, Writers: []string{"file", "stdout"}, LoggerFile: "edgecontroller.log"}
	if err := Write2File(path, loggingData); err != nil {
		return err
	}
	return nil
}

//WriteEdgeLoggingYamlFile writes logging yaml for edge component
func WriteEdgeLoggingYamlFile(path string) error {
	loggingData := LoggingYaml{LoggerLevel: "DEBUG", EnableRsysLog: false, LogFormatText: true, Writers: []string{"stdout"}}
	if err := Write2File(path, loggingData); err != nil {
		return err
	}
	return nil
}

//WriteEdgeModulesYamlFile writes modules.yaml for edge component
func WriteEdgeModulesYamlFile(path string) error {
	modulesData := ModulesYaml{Modules: ModulesSt{Enabled: []string{"eventbus", "servicebus", "websocket", "metaManager", "edged", "twin", "edgemesh"}}}
	if err := Write2File(path, modulesData); err != nil {
		return err
	}
	return nil
}

//WriteEdgeYamlFile write conf/edge.yaml for edge component
func WriteEdgeYamlFile(path string, modifiedEdgeYaml *EdgeYamlSt) error {

	edgeID := "fb4ebb70-2783-42b8-b3ef-63e2fd6d242e"
	url := fmt.Sprintf("wss://0.0.0.0:10000/%s/fb4ebb70-2783-42b8-b3ef-63e2fd6d242e/events", DefaultProjectID)
	version := "2.0.0"
	runtimeType := "docker"

	if nil != modifiedEdgeYaml {
		if "" != modifiedEdgeYaml.EdgeHub.WebSocket.URL {
			url = modifiedEdgeYaml.EdgeHub.WebSocket.URL
			edgeID = strings.Split(modifiedEdgeYaml.EdgeHub.WebSocket.URL, "/")[4]
		}
		if "" != modifiedEdgeYaml.EdgeD.Version {
			version = modifiedEdgeYaml.EdgeD.Version
		}
		if "" != modifiedEdgeYaml.EdgeD.RuntimeType {
			runtimeType = modifiedEdgeYaml.EdgeD.RuntimeType
		}
	}

	edgeData := EdgeYamlSt{MQTT: MQTTConfig{Server: "tcp://127.0.0.1:1883", InternalServer: "tcp://127.0.0.1:1884", Mode: MQTTInternalMode, QOS: MQTTQoSAtMostOnce,
		Retain: false, SessionQueueSize: 100},
		EdgeHub: EdgeHubSt{WebSocket: WebSocketSt{URL: url, CertFile: "/etc/kubeedge/certs/edge.crt", KeyFile: "/etc/kubeedge/certs/edge.key",
			HandshakeTimeout: 30, WriteDeadline: 15, ReadDeadline: 15},
			Controller: ControllerSt{Placement: false, Heartbeat: 15, RefreshAKSKInterval: 10, AuthInfoFilesPath: "/var/IEF/secret",
				PlacementURL: "https://10.154.193.32:7444/v1/placement_external/message_queue", ProjectID: DefaultProjectID,
				NodeID: edgeID}},
		EdgeD: EdgeDSt{RegisterNodeNamespace: "default", HostnameOverride: edgeID, InterfaceName: "eth0",
			NodeStatusUpdateFrequency: 10, DevicePluginEnabled: false, GPUPluginEnabled: false, ImageGCHighThreshold: 80, ImageGCLowThreshold: 40,
			MaximumDeadContainersPerContainer: 1, DockerAddress: "unix:///var/run/docker.sock", Version: version, RuntimeType: runtimeType, RuntimeEndpoint: "/var/run/containerd/containerd.sock", ImageEndpoint: "/var/run/containerd/containerd.sock", RequestTimeout: 2, PodSandboxImage: "k8s.gcr.io/pause"},
		Mesh: Mesh{LB: LoadBalance{StrategyName: "RoundRobin"}},
	}
	if err := Write2File(path, edgeData); err != nil {
		return err
	}
	return nil
}
