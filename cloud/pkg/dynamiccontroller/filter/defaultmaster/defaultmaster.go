package defaultmaster

import (
	discovery "k8s.io/api/discovery/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/klog/v2"

	"github.com/kubeedge/kubeedge/cloud/pkg/dynamiccontroller/filter"
)

// FilterImpl implement enpointslice filter
type FilterImpl struct {
	hostIP string
	port   int32
}

const (
	defaultEndpointSliceName      = "kubernetes"
	defaultEndpointSliceNameSpace = "default"
	defaultEndpointSlicePortName  = "https"
	resourceName                  = "EndpointSlice"
	filterName                    = "defaultMaster"
	defaultMetaServerIP           = "127.0.0.1"
	defaultMetaServerPort         = 10550
)

func newDefaultMasterFilter() *FilterImpl {
	return &FilterImpl{
		hostIP: defaultMetaServerIP,
		port:   defaultMetaServerPort,
	}
}

func Register() {
	filter.Register(newDefaultMasterFilter())
}

func (f *FilterImpl) Name() string {
	return filterName
}

func (f *FilterImpl) NeedFilter(content interface{}) bool {
	if objList, ok := content.(*unstructured.UnstructuredList); ok {
		if len(objList.Items) != 0 && objList.Items[0].GetObjectKind().GroupVersionKind().Kind == resourceName {
			return true
		}
		return false
	}
	if obj, ok := content.(*unstructured.Unstructured); ok {
		if obj.GetObjectKind().GroupVersionKind().Kind == resourceName && obj.GetName() == defaultEndpointSliceName &&
			obj.GetNamespace() == defaultEndpointSliceNameSpace {
			return true
		}
	}
	return false
}

func (f *FilterImpl) FilterResource(targetNode string, obj runtime.Object) {
	unstruct, ok := obj.(*unstructured.Unstructured)
	if !ok {
		return
	}
	var eps discovery.EndpointSlice
	err := runtime.DefaultUnstructuredConverter.FromUnstructured(unstruct.UnstructuredContent(), &eps)
	if err != nil {
		klog.Errorf("convert unstructure content %v err: %v", unstruct.GetName(), err)
		return
	}
	if len(eps.Endpoints) <= 0 {
		klog.V(4).Info("default endpointSlice length 0")
		return
	}
	eps.Endpoints = eps.Endpoints[:1]
	if len(eps.Endpoints[0].Addresses) > 0 {
		// only need one endpoint which represent local host ip
		eps.Endpoints[0].Addresses = eps.Endpoints[0].Addresses[:1]
		eps.Endpoints[0].Addresses[0] = defaultMetaServerIP
		for i := range eps.Ports {
			if *(eps.Ports[i].Name) != defaultEndpointSlicePortName {
				continue
			}
			*(eps.Ports[i].Port) = defaultMetaServerPort
			unstrRaw, err := runtime.DefaultUnstructuredConverter.ToUnstructured(&eps)
			if err != nil {
				klog.Errorf("default endpointslice %v convert to unstructure error: %v", eps.Name, err)
				return
			}
			unstruct.SetUnstructuredContent(unstrRaw)
		}
	}
}
