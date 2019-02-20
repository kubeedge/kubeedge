/*
Copyright 2016 The Kubernetes Authors.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.

@CHANGELOG
KubeEdge Authors: To create mini-kubelet for edge deployment scenario,
This file is derived from K8S Kubelet code with reduced set of methods
Changes done are
1. This file is derived from kubernetes/pkg/kubelet/volume_host.go
 edgedVolumeHost is derived from kubeletVolumeHost but pruned sections that we don't need
 and made some variant.
*/

package edged

import (
	"fmt"
	"net"

	api "k8s.io/api/core/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/tools/cache"
	"k8s.io/kubernetes/pkg/cloudprovider"
	"k8s.io/kubernetes/pkg/util/io"
	"k8s.io/kubernetes/pkg/util/mount"
	"k8s.io/kubernetes/pkg/volume"

	"k8s.io/apimachinery/pkg/types"
)

// NewInitializedVolumePluginMgr returns a new instance of volume.VolumePluginMgr
func NewInitializedVolumePluginMgr(
	edge *edged,
	plugins []volume.VolumePlugin) (*volume.VolumePluginMgr, error) {
	evh := &edgedVolumeHost{
		edge:            edge,
		volumePluginMgr: volume.VolumePluginMgr{},
	}

	if err := evh.volumePluginMgr.InitPlugins(plugins, nil, evh); err != nil {
		return nil, fmt.Errorf(
			"Could not initialize volume plugins for KubeletVolumePluginMgr: %v",
			err)
	}

	return &evh.volumePluginMgr, nil
}

// Compile-time check to ensure kubeletVolumeHost implements the VolumeHost interface
var _ volume.VolumeHost = &edgedVolumeHost{}

func (evh *edgedVolumeHost) GetSecretStore() cache.Store {
	return evh.edge.secretStore
}

func (evh *edgedVolumeHost) GetConfigMapStore() cache.Store {
	return evh.edge.configMapStore
}

func (evh *edgedVolumeHost) GetPluginDir(pluginName string) string {
	return evh.edge.getPluginDir(pluginName)
}

type edgedVolumeHost struct {
	edge            *edged
	volumePluginMgr volume.VolumePluginMgr
}

func (evh *edgedVolumeHost) GetPodVolumeDir(podUID types.UID, pluginName string, volumeName string) string {
	return evh.edge.getPodVolumeDir(podUID, pluginName, volumeName)
}

func (evh *edgedVolumeHost) GetPodPluginDir(podUID types.UID, pluginName string) string {
	return evh.edge.getPodPluginDir(podUID, pluginName)
}

func (evh *edgedVolumeHost) GetKubeClient() kubernetes.Interface {
	// TODO: we need figure out a way to return metaClient
	// return evh.edge.metaClient
	return nil
}

func (evh *edgedVolumeHost) NewWrapperMounter(
	volName string,
	spec volume.Spec,
	pod *api.Pod,
	opts volume.VolumeOptions) (volume.Mounter, error) {
	// The name of wrapper volume is set to "wrapped_{wrapped_volume_name}"
	wrapperVolumeName := "wrapped_" + volName
	if spec.Volume != nil {
		spec.Volume.Name = wrapperVolumeName
	}

	return evh.edge.newVolumeMounterFromPlugins(&spec, pod, opts)
}

func (evh *edgedVolumeHost) NewWrapperUnmounter(volName string, spec volume.Spec, podUID types.UID) (volume.Unmounter, error) {
	// The name of wrapper volume is set to "wrapped_{wrapped_volume_name}"
	wrapperVolumeName := "wrapped_" + volName
	if spec.Volume != nil {
		spec.Volume.Name = wrapperVolumeName
	}

	plugin, err := evh.edge.volumePluginMgr.FindPluginBySpec(&spec)
	if err != nil {
		return nil, err
	}

	return plugin.NewUnmounter(spec.Name(), podUID)
}

// Below is part of k8s.io/kubernetes/pkg/volume.VolumeHost interface.
func (evh *edgedVolumeHost) GetMounter(pluginName string) mount.Interface { return evh.edge.mounter }
func (evh *edgedVolumeHost) GetWriter() io.Writer                         { return evh.edge.writer }
func (evh *edgedVolumeHost) GetHostName() string                          { return evh.edge.hostname }
func (evh *edgedVolumeHost) GetCloudProvider() cloudprovider.Interface    { return nil }
func (evh *edgedVolumeHost) GetConfigMapFunc() func(namespace, name string) (*api.ConfigMap, error) {
	return func(namespace, name string) (*api.ConfigMap, error) {
		return evh.edge.metaClient.ConfigMaps(namespace).Get(name)
	}
}
func (evh *edgedVolumeHost) GetExec(pluginName string) mount.Exec          { return nil }
func (evh *edgedVolumeHost) GetHostIP() (net.IP, error)                    { return nil, nil }
func (evh *edgedVolumeHost) GetNodeAllocatable() (api.ResourceList, error) { return nil, nil }
func (evh *edgedVolumeHost) GetNodeLabels() (map[string]string, error)     { return nil, nil }
func (evh *edgedVolumeHost) GetNodeName() types.NodeName                   { return "" }
func (evh *edgedVolumeHost) GetPodVolumeDeviceDir(podUID types.UID, pluginName string) string {
	return ""
}
func (evh *edgedVolumeHost) GetSecretFunc() func(namespace, name string) (*api.Secret, error) {
	return func(namespace, name string) (*api.Secret, error) {
		return evh.edge.metaClient.Secrets(namespace).Get(name)
	}
}
func (evh *edgedVolumeHost) GetVolumeDevicePluginDir(pluginName string) string { return "" }
