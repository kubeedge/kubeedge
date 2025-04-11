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
	"context"
	"fmt"
	"time"

	v1 "k8s.io/api/core/v1"
	"k8s.io/client-go/informers"
	"k8s.io/client-go/tools/cache"
	"k8s.io/klog/v2"

	"github.com/kubeedge/kubeedge/edge/pkg/edged/bandwidth/consts"
	"github.com/kubeedge/kubeedge/edge/pkg/edged/bandwidth/kube"
)

const (
	refreshInterval = 5
)

var nodeName string

var AddBandwidthFunc = func(obj interface{}) {
	p, exist := obj.(*v1.Pod)
	if !exist {
		klog.Error("Pod type assertion failed")
		return
	}
	// check whether there is limited flow annotation
	if !filterPod(p) {
		return
	}
	if !isPodRunning(p) {
		klog.Warningf("ADD EVENT: pod `%s` is not running, please check reason", p.Name)
		return
	}
	klog.Infof("ADD EVENT: pod `%s` netlink interface bandwidth limit", p.Name)
	// pod network limit
	podBandwidthLimit(p)
	klog.Infof("ADD EVENT END...")
}

var UpdateBandwidthFunc = func(_, newObj interface{}) {
	p, exist := newObj.(*v1.Pod)
	if !exist {
		klog.Error("Pod type assertion failed")
		return
	}
	// check whether there is limited flow annotation
	if !filterPod(p) {
		return
	}
	// check whether the new container is running normally
	if !isPodRunning(p) {
		// If terminated state, delete the ifb network interface
		if isPodTerminated(p) {
			klog.Infof("UPDATE EVENT: pod `%s` terminated,delete ifb netlink interface,pod status:%s", p.Name, p.Status.Phase)
			deleteNetworkLimit(p)
			return
		}
		klog.Warningf("UPDATE EVENT: pod `%s` is not running,pod status:%s", p.Name, p.Status.Phase)
		return
	}
	klog.Infof("UPDATE EVENT: pod `%s` netlink interface bandwidth limit", p.Name)
	// pod network limit
	podBandwidthLimit(p)
	klog.Infof("UPDATE EVENT END...")
}

var DeleteBandwidthFunc = func(obj interface{}) {
	p, ok := obj.(*v1.Pod)
	if !ok {
		klog.Errorf("obj is not a *v1.Pod")
	}
	klog.Infof("DELETE EVENT: pod `%s` deleted", p.Name)
	// check whether there is limited flow annotation
	if !filterPod(p) {
		return
	}
	klog.Infof("DELETE EVENT: Delete pod `%s` ifb netlink interface...", p.Name)
	// delete the pod network interface
	deleteNetworkLimit(p)
	klog.Infof("DELETE EVENT END...")
}

func EdgeWatch(ctx context.Context, hostnameOverride string) error {
	nodeName = hostnameOverride
	client := kube.EdgeClient()
	// list sync cache every 5 minutes
	factory := informers.NewSharedInformerFactory(client, time.Minute*refreshInterval)
	// register event handler
	_, err := factory.Core().V1().Pods().Informer().AddEventHandler(cache.ResourceEventHandlerFuncs{
		AddFunc:    AddBandwidthFunc,
		UpdateFunc: UpdateBandwidthFunc,
		DeleteFunc: DeleteBandwidthFunc,
	})
	if err != nil {
		return fmt.Errorf("failed to add event handler: %v", err)
	}
	// start informer, List & Watch
	go factory.Start(ctx.Done())
	return nil
}

func filterPod(pod *v1.Pod) bool {
	if pod.Spec.NodeName != nodeName {
		return false
	}
	annotions := pod.ObjectMeta.Annotations
	if _, ok := annotions[consts.AnnotationIngressBandwidth]; ok {
		return true
	}
	if _, ok := annotions[consts.AnnotationEgressBandwidth]; ok {
		return true
	}
	return false
}

func isPodRunning(pod *v1.Pod) bool {
	// check if the pod is started
	status := pod.Status
	return status.Phase == v1.PodRunning
}

func isPodTerminated(pod *v1.Pod) bool {
	// check if pod is deleted
	status := pod.Status
	return status.Phase == v1.PodSucceeded
}
