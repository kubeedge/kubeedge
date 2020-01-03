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

package v1alpha1

import (
	"path"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	deviceconstants "github.com/kubeedge/kubeedge/cloud/pkg/devicecontroller/constants"
	"github.com/kubeedge/kubeedge/common/constants"
	metaconfig "github.com/kubeedge/kubeedge/pkg/apis/meta/v1alpha1"
)

// NewDefaultCloudCoreConfig return a default CloudCoreConfig object
func NewDefaultCloudCoreConfig() *CloudCoreConfig {
	return &CloudCoreConfig{
		TypeMeta: metav1.TypeMeta{
			Kind:       Kind,
			APIVersion: path.Join(GroupName, APIVersion),
		},
		KubeAPIConfig: KubeAPIConfig{
			Master:      "",
			ContentType: constants.DefaultKubeContentType,
			QPS:         constants.DefaultKubeQPS,
			Burst:       constants.DefaultKubeBurst,
			KubeConfig:  constants.DefaultKubeConfig,
		},
		Modules: Modules{
			CloudHub: CloudHub{
				Enable:            true,
				KeepaliveInterval: 30,
				NodeLimit:         10,
				TLSCAFile:         constants.DefaultCAFile,
				TLSCertFile:       constants.DefaultCertFile,
				TLSPrivateKeyFile: constants.DefaultKeyFile,
				WriteTimeout:      30,
				Quic: CloudHubQuic{
					Enable:             false,
					Address:            "0.0.0.0",
					Port:               10001,
					MaxIncomingStreams: 10000,
				},
				UnixSocket: CloudHubUnixSocket{
					Enable:  true,
					Address: "unix:///var/lib/kubeedge/kubeedge.sock",
				},
				WebSocket: CloudHubWebSocket{
					Enable:  true,
					Port:    10000,
					Address: "0.0.0.0",
				},
			},
			EdgeController: EdgeController{
				Enable:              true,
				NodeUpdateFrequency: 10,
				Buffer: EdgeControllerBuffer{
					UpdatePodStatus:            constants.DefaultUpdatePodStatusBuffer,
					UpdateNodeStatus:           constants.DefaultUpdateNodeStatusBuffer,
					QueryConfigmap:             constants.DefaultQueryConfigMapBuffer,
					QuerySecret:                constants.DefaultQuerySecretBuffer,
					QueryService:               constants.DefaultQueryServiceBuffer,
					QueryEndpoints:             constants.DefaultQueryEndpointsBuffer,
					PodEvent:                   constants.DefaultPodEventBuffer,
					ConfigmapEvent:             constants.DefaultConfigMapEventBuffer,
					SecretEvent:                constants.DefaultSecretEventBuffer,
					ServiceEvent:               constants.DefaultServiceEventBuffer,
					EndpointsEvent:             constants.DefaultEndpointsEventBuffer,
					QueryPersistentvolume:      constants.DefaultQueryPersistentVolumeBuffer,
					QueryPersistentvolumeclaim: constants.DefaultQueryPersistentVolumeClaimBuffer,
					QueryVolumeattachment:      constants.DefaultQueryVolumeAttachmentBuffer,
					QueryNode:                  constants.DefaultQueryNodeBuffer,
					UpdateNode:                 constants.DefaultUpdateNodeBuffer,
				},
				Context: EdgeControllerContext{
					SendModule:     metaconfig.ModuleNameCloudHub,
					ReceiveModule:  metaconfig.ModuleNameEdgeController,
					ResponseModule: metaconfig.ModuleNameCloudHub,
				},
				Load: EdgeControllerLoad{
					UpdatePodStatusWorkers:            constants.DefaultUpdatePodStatusWorkers,
					UpdateNodeStatusWorkers:           constants.DefaultUpdateNodeStatusWorkers,
					QueryConfigmapWorkers:             constants.DefaultQueryConfigMapWorkers,
					QuerySecretWorkers:                constants.DefaultQuerySecretWorkers,
					QueryServiceWorkers:               constants.DefaultQueryServiceWorkers,
					QueryEndpointsWorkers:             constants.DefaultQueryEndpointsWorkers,
					QueryPersistentvolumeWorkers:      constants.DefaultQueryPersistentVolumeWorkers,
					QueryPersistentvolumeclaimWorkers: constants.DefaultQueryPersistentVolumeClaimWorkers,
					QueryVolumeattachmentWorkers:      constants.DefaultQueryVolumeAttachmentWorkers,
					QueryNodeWorkers:                  constants.DefaultQueryNodeWorkers,
					UpdateNodeWorkers:                 constants.DefaultUpdateNodeWorkers,
				},
			},
			DeviceController: DeviceController{
				Enable: true,
				Context: DeviceControllerContext{
					SendModule:     metaconfig.ModuleNameCloudHub,
					ReceiveModule:  metaconfig.ModuleNameDeviceController,
					ResponseModule: metaconfig.ModuleNameCloudHub,
				},
				Buffer: DeviceControllerBuffer{
					UpdateDeviceStatus: deviceconstants.DefaultUpdateDeviceStatusBuffer,
					DeviceEvent:        deviceconstants.DefaultDeviceEventBuffer,
					DeviceModelEvent:   deviceconstants.DefaultDeviceModelEventBuffer,
				},
				Load: DeviceControllerLoad{
					UpdateDeviceStatusWorkers: deviceconstants.DefaultUpdateDeviceStatusWorkers,
				},
			},
		},
	}
}
