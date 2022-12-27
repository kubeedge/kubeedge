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
	v1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/klog/v2"
)

const (
	masterEndpointsName      = "kubernetes"
	masterEndpointsNameSpace = "default"
	masterPortName           = "https"
	defaultMetaServerIP      = "127.0.0.1"
	defaultMetaServerPort    = 10550
)

type KubeMasterFilter struct{}

func (kmf *KubeMasterFilter) Name() string {
	return "KubeMasterFilter"
}

func (kmf *KubeMasterFilter) NeedFilter(content runtime.Object) bool {
	switch v := content.(type) {
	case *v1.Endpoints:
		if v.GetName() == masterEndpointsName && v.GetNamespace() == masterEndpointsNameSpace {
			return true
		}
	case *v1.EndpointsList:
		if len(v.Items) != 0 {
			return true
		}
	case *unstructured.Unstructured:
		if v.GetName() == masterEndpointsName && v.GetNamespace() == masterEndpointsNameSpace {
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
	case *v1.Endpoints:
		kmf.mutateMasterEndpoint(v)
	case *v1.EndpointsList:
		for index := range v.Items {
			if newEp, changed := kmf.mutateMasterEndpoint(&v.Items[index]); changed {
				v.Items[index] = *newEp
				break
			}
		}
	case *unstructured.Unstructured:
		kmf.mutateUnstructuredEndpoint(v)
	case *unstructured.UnstructuredList:
		for index := range v.Items {
			if newEp, changed := kmf.mutateUnstructuredEndpoint(&v.Items[index]); changed {
				v.Items[index] = *newEp
				break
			}
		}
	}
}

func (kmf *KubeMasterFilter) mutateMasterEndpoint(ep *v1.Endpoints) (*v1.Endpoints, bool) {
	if ep.GetName() != masterEndpointsName || ep.GetNamespace() != masterEndpointsNameSpace {
		return ep, false
	}

	if len(ep.Subsets) <= 0 {
		return ep, false
	}

	ep.Subsets = ep.Subsets[:1]

	if len(ep.Subsets[0].Addresses) > 0 {
		// only need one endpoint which represent local host ip
		ep.Subsets[0].Addresses = ep.Subsets[0].Addresses[:1]
		ep.Subsets[0].Addresses[0].IP = defaultMetaServerIP
		for i := range ep.Subsets[0].Ports {
			if ep.Subsets[0].Ports[i].Name == masterPortName {
				ep.Subsets[0].Ports[i].Port = defaultMetaServerPort
				break
			}
		}
	}

	return ep, true
}

func (kmf *KubeMasterFilter) mutateUnstructuredEndpoint(obj *unstructured.Unstructured) (*unstructured.Unstructured, bool) {
	var ep v1.Endpoints
	err := runtime.DefaultUnstructuredConverter.FromUnstructured(obj.UnstructuredContent(), &ep)
	if err != nil {
		klog.Errorf("convert unstructured content %v err: %v", obj.GetName(), err)
		return obj, false
	}

	newEp, changed := kmf.mutateMasterEndpoint(&ep)
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
