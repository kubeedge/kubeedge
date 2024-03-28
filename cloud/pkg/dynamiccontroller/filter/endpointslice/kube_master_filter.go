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
	discovery "k8s.io/api/discovery/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/klog/v2"
)

const (
	masterEndpointSliceName      = "kubernetes"
	masterEndpointSliceNameSpace = "default"
	defaultMetaServerIP          = "127.0.0.1"
	defaultMetaServerPort        = 10550
	defaultEndpointSlicePortName = "https"
)

type KubeMasterFilter struct{}

func (kmf *KubeMasterFilter) Name() string {
	return "KubeMasterFilter"
}

func (kmf *KubeMasterFilter) NeedFilter(content runtime.Object) bool {
	switch v := content.(type) {
	case *discovery.EndpointSlice:
		if v.GetName() == masterEndpointSliceName && v.GetNamespace() == masterEndpointSliceNameSpace {
			return true
		}
	case *discovery.EndpointSliceList:
		if len(v.Items) != 0 {
			return true
		}
	case *unstructured.Unstructured:
		if v.GetName() == masterEndpointSliceName && v.GetNamespace() == masterEndpointSliceNameSpace {
			return true
		}
	case *unstructured.UnstructuredList:
		if len(v.Items) != 0 {
			return true
		}
	}
	return false
}

func (kmf *KubeMasterFilter) FilterResource(targetNode string, obj runtime.Object) {
	switch v := obj.(type) {
	case *discovery.EndpointSlice:
		kmf.mutateMasterEndpointSlice(v)
	case *discovery.EndpointSliceList:
		for index := range v.Items {
			if newEp, changed := kmf.mutateMasterEndpointSlice(&v.Items[index]); changed {
				v.Items[index] = *newEp
				break
			}
		}
	case *unstructured.Unstructured:
		kmf.mutateUnstructuredEndpointSlice(v)
	case *unstructured.UnstructuredList:
		for index := range v.Items {
			if newEp, changed := kmf.mutateUnstructuredEndpointSlice(&v.Items[index]); changed {
				v.Items[index] = *newEp
				break
			}
		}
	}
}

func (kmf *KubeMasterFilter) mutateMasterEndpointSlice(eps *discovery.EndpointSlice) (*discovery.EndpointSlice, bool) {
	if eps.GetName() != masterEndpointSliceName || eps.GetNamespace() != masterEndpointSliceNameSpace {
		return eps, false
	}

	if len(eps.Endpoints) <= 0 {
		klog.V(4).Info("default endpointSlice length 0")
		return eps, false
	}

	eps.Endpoints = eps.Endpoints[:1]
	if len(eps.Endpoints[0].Addresses) > 0 {
		// only need one endpoint which represent local host ip
		eps.Endpoints[0].Addresses = eps.Endpoints[0].Addresses[:1]
		eps.Endpoints[0].Addresses[0] = defaultMetaServerIP
		for i := range eps.Ports {
			if *(eps.Ports[i].Name) == defaultEndpointSlicePortName {
				*(eps.Ports[i].Port) = defaultMetaServerPort
				break
			}
		}
	}
	return eps, true
}

func (kmf *KubeMasterFilter) mutateUnstructuredEndpointSlice(obj *unstructured.Unstructured) (*unstructured.Unstructured, bool) {
	var eps discovery.EndpointSlice
	err := runtime.DefaultUnstructuredConverter.FromUnstructured(obj.UnstructuredContent(), &eps)
	if err != nil {
		klog.Errorf("convert unstructure content %v err: %v", obj.GetName(), err)
		return obj, false
	}

	newEps, changed := kmf.mutateMasterEndpointSlice(&eps)
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
