/*
Copyright 2022 The KubeEdge Authors.

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

package endpoints

import (
	"fmt"

	v1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/tools/cache"
	"k8s.io/klog/v2"

	"github.com/kubeedge/kubeedge/cloud/pkg/common/informers"
	"github.com/kubeedge/kubeedge/cloud/pkg/controllermanager/nodegroup"
	"github.com/kubeedge/kubeedge/cloud/pkg/dynamiccontroller/filter"
)

type NodeGroupFilter struct {
	serviceLister cache.GenericLister
	nodeLister    cache.GenericLister
}

func newNodeGroupFilter() *NodeGroupFilter {
	serviceLister, err := informers.GetInformersManager().GetLister(v1.SchemeGroupVersion.WithResource("services"))
	if err != nil {
		panic(fmt.Errorf("endpoints NodeGroupFilter Register failed: get service lister err: %v", err))
	}

	nodeLister, err := informers.GetInformersManager().GetLister(v1.SchemeGroupVersion.WithResource("nodes"))
	if err != nil {
		panic(fmt.Errorf("endpoints NodeGroupFilter Register failed: get node lister err: %v", err))
	}

	return &NodeGroupFilter{serviceLister: serviceLister, nodeLister: nodeLister}
}

func (ngf *NodeGroupFilter) Name() string {
	return "NodeGroupFilter"
}

func (ngf *NodeGroupFilter) NeedFilter(content runtime.Object) bool {
	switch v := content.(type) {
	case *v1.Endpoints, *unstructured.Unstructured:
		return true
	case *v1.EndpointsList:
		if len(v.Items) != 0 {
			return true
		}
	case *unstructured.UnstructuredList:
		if len(v.Items) != 0 {
			return true
		}
	}
	return false
}

func (ngf *NodeGroupFilter) mutateEndpoint(ep *v1.Endpoints, targetNode string) (*v1.Endpoints, bool) {
	service, err := ngf.getService(ep)
	if err != nil {
		klog.Errorf("[NodeGroupFilter] get svc %s error: %v", ep.GetName(), err)
		return ep, false
	}

	if service.GetAnnotations()[nodegroup.ServiceTopologyAnnotation] != nodegroup.ServiceTopologyRangeNodegroup {
		klog.V(4).Info("skip filter for endpoints %v", ep.GetName())
		return ep, false
	}

	for i := range ep.Subsets {
		ep.Subsets[i].Addresses = ngf.filterEndpointsAddress(targetNode, ep.Subsets[i].Addresses)
		ep.Subsets[i].NotReadyAddresses = ngf.filterEndpointsAddress(targetNode, ep.Subsets[i].NotReadyAddresses)
	}

	return ep, true
}

func (ngf *NodeGroupFilter) mutateUnstructuredEndpoint(obj *unstructured.Unstructured, targetNode string) (*unstructured.Unstructured, bool) {
	var ep v1.Endpoints
	err := runtime.DefaultUnstructuredConverter.FromUnstructured(obj.UnstructuredContent(), &ep)
	if err != nil {
		klog.Errorf("convert unstructured content %v err: %v", obj.GetName(), err)
		return obj, false
	}

	newEp, changed := ngf.mutateEndpoint(&ep, targetNode)
	if !changed {
		return obj, false
	}

	unstrRaw, err := runtime.DefaultUnstructuredConverter.ToUnstructured(&newEp)
	if err != nil {
		klog.Errorf("default endpoints %v convert to unstructured error: %v", ep.Name, err)
		return obj, false
	}
	obj.SetUnstructuredContent(unstrRaw)
	return obj, true
}

func (ngf *NodeGroupFilter) FilterResource(targetNode string, obj runtime.Object) {
	switch v := obj.(type) {
	case *v1.Endpoints:
		ngf.mutateEndpoint(v, targetNode)
	case *unstructured.Unstructured:
		ngf.mutateUnstructuredEndpoint(v, targetNode)
	case *v1.EndpointsList:
		for index := range v.Items {
			if newEp, changed := ngf.mutateEndpoint(&v.Items[index], targetNode); changed {
				v.Items[index] = *newEp
				break
			}
		}
	case *unstructured.UnstructuredList:
		for index := range v.Items {
			if newEp, changed := ngf.mutateUnstructuredEndpoint(&v.Items[index], targetNode); changed {
				v.Items[index] = *newEp
				break
			}
		}
	}
}

func (ngf *NodeGroupFilter) filterEndpointsAddress(targetNode string, address []v1.EndpointAddress) []v1.EndpointAddress {
	var tmpAddress []v1.EndpointAddress
	for _, addr := range address {
		if addr.NodeName == nil {
			continue
		}
		if filter.IsBelongToSameGroup(targetNode, *addr.NodeName, ngf.nodeLister) {
			tmpAddress = append(tmpAddress, addr)
		}
	}
	return tmpAddress
}

func (ngf *NodeGroupFilter) getService(ep *v1.Endpoints) (*v1.Service, error) {
	ret, err := ngf.serviceLister.ByNamespace(ep.Namespace).Get(ep.Name)
	if err != nil {
		return nil, err
	}
	return ret.(*v1.Service), err
}
