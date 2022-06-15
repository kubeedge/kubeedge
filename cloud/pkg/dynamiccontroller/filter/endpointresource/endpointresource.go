package endpointresource

import (
	v1 "k8s.io/api/core/v1"
	discovery "k8s.io/api/discovery/v1"
	"k8s.io/apimachinery/pkg/api/meta"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/klog/v2"

	"github.com/kubeedge/kubeedge/cloud/pkg/controllermanager/nodegroup"
	"github.com/kubeedge/kubeedge/cloud/pkg/dynamiccontroller/application"
	"github.com/kubeedge/kubeedge/cloud/pkg/dynamiccontroller/filter"
)

// FilterImpl implement enpointslice filter
type FilterImpl struct {
	NodesInformer    *application.CommonResourceEventHandler
	ServicesInformer *application.CommonResourceEventHandler
}

const (
	resourceEpSliceName = "EndpointSlice"
	resourceEpName      = "Endpoints"
	filterName          = "EndpointResource"
)

func newEndpointsliceFilter() *FilterImpl {
	return &FilterImpl{}
}

func Register() {
	filter.Register(newEndpointsliceFilter())
}

func (f *FilterImpl) Name() string {
	return filterName
}

func (f *FilterImpl) NeedFilter(content interface{}) bool {
	if objList, ok := content.(*unstructured.UnstructuredList); ok {
		if len(objList.Items) != 0 && (objList.Items[0].GetObjectKind().GroupVersionKind().Kind == resourceEpSliceName ||
			objList.Items[0].GetObjectKind().GroupVersionKind().Kind == resourceEpName) {
			return true
		}
		return false
	}
	if obj, ok := content.(*unstructured.Unstructured); ok {
		if obj.GetObjectKind().GroupVersionKind().Kind == resourceEpSliceName ||
			obj.GetObjectKind().GroupVersionKind().Kind == resourceEpName {
			return true
		}
	}
	return false
}

func filterEndpointSlice(targetNode string, obj runtime.Object) {
	unstruct, ok := obj.(*unstructured.Unstructured)
	if !ok {
		return
	}
	var epSlice discovery.EndpointSlice
	err := runtime.DefaultUnstructuredConverter.FromUnstructured(unstruct.UnstructuredContent(), &epSlice)
	if err != nil {
		klog.Errorf("convert unstructure content %v err: %v", unstruct.GetName(), err)
		return
	}
	var svcTopology string
	if svcName, ok := epSlice.Labels[discovery.LabelServiceName]; ok {
		svcRaw, err := filter.GetDynamicResourceInformer(v1.SchemeGroupVersion.WithResource("services")).Lister().ByNamespace(epSlice.Namespace).Get(svcName)
		if err != nil {
			klog.Errorf("filter endpoint slice for svc %s error: %v", svcName, err)
			return
		}
		svcObj, err := meta.Accessor(svcRaw)
		if err != nil {
			klog.Errorf("get service %v accessor error: %v", svcName, err)
			return
		}
		svcTopology = svcObj.GetAnnotations()[nodegroup.ServiceTopologyAnnotation]
	}
	if svcTopology != nodegroup.ServiceTopologyRangeNodegroup {
		klog.V(4).Info("skip filter for endpointSlice %v", unstruct.GetName())
		return
	}
	var epsTmp []discovery.Endpoint
	for _, ep := range epSlice.Endpoints {
		if filter.IsBelongToSameGroup(targetNode, *ep.NodeName) {
			epsTmp = append(epsTmp, ep)
		}
	}
	epSlice.Endpoints = epsTmp
	unstrRaw, err := runtime.DefaultUnstructuredConverter.ToUnstructured(&epSlice)
	if err != nil {
		klog.Errorf("endpointslice %v convert to unstructure error: %v", epSlice.Name, err)
		return
	}
	unstruct.SetUnstructuredContent(unstrRaw)
}

func filterEndpointsAddress(targetNode string, address []v1.EndpointAddress) []v1.EndpointAddress {
	var tmpAddress []v1.EndpointAddress
	for _, addr := range address {
		if addr.NodeName == nil {
			continue
		}
		if filter.IsBelongToSameGroup(targetNode, *addr.NodeName) {
			tmpAddress = append(tmpAddress, addr)
		}
	}
	return tmpAddress
}

func filterEndpoints(targetNode string, obj runtime.Object) {
	unstruct, ok := obj.(*unstructured.Unstructured)
	if !ok {
		return
	}
	var ep v1.Endpoints
	err := runtime.DefaultUnstructuredConverter.FromUnstructured(unstruct.UnstructuredContent(), &ep)
	if err != nil {
		klog.Errorf("convert unstructure content %v err: %v", unstruct.GetName(), err)
		return
	}
	svcName := ep.GetName()
	svcRaw, err := filter.GetDynamicResourceInformer(v1.SchemeGroupVersion.WithResource("services")).Lister().ByNamespace(ep.Namespace).Get(svcName)
	if err != nil {
		klog.Errorf("filter endpoint slice for svc %s error: %v", svcName, err)
		return
	}
	svcObj, err := meta.Accessor(svcRaw)
	if err != nil {
		klog.Errorf("get service %v accessor error: %v", svcName, err)
		return
	}

	if svcObj.GetAnnotations()[nodegroup.ServiceTopologyAnnotation] != nodegroup.ServiceTopologyRangeNodegroup {
		klog.V(4).Info("skip filter for endpointSlice %v", unstruct.GetName())
		return
	}
	for i := range ep.Subsets {
		ep.Subsets[i].Addresses = filterEndpointsAddress(targetNode, ep.Subsets[i].Addresses)
		ep.Subsets[i].NotReadyAddresses = filterEndpointsAddress(targetNode, ep.Subsets[i].NotReadyAddresses)
	}
	unstrRaw, err := runtime.DefaultUnstructuredConverter.ToUnstructured(&ep)
	if err != nil {
		klog.Errorf("endpointslice %v convert to unstructure error: %v", ep.Name, err)
		return
	}
	unstruct.SetUnstructuredContent(unstrRaw)
}

func (f *FilterImpl) FilterResource(targetNode string, obj runtime.Object) {
	if obj.GetObjectKind().GroupVersionKind().Kind == resourceEpSliceName {
		filterEndpointSlice(targetNode, obj)
	} else if obj.GetObjectKind().GroupVersionKind().Kind == resourceEpName {
		filterEndpoints(targetNode, obj)
	}
}
