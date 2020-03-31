// +build cgo,linux

/*
Copyright 2015 The Kubernetes Authors.
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
KubeEdge Authors: To init a cadvisor client, implement the cadvisor interface by empty functions.
This file is derived from k8s kubelet code with pruned structures and interfaces
and changed most of the realization.
changes done are
1.For cadvisor.Interface is been implemented here.
*/

package cadvisor

import (
	"github.com/google/cadvisor/events"
	cadvisorapi "github.com/google/cadvisor/info/v1"
	cadvisorapi2 "github.com/google/cadvisor/info/v2"
	"k8s.io/kubernetes/pkg/kubelet/cadvisor"
)

type cadvisorClient struct {
	rootPath string
}

// New creates new cadvisor client
func New(rootPath string) (cadvisor.Interface, error) {
	return &cadvisorClient{
		rootPath: rootPath,
	}, nil
}

func (cadvisorClient) Start() error {
	return nil
}

func (cadvisorClient) DockerContainer(name string, req *cadvisorapi.ContainerInfoRequest) (cadvisorapi.ContainerInfo, error) {
	return cadvisorapi.ContainerInfo{}, nil
}

func (cadvisorClient) ContainerInfo(name string, req *cadvisorapi.ContainerInfoRequest) (*cadvisorapi.ContainerInfo, error) {
	return nil, nil
}

func (cadvisorClient) ContainerInfoV2(name string, options cadvisorapi2.RequestOptions) (map[string]cadvisorapi2.ContainerInfo, error) {
	return nil, nil
}

func (cadvisorClient) SubcontainerInfo(name string, req *cadvisorapi.ContainerInfoRequest) (map[string]*cadvisorapi.ContainerInfo, error) {
	return nil, nil
}

//MachineInfo implement by hard code, just for initing a cadvisor client, edged will not use this function to get machine info.
func (cadvisorClient) MachineInfo() (*cadvisorapi.MachineInfo, error) {
	return &cadvisorapi.MachineInfo{
		NumCores:       4,
		CpuFrequency:   3,
		MemoryCapacity: 16000000,
		HugePages:      []cadvisorapi.HugePagesInfo{{PageSize: 4096, NumPages: 1024}},
		Filesystems:    []cadvisorapi.FsInfo{},
		DiskMap:        make(map[string]cadvisorapi.DiskInfo),
		NetworkDevices: []cadvisorapi.NetInfo{},
		Topology:       []cadvisorapi.Node{},
		CloudProvider:  cadvisorapi.UnknownProvider,
		InstanceType:   cadvisorapi.UnknownInstance,
		InstanceID:     cadvisorapi.UnNamedInstance,
	}, nil
}

func (cadvisorClient) VersionInfo() (*cadvisorapi.VersionInfo, error) {
	return nil, nil
}

func (cadvisorClient) ImagesFsInfo() (cadvisorapi2.FsInfo, error) {
	return cadvisorapi2.FsInfo{}, nil
}

func (cadvisorClient) RootFsInfo() (cadvisorapi2.FsInfo, error) {
	return cadvisorapi2.FsInfo{}, nil
}

func (cadvisorClient) WatchEvents(request *events.Request) (*events.EventChannel, error) {
	return nil, nil
}

func (cadvisorClient) GetDirFsInfo(path string) (cadvisorapi2.FsInfo, error) {
	return cadvisorapi2.FsInfo{}, nil
}
