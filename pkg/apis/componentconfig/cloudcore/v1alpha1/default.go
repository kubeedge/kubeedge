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
	utilnet "k8s.io/apimachinery/pkg/util/net"

	"github.com/kubeedge/kubeedge/common/constants"
)

// NewDefaultCloudCoreConfig returns a full CloudCoreConfig object
func NewDefaultCloudCoreConfig() *CloudCoreConfig {
	advertiseAddress, _ := utilnet.ChooseHostInterface()

	c := &CloudCoreConfig{
		TypeMeta: metav1.TypeMeta{
			Kind:       Kind,
			APIVersion: path.Join(GroupName, APIVersion),
		},
		CommonConfig: &CommonConfig{
			TunnelPort: constants.ServerPort,
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
				TokenRefreshDuration:    12,
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
					RulesEvent:                 constants.DefaultRulesEventBuffer,
					RuleEndpointsEvent:         constants.DefaultRuleEndpointsEventBuffer,
					QueryPersistentVolume:      constants.DefaultQueryPersistentVolumeBuffer,
					QueryPersistentVolumeClaim: constants.DefaultQueryPersistentVolumeClaimBuffer,
					QueryVolumeAttachment:      constants.DefaultQueryVolumeAttachmentBuffer,
					QueryNode:                  constants.DefaultQueryNodeBuffer,
					UpdateNode:                 constants.DefaultUpdateNodeBuffer,
					DeletePod:                  constants.DefaultDeletePodBuffer,
					ServiceAccountToken:        constants.DefaultServiceAccountTokenBuffer,
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
					UpdateRuleStatusWorkers:           constants.DefaultUpdateRuleStatusWorkers,
					ServiceAccountTokenWorkers:        constants.DefaultServiceAccountTokenWorkers,
				},
			},
			DeviceController: &DeviceController{
				Enable: true,
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
			DynamicController: &DynamicController{
				Enable: false,
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
			Router: &Router{
				Enable:      false,
				Address:     "0.0.0.0",
				Port:        9443,
				RestTimeout: 60,
			},
			IptablesManager: &IptablesManager{
				Enable: true,
				Mode:   InternalMode,
			},
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
			Router: &Router{
				Enable:      false,
				Address:     "0.0.0.0",
				Port:        9443,
				RestTimeout: 60,
			},
			IptablesManager: &IptablesManager{
				Enable: true,
				Mode:   InternalMode,
			},
		},
	}
}
