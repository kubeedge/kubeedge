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
	"os"
	"path"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/kubeedge/kubeedge/common/constants"
	cloudcoreconfig "github.com/kubeedge/kubeedge/pkg/apis/componentconfig/cloudcore/v1alpha1"
	edgecoreconfig "github.com/kubeedge/kubeedge/pkg/apis/componentconfig/edgecore/v1alpha1"
	metaconfig "github.com/kubeedge/kubeedge/pkg/apis/componentconfig/meta/v1alpha1"
	"github.com/kubeedge/kubeedge/pkg/util"
)

// NewDefaultEdgeSiteConfig returns a full EdgeSiteConfig object
func NewDefaultEdgeSiteConfig() *EdgeSiteConfig {
	hostnameOverride, err := os.Hostname()
	if err != nil {
		hostnameOverride = constants.DefaultHostnameOverride
	}
	localIP, _ := util.GetLocalIP(hostnameOverride)
	return &EdgeSiteConfig{
		TypeMeta: metav1.TypeMeta{
			Kind:       Kind,
			APIVersion: path.Join(GroupName, APIVersion),
		},
		DataBase: &edgecoreconfig.DataBase{
			DriverName: DataBaseDriverName,
			AliasName:  DataBaseAliasName,
			DataSource: DataBaseDataSource,
		},
		KubeAPIConfig: &cloudcoreconfig.KubeAPIConfig{
			Master:      "",
			ContentType: constants.DefaultKubeContentType,
			QPS:         constants.DefaultKubeQPS,
			Burst:       constants.DefaultKubeBurst,
			KubeConfig:  constants.DefaultKubeConfig,
		},
		Modules: &Modules{
			EdgeController: &cloudcoreconfig.EdgeController{
				Enable:              true,
				NodeUpdateFrequency: 10,
				Buffer: &cloudcoreconfig.EdgeControllerBuffer{
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
				Context: &cloudcoreconfig.EdgeControllerContext{
					SendModule:     metaconfig.ModuleNameMetaManager,
					ReceiveModule:  metaconfig.ModuleNameEdgeController,
					ResponseModule: metaconfig.ModuleNameMetaManager,
				},
				Load: &cloudcoreconfig.EdgeControllerLoad{
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
			Edged: &edgecoreconfig.Edged{
				Enable:                      true,
				NodeStatusUpdateFrequency:   constants.DefaultNodeStatusUpdateFrequency,
				DockerAddress:               constants.DefaultDockerAddress,
				RuntimeType:                 constants.DefaultRuntimeType,
				NodeIP:                      localIP,
				ClusterDNS:                  "",
				ClusterDomain:               "",
				EdgedMemoryCapacity:         constants.DefaultEdgedMemoryCapacity,
				RemoteRuntimeEndpoint:       constants.DefaultRemoteRuntimeEndpoint,
				RemoteImageEndpoint:         constants.DefaultRemoteImageEndpoint,
				PodSandboxImage:             constants.DefaultPodSandboxImage,
				ImagePullProgressDeadline:   constants.DefaultImagePullProgressDeadline,
				RuntimeRequestTimeout:       constants.DefaultRuntimeRequestTimeout,
				HostnameOverride:            hostnameOverride,
				RegisterNode:                true,
				ConcurrentConsumers:         constants.DefaultConcurrentConsumers,
				RegisterNodeNamespace:       constants.DefaultRegisterNodeNamespace,
				DevicePluginEnabled:         false,
				GPUPluginEnabled:            false,
				ImageGCHighThreshold:        constants.DefaultImageGCHighThreshold,
				ImageGCLowThreshold:         constants.DefaultImageGCLowThreshold,
				MaximumDeadContainersPerPod: constants.DefaultMaximumDeadContainersPerPod,
				CGroupDriver:                edgecoreconfig.CGroupDriverCGroupFS,
			},
			MetaManager: &edgecoreconfig.MetaManager{
				Enable:                true,
				ContextSendGroup:      metaconfig.GroupNameEdgeController,
				ContextSendModule:     metaconfig.ModuleNameEdgeController,
				PodStatusSyncInterval: constants.DefaultPodStatusSyncInterval,
				RemoteQueryTimeout:    constants.DefaultRemoteQueryTimeout,
			},
		},
	}
}

// NewMinEdgeSiteConfig returns a common EdgeSiteConfig object
func NewMinEdgeSiteConfig() *EdgeSiteConfig {
	hostnameOverride, err := os.Hostname()
	if err != nil {
		hostnameOverride = constants.DefaultHostnameOverride
	}
	localIP, _ := util.GetLocalIP(hostnameOverride)
	return &EdgeSiteConfig{
		TypeMeta: metav1.TypeMeta{
			Kind:       Kind,
			APIVersion: path.Join(GroupName, APIVersion),
		},
		DataBase: &edgecoreconfig.DataBase{
			DataSource: DataBaseDataSource,
		},
		KubeAPIConfig: &cloudcoreconfig.KubeAPIConfig{
			Master:     "",
			KubeConfig: constants.DefaultKubeConfig,
		},
		Modules: &Modules{
			Edged: &edgecoreconfig.Edged{
				DockerAddress:         constants.DefaultDockerAddress,
				RuntimeType:           constants.DefaultRuntimeType,
				NodeIP:                localIP,
				ClusterDNS:            "",
				ClusterDomain:         "",
				RemoteRuntimeEndpoint: constants.DefaultRemoteRuntimeEndpoint,
				RemoteImageEndpoint:   constants.DefaultRemoteImageEndpoint,
				PodSandboxImage:       util.GetPodSandboxImage(),
				HostnameOverride:      hostnameOverride,
				DevicePluginEnabled:   false,
				GPUPluginEnabled:      false,
				CGroupDriver:          edgecoreconfig.CGroupDriverCGroupFS,
			},
		},
	}
}
