/*
Copyright 2024 The KubeEdge Authors.

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

package tclimit

import (
	"strings"

	v1 "k8s.io/api/core/v1"
	"k8s.io/klog/v2"

	"github.com/kubeedge/kubeedge/edge/pkg/edged/bandwidth/tclinux"
)

func doIngressBandwidthLimit(pod *v1.Pod) {
	klog.Infof("pod `%s` start Ingress bandwidth limit", pod.Name)
	// get container ID
	containerID, err := getContainerID(&pod.Status.ContainerStatuses[0])
	if err != nil {
		klog.Warningf("get container id failed when ingress bandwidth limit,err:%v", err)
		return
	}
	// get container network interface device
	netlinkDeviceName, err := getContainerHostNetworkDevice(pod, containerID)
	if err != nil {
		klog.Warningf("get container network device name failed when ingress bandwidth "+
			"limit,containerId:%s,err:%v", containerID, err)
		return
	}
	// parse current limit configuration
	conf, err := parseIngressTrafficControlConf(pod, netlinkDeviceName)
	if err != nil {
		klog.Warningf("parse ingress bandwidth conf failed,err:%v", err)
		return
	}
	// bandwidth limit (tbf solution)
	err = tclinux.CreateIngressQdisc(conf.rate, conf.burst, conf.netlinkDeviceName)
	if err != nil {
		klog.Warningf("ingress bandwidth limit failed,err:%v", err)
		return
	}
	// Store the current network interface and pod mapping relationship
	tclinux.StorePodNetworkDeviceMapping(pod.Name, conf.netlinkDeviceName)
	klog.Infof("pod `%s` Ingress bandwidth limit succcess to network interface name:%s", pod.Name, conf.netlinkDeviceName)
}

func doEgressBandwidthLimit(pod *v1.Pod) {
	klog.Infof("pod `%s` start Egress bandwidth limit", pod.Name)
	// get container ID
	containerID, err := getContainerID(&pod.Status.ContainerStatuses[0])
	if err != nil {
		klog.Warningf("get container id failed when egress bandwidth limit,err:%v", err)
		return
	}
	hostNetworkDevice, err := getContainerHostNetworkDevice(pod, containerID)
	if err != nil {
		klog.Warningf("get container network device name failed when egress bandwidth "+
			"limit,containerId:%s,err:%v", containerID, err)
		return
	}
	mtu, err := tclinux.GetMTU(hostNetworkDevice)
	if err != nil {
		klog.Warningf("pod `%s` get mtu param value failed when egress bandwidth limit,error:%v", pod.Name, err)
		return
	}
	// check if ifb device already exists
	ifbName := tclinux.GetIfbDeviceByPod(pod.Name)
	if ifbName == "" {
		// create ifb network card device
		ifbName = tclinux.CalIfbName(containerID)
		err = tclinux.CreateIfb(ifbName, mtu)
		if err != nil {
			klog.Warningf("pod `%s` create ifb device failed when egress bandwidth limit,error:%v", pod.Name, err)
			return
		}
	}
	// parse egress limit configuration parameters
	conf, err := parseEgressTrafficControlConf(pod, ifbName)
	if err != nil {
		klog.Warningf("pod `%s` parse egress bandwidth param failed,error:%v", pod.Name, err)
		return
	}
	// egress queue traffic limit and mirror
	err = tclinux.CreateEgressQdisc(conf.rate, conf.burst, hostNetworkDevice, ifbName)
	if err != nil {
		klog.Warningf("pod `%s` Egress bandwidth limit create qdisc failed,error:%v", pod.Name, err)
		return
	}
	// storage egress limit ifb network interface and pod relationship
	tclinux.StorePodNetworkDeviceMapping(pod.Name, ifbName)
	klog.Infof("pod `%s` egress bandwidth limit succcess to network interface name:%s", pod.Name, conf.netlinkDeviceName)
}

func getContainerHostNetworkDevice(pod *v1.Pod, containerID string) (string, error) {
	// get the container's network device from the cache
	var err error
	hostNetworkDevice := tclinux.GetNetworkDeviceByPod(pod.Name)
	if hostNetworkDevice == "" {
		// get in real time
		hostNetworkDevice, err = GetNetlinkDeviceName(containerID)
		if err != nil {
			return "", err
		}
		// save relationship between the pod name and the host network interface
		tclinux.StorePodNetworkDeviceMapping(pod.Name, hostNetworkDevice)
	}
	return hostNetworkDevice, err
}

func getContainerID(cs *v1.ContainerStatus) (string, error) {
	podContainerID := cs.ContainerID
	// get the pause container id of the current container
	index := strings.LastIndex(podContainerID, "/")
	return strings.Trim(podContainerID[index+1:], "\""), nil
}
