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
			MonitorServer: MonitorServer{
				BindAddress:     "127.0.0.1:9091",
				EnableProfiling: false,
			},
		},
		KubeAPIConfig: &KubeAPIConfig{
			ContentType: constants.DefaultKubeContentType,
			QPS:         5 * constants.DefaultNodeLimit,
			Burst:       10 * constants.DefaultNodeLimit,
		},
		Modules: &Modules{
			CloudHub: &CloudHub{
				Enable:                  true,
				KeepaliveInterval:       30,
				NodeLimit:               constants.DefaultNodeLimit,
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
				Buffer:              getDefaultEdgeControllerBuffer(constants.DefaultNodeLimit),
				Load:                getDefaultEdgeControllerLoad(constants.DefaultNodeLimit),
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
			NodeUpgradeJobController: &NodeUpgradeJobController{
				Enable: false,
				Buffer: &NodeUpgradeJobControllerBuffer{
					UpdateNodeUpgradeJobStatus: constants.DefaultNodeUpgradeJobStatusBuffer,
					NodeUpgradeJobEvent:        constants.DefaultNodeUpgradeJobEventBuffer,
				},
				Load: &NodeUpgradeJobControllerLoad{
					NodeUpgradeJobWorkers: constants.DefaultNodeUpgradeJobWorkers,
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

// NodeLimit is a maximum number of edge node that can connect to the single CloudCore
// instance. You should take this parameter seriously, because this parameter is closely
// related to the number of goroutines for upstream message processing.
// getDefaultEdgeControllerLoad return Default EdgeControllerLoad based on nodeLimit
func getDefaultEdgeControllerLoad(nodeLimit int32) *EdgeControllerLoad {
	return &EdgeControllerLoad{
		UpdatePodStatusWorkers:            constants.DefaultUpdatePodStatusWorkers,
		UpdateNodeStatusWorkers:           constants.DefaultUpdateNodeStatusWorkers,
		QueryConfigMapWorkers:             constants.DefaultQueryConfigMapWorkers,
		QuerySecretWorkers:                constants.DefaultQuerySecretWorkers,
		QueryPersistentVolumeWorkers:      constants.DefaultQueryPersistentVolumeWorkers,
		QueryPersistentVolumeClaimWorkers: constants.DefaultQueryPersistentVolumeClaimWorkers,
		QueryVolumeAttachmentWorkers:      constants.DefaultQueryVolumeAttachmentWorkers,
		QueryNodeWorkers:                  nodeLimit,
		CreateNodeWorkers:                 constants.DefaultCreateNodeWorkers,
		PatchNodeWorkers:                  100 + nodeLimit/50,
		UpdateNodeWorkers:                 constants.DefaultUpdateNodeWorkers,
		PatchPodWorkers:                   constants.DefaultPatchPodWorkers,
		DeletePodWorkers:                  constants.DefaultDeletePodWorkers,
		CreateLeaseWorkers:                nodeLimit,
		QueryLeaseWorkers:                 constants.DefaultQueryLeaseWorkers,
		UpdateRuleStatusWorkers:           constants.DefaultUpdateRuleStatusWorkers,
		ServiceAccountTokenWorkers:        constants.DefaultServiceAccountTokenWorkers,
	}
}

// getDefaultEdgeControllerBuffer return Default EdgeControllerBuffer based on nodeLimit
func getDefaultEdgeControllerBuffer(nodeLimit int32) *EdgeControllerBuffer {
	return &EdgeControllerBuffer{
		UpdatePodStatus:            constants.DefaultUpdatePodStatusBuffer,
		UpdateNodeStatus:           constants.DefaultUpdateNodeStatusBuffer,
		QueryConfigMap:             constants.DefaultQueryConfigMapBuffer,
		QuerySecret:                constants.DefaultQuerySecretBuffer,
		PodEvent:                   constants.DefaultPodEventBuffer,
		ConfigMapEvent:             constants.DefaultConfigMapEventBuffer,
		SecretEvent:                constants.DefaultSecretEventBuffer,
		RulesEvent:                 constants.DefaultRulesEventBuffer,
		RuleEndpointsEvent:         constants.DefaultRuleEndpointsEventBuffer,
		QueryPersistentVolume:      constants.DefaultQueryPersistentVolumeBuffer,
		QueryPersistentVolumeClaim: constants.DefaultQueryPersistentVolumeClaimBuffer,
		QueryVolumeAttachment:      constants.DefaultQueryVolumeAttachmentBuffer,
		CreateNode:                 constants.DefaultCreateNodeBuffer,
		PatchNode:                  1024 + nodeLimit/2,
		QueryNode:                  1024 + nodeLimit,
		UpdateNode:                 constants.DefaultUpdateNodeBuffer,
		PatchPod:                   constants.DefaultPatchPodBuffer,
		DeletePod:                  constants.DefaultDeletePodBuffer,
		CreateLease:                1024 + nodeLimit,
		QueryLease:                 constants.DefaultQueryLeaseBuffer,
		ServiceAccountToken:        constants.DefaultServiceAccountTokenBuffer,
	}
}

func AdjustCloudCoreConfig(c *CloudCoreConfig) bool {
	changed := false
	nodeLimit := c.Modules.CloudHub.NodeLimit

	if c.KubeAPIConfig.QPS != 5*nodeLimit {
		changed = true
		c.KubeAPIConfig.QPS = 5 * nodeLimit
	}

	if c.KubeAPIConfig.Burst != 10*nodeLimit {
		changed = true
		c.KubeAPIConfig.Burst = 10 * nodeLimit
	}

	if c.Modules.EdgeController.Load.QueryNodeWorkers < nodeLimit {
		changed = true
		c.Modules.EdgeController.Load.QueryNodeWorkers = nodeLimit
	}

	if c.Modules.EdgeController.Load.PatchNodeWorkers < 100+nodeLimit/50 {
		changed = true
		c.Modules.EdgeController.Load.PatchNodeWorkers = 100 + nodeLimit/50
	}

	if c.Modules.EdgeController.Load.CreateLeaseWorkers < nodeLimit {
		changed = true
		c.Modules.EdgeController.Load.CreateLeaseWorkers = nodeLimit
	}

	if c.Modules.EdgeController.Buffer.PatchNode < 1024+nodeLimit/2 {
		changed = true
		c.Modules.EdgeController.Buffer.PatchNode = 1024 + nodeLimit/2
	}

	if c.Modules.EdgeController.Buffer.QueryNode < 1024+nodeLimit {
		changed = true
		c.Modules.EdgeController.Buffer.QueryNode = 1024 + nodeLimit
	}

	if c.Modules.EdgeController.Buffer.CreateLease < 1024+nodeLimit {
		changed = true
		c.Modules.EdgeController.Buffer.CreateLease = 1024 + nodeLimit
	}

	return changed
}

// NewMinCloudCoreConfig returns a min CloudCoreConfig object
func NewMinCloudCoreConfig() *CloudCoreConfig {
	advertiseAddress, _ := utilnet.ChooseHostInterface()

	return &CloudCoreConfig{
		TypeMeta: metav1.TypeMeta{
			Kind:       Kind,
			APIVersion: path.Join(GroupName, APIVersion),
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
