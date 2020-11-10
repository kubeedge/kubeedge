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
	"time"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	utilnet "k8s.io/apimachinery/pkg/util/net"
	componentbaseconfig "k8s.io/component-base/config"

	"github.com/kubeedge/kubeedge/common/constants"
	metaconfig "github.com/kubeedge/kubeedge/pkg/apis/componentconfig/meta/v1alpha1"
)

// NewDefaultCloudCoreConfig returns a full CloudCoreConfig object
func NewDefaultCloudCoreConfig() *CloudCoreConfig {
	advertiseAddress, _ := utilnet.ChooseHostInterface()

	c := &CloudCoreConfig{
		TypeMeta: metav1.TypeMeta{
			Kind:       Kind,
			APIVersion: path.Join(GroupName, APIVersion),
		},
		KubeAPIConfig: &KubeAPIConfig{
			Master:      "",
			ContentType: constants.DefaultKubeContentType,
			QPS:         constants.DefaultKubeQPS,
			Burst:       constants.DefaultKubeBurst,
			KubeConfig:  constants.DefaultKubeConfig,
		},
		Modules: &Modules{
			CloudHub: &CloudHub{
				Enable:                  true,
				KeepaliveInterval:       30,
				NodeLimit:               1000,
				TLSCAFile:               constants.DefaultCAFile,
				TLSCAKeyFile:            constants.DefaultCAKeyFile,
				TLSCertFile:             constants.DefaultCertFile,
				TLSPrivateKeyFile:       constants.DefaultKeyFile,
				WriteTimeout:            30,
				AdvertiseAddress:        []string{advertiseAddress.String()},
				DNSNames:                []string{""},
				EdgeCertSigningDuration: 365,
				Quic: &CloudHubQUIC{
					Enable:             false,
					Address:            "0.0.0.0",
					Port:               10001,
					MaxIncomingStreams: 10000,
				},
				UnixSocket: &CloudHubUnixSocket{
					Enable:  true,
					Address: "unix:///var/lib/kubeedge/kubeedge.sock",
				},
				WebSocket: &CloudHubWebSocket{
					Enable:  true,
					Port:    10000,
					Address: "0.0.0.0",
				},
				HTTPS: &CloudHubHTTPS{
					Enable:  true,
					Port:    10002,
					Address: "0.0.0.0",
				},
			},
			EdgeController: &EdgeController{
				Enable:              true,
				NodeUpdateFrequency: 10,
				Buffer: &EdgeControllerBuffer{
					UpdatePodStatus:            constants.DefaultUpdatePodStatusBuffer,
					UpdateNodeStatus:           constants.DefaultUpdateNodeStatusBuffer,
					QueryConfigMap:             constants.DefaultQueryConfigMapBuffer,
					QuerySecret:                constants.DefaultQuerySecretBuffer,
					QueryService:               constants.DefaultQueryServiceBuffer,
					QueryEndpoints:             constants.DefaultQueryEndpointsBuffer,
					PodEvent:                   constants.DefaultPodEventBuffer,
					ConfigMapEvent:             constants.DefaultConfigMapEventBuffer,
					SecretEvent:                constants.DefaultSecretEventBuffer,
					ServiceEvent:               constants.DefaultServiceEventBuffer,
					EndpointsEvent:             constants.DefaultEndpointsEventBuffer,
					QueryPersistentVolume:      constants.DefaultQueryPersistentVolumeBuffer,
					QueryPersistentVolumeClaim: constants.DefaultQueryPersistentVolumeClaimBuffer,
					QueryVolumeAttachment:      constants.DefaultQueryVolumeAttachmentBuffer,
					QueryNode:                  constants.DefaultQueryNodeBuffer,
					UpdateNode:                 constants.DefaultUpdateNodeBuffer,
					DeletePod:                  constants.DefaultDeletePodBuffer,
				},
				Context: &EdgeControllerContext{
					SendModule:     metaconfig.ModuleNameCloudHub,
					ReceiveModule:  metaconfig.ModuleNameEdgeController,
					ResponseModule: metaconfig.ModuleNameCloudHub,
				},
				Load: &EdgeControllerLoad{
					UpdatePodStatusWorkers:            constants.DefaultUpdatePodStatusWorkers,
					UpdateNodeStatusWorkers:           constants.DefaultUpdateNodeStatusWorkers,
					QueryConfigMapWorkers:             constants.DefaultQueryConfigMapWorkers,
					QuerySecretWorkers:                constants.DefaultQuerySecretWorkers,
					QueryServiceWorkers:               constants.DefaultQueryServiceWorkers,
					QueryEndpointsWorkers:             constants.DefaultQueryEndpointsWorkers,
					QueryPersistentVolumeWorkers:      constants.DefaultQueryPersistentVolumeWorkers,
					QueryPersistentVolumeClaimWorkers: constants.DefaultQueryPersistentVolumeClaimWorkers,
					QueryVolumeAttachmentWorkers:      constants.DefaultQueryVolumeAttachmentWorkers,
					QueryNodeWorkers:                  constants.DefaultQueryNodeWorkers,
					UpdateNodeWorkers:                 constants.DefaultUpdateNodeWorkers,
					DeletePodWorkers:                  constants.DefaultDeletePodWorkers,
				},
			},
			DeviceController: &DeviceController{
				Enable: true,
				Context: &DeviceControllerContext{
					SendModule:     metaconfig.ModuleNameCloudHub,
					ReceiveModule:  metaconfig.ModuleNameDeviceController,
					ResponseModule: metaconfig.ModuleNameCloudHub,
				},
				Buffer: &DeviceControllerBuffer{
					UpdateDeviceStatus: constants.DefaultUpdateDeviceStatusBuffer,
					DeviceEvent:        constants.DefaultDeviceEventBuffer,
					DeviceModelEvent:   constants.DefaultDeviceModelEventBuffer,
				},
				Load: &DeviceControllerLoad{
					UpdateDeviceStatusWorkers: constants.DefaultUpdateDeviceStatusWorkers,
				},
			},
			SyncController: &SyncController{
				Enable: true,
			},
			CloudStream: &CloudStream{
				Enable:                  false,
				TLSTunnelCAFile:         constants.DefaultCAFile,
				TLSTunnelCertFile:       constants.DefaultCertFile,
				TLSTunnelPrivateKeyFile: constants.DefaultKeyFile,
				TunnelPort:              constants.DefaultTunnelPort,
				TLSStreamCAFile:         constants.DefaultStreamCAFile,
				TLSStreamCertFile:       constants.DefaultStreamCertFile,
				TLSStreamPrivateKeyFile: constants.DefaultStreamKeyFile,
				StreamPort:              10003,
			},
		},
		LeaderElection: &componentbaseconfig.LeaderElectionConfiguration{
			LeaderElect:       false,
			LeaseDuration:     metav1.Duration{Duration: 15 * time.Second},
			RenewDeadline:     metav1.Duration{Duration: 10 * time.Second},
			RetryPeriod:       metav1.Duration{Duration: 2 * time.Second},
			ResourceLock:      "endpointsleases",
			ResourceNamespace: constants.KubeEdgeNameSpace,
			ResourceName:      "cloudcorelease",
		},
	}
	return c
}

// NewMinCloudCoreConfig returns a min CloudCoreConfig object
func NewMinCloudCoreConfig() *CloudCoreConfig {
	advertiseAddress, _ := utilnet.ChooseHostInterface()

	return &CloudCoreConfig{
		TypeMeta: metav1.TypeMeta{
			Kind:       Kind,
			APIVersion: path.Join(GroupName, APIVersion),
		},
		KubeAPIConfig: &KubeAPIConfig{
			Master:     "",
			KubeConfig: constants.DefaultKubeConfig,
		},
		Modules: &Modules{
			CloudHub: &CloudHub{
				NodeLimit:         1000,
				TLSCAFile:         constants.DefaultCAFile,
				TLSCAKeyFile:      constants.DefaultCAKeyFile,
				TLSCertFile:       constants.DefaultCertFile,
				TLSPrivateKeyFile: constants.DefaultKeyFile,
				AdvertiseAddress:  []string{advertiseAddress.String()},
				UnixSocket: &CloudHubUnixSocket{
					Enable:  true,
					Address: "unix:///var/lib/kubeedge/kubeedge.sock",
				},
				WebSocket: &CloudHubWebSocket{
					Enable:  true,
					Port:    10000,
					Address: "0.0.0.0",
				},
				HTTPS: &CloudHubHTTPS{
					Enable:  true,
					Port:    10002,
					Address: "0.0.0.0",
				},
			},
		},
		LeaderElection: &componentbaseconfig.LeaderElectionConfiguration{
			LeaderElect: false,
		},
	}
}
