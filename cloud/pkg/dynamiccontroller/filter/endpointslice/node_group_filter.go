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

package endpointslice

import (
	"fmt"

	v1 "k8s.io/api/core/v1"
	discovery "k8s.io/api/discovery/v1"
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
		panic(fmt.Errorf("endpointSlices NodeGroupFilter Register failed: get service lister err: %v", err))
	}

	nodeLister, err := informers.GetInformersManager().GetLister(v1.SchemeGroupVersion.WithResource("nodes"))
	if err != nil {
		panic(fmt.Errorf("endpointSlices NodeGroupFilter Register failed: get node lister err: %v", err))
	}

	return &NodeGroupFilter{serviceLister: serviceLister, nodeLister: nodeLister}
}

func (ngf *NodeGroupFilter) Name() string {
	return "NodeGroupFilter"
}

func (ngf *NodeGroupFilter) NeedFilter(content runtime.Object) bool {
	switch v := content.(type) {
	case *discovery.EndpointSlice, *unstructured.Unstructured:
		return true
	case *discovery.EndpointSliceList:
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

func (ngf *NodeGroupFilter) mutateEndpointSlice(epSlice *discovery.EndpointSlice, targetNode string) (*discovery.EndpointSlice, bool) {
	var svcTopology string
	if svcName, ok := epSlice.Labels[discovery.LabelServiceName]; ok {
		service, err := ngf.getService(epSlice.Namespace, svcName)
		if err != nil {
			klog.Errorf("[NodeGroupFilter] get svc %s error: %v", svcName, err)
			return epSlice, false
		}
		svcTopology = service.Annotations[nodegroup.ServiceTopologyAnnotation]
	}

	if svcTopology != nodegroup.ServiceTopologyRangeNodegroup {
		klog.V(4).Info("skip filter for endpointSlice %v", epSlice.GetName())
		return epSlice, false
	}

	var epsTmp []discovery.Endpoint
	for _, ep := range epSlice.Endpoints {
		if filter.IsBelongToSameGroup(targetNode, *ep.NodeName, ngf.nodeLister) {
			epsTmp = append(epsTmp, ep)
		}
	}
	epSlice.Endpoints = epsTmp

	return epSlice, true
}

func (ngf *NodeGroupFilter) mutateUnstructuredEndpointSlice(obj *unstructured.Unstructured, targetNode string) (*unstructured.Unstructured, bool) {
	var eps discovery.EndpointSlice
	err := runtime.DefaultUnstructuredConverter.FromUnstructured(obj.UnstructuredContent(), &eps)
	if err != nil {
		klog.Errorf("convert unstructured content %v err: %v", obj.GetName(), err)
		return obj, false
	}

	newEps, changed := ngf.mutateEndpointSlice(&eps, targetNode)
	if !changed {
		return obj, false
	}

	unstrRaw, err := runtime.DefaultUnstructuredConverter.ToUnstructured(&newEps)
	if err != nil {
		klog.Errorf("default endpointSlice %v convert to unstructured error: %v", eps.Name, err)
		return obj, false
	}
	obj.SetUnstructuredContent(unstrRaw)

	return obj, true
}

func (ngf *NodeGroupFilter) FilterResource(targetNode string, obj runtime.Object) {
	switch v := obj.(type) {
	case *discovery.EndpointSlice:
		ngf.mutateEndpointSlice(v, targetNode)
	case *unstructured.Unstructured:
		ngf.mutateUnstructuredEndpointSlice(v, targetNode)
	case *discovery.EndpointSliceList:
		for index := range v.Items {
			if newEp, changed := ngf.mutateEndpointSlice(&v.Items[index], targetNode); changed {
				v.Items[index] = *newEp
				break
			}
		}
	case *unstructured.UnstructuredList:
		for index := range v.Items {
			if newEp, changed := ngf.mutateUnstructuredEndpointSlice(&v.Items[index], targetNode); changed {
				v.Items[index] = *newEp
				break
			}
		}
	}
}

func (ngf *NodeGroupFilter) getService(ns, name string) (*v1.Service, error) {
	ret, err := ngf.serviceLister.ByNamespace(ns).Get(name)
	if err != nil {
		return nil, err
	}
	return ret.(*v1.Service), err
}
