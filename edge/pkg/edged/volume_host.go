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
	"os"

	authenticationv1 "k8s.io/api/authentication/v1"
	api "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/informers"
	"k8s.io/client-go/kubernetes"
	storagelisters "k8s.io/client-go/listers/storage/v1"
	"k8s.io/client-go/tools/cache"
	recordtools "k8s.io/client-go/tools/record"
	cloudprovider "k8s.io/cloud-provider"
	"k8s.io/klog/v2"
	"k8s.io/kubernetes/pkg/volume"
	"k8s.io/kubernetes/pkg/volume/util/hostutil"
	"k8s.io/kubernetes/pkg/volume/util/subpath"
	utilexec "k8s.io/utils/exec"
	"k8s.io/utils/mount"
)

// NewInitializedVolumePluginMgr returns a new instance of volume.VolumePluginMgr
func NewInitializedVolumePluginMgr(
	edge *edged,
	plugins []volume.VolumePlugin) *volume.VolumePluginMgr {
	evh := &edgedVolumeHost{
		edge:            edge,
		volumePluginMgr: volume.VolumePluginMgr{},
	}

	if err := evh.volumePluginMgr.InitPlugins(plugins, nil, evh); err != nil {
		klog.Errorf("Could not initialize volume plugins for KubeletVolumePluginMgr: %v", err)
		os.Exit(1)
	}

	return &evh.volumePluginMgr
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
	return evh.edge.kubeClient
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
func (evh *edgedVolumeHost) GetHostName() string                          { return evh.edge.hostname }
func (evh *edgedVolumeHost) GetCloudProvider() cloudprovider.Interface    { return nil }
func (evh *edgedVolumeHost) GetConfigMapFunc() func(namespace, name string) (*api.ConfigMap, error) {
	return func(namespace, name string) (*api.ConfigMap, error) {
		return evh.edge.metaClient.ConfigMaps(namespace).Get(name)
	}
}
func (evh *edgedVolumeHost) GetExec(pluginName string) utilexec.Interface  { return nil }
func (evh *edgedVolumeHost) GetHostIP() (net.IP, error)                    { return nil, nil }
func (evh *edgedVolumeHost) GetNodeAllocatable() (api.ResourceList, error) { return nil, nil }
func (evh *edgedVolumeHost) GetNodeLabels() (map[string]string, error) {
	node, err := evh.edge.initialNode()
	if err != nil {
		return nil, fmt.Errorf("error retrieving node: %v", err)
	}
	return node.Labels, nil
}
func (evh *edgedVolumeHost) GetNodeName() types.NodeName { return types.NodeName(evh.edge.nodeName) }
func (evh *edgedVolumeHost) GetPodVolumeDeviceDir(podUID types.UID, pluginName string) string {
	return ""
}
func (evh *edgedVolumeHost) GetSecretFunc() func(namespace, name string) (*api.Secret, error) {
	return func(namespace, name string) (*api.Secret, error) {
		return evh.edge.metaClient.Secrets(namespace).Get(name)
	}
}
func (evh *edgedVolumeHost) GetVolumeDevicePluginDir(pluginName string) string { return "" }

func (evh *edgedVolumeHost) DeleteServiceAccountTokenFunc() func(podUID types.UID) {
	return func(types.UID) {}
}

func (evh *edgedVolumeHost) GetEventRecorder() recordtools.EventRecorder {
	return evh.edge.recorder
}

func (evh *edgedVolumeHost) GetPodsDir() string {
	return evh.edge.getPodsDir()
}

func (evh *edgedVolumeHost) GetServiceAccountTokenFunc() func(namespace, name string, tr *authenticationv1.TokenRequest) (*authenticationv1.TokenRequest, error) {
	return func(_, _ string, _ *authenticationv1.TokenRequest) (*authenticationv1.TokenRequest, error) {
		return nil, fmt.Errorf("GetServiceAccountToken unsupported")
	}
}

func (evh *edgedVolumeHost) GetSubpather() subpath.Interface {
	// No volume plugin needs Subpaths
	return subpath.New(evh.edge.mounter)
}

func (evh *edgedVolumeHost) GetHostUtil() hostutil.HostUtils {
	return evh.edge.hostUtil
}

// TODO: Evaluate the funcs releated to csi
func (evh *edgedVolumeHost) SetKubeletError(err error) {
}

func (evh *edgedVolumeHost) GetInformerFactory() informers.SharedInformerFactory {
	const resyncPeriod = 0
	return informers.NewSharedInformerFactory(evh.edge.kubeClient, resyncPeriod)
}

func (evh *edgedVolumeHost) CSIDriverLister() storagelisters.CSIDriverLister {
	return nil
}

func (evh *edgedVolumeHost) CSIDriversSynced() cache.InformerSynced {
	return nil
}

// WaitForCacheSync is a helper function that waits for cache sync for CSIDriverLister
func (evh *edgedVolumeHost) WaitForCacheSync() error {
	return nil
}
