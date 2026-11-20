package endpointresource

import (
	"context"
	"fmt"
	"sync"

	v1 "k8s.io/api/core/v1"
	discovery "k8s.io/api/discovery/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/klog/v2"

	"github.com/kubeedge/kubeedge/cloud/pkg/common/client"
	"github.com/kubeedge/kubeedge/cloud/pkg/controllermanager/nodegroup"
	"github.com/kubeedge/kubeedge/cloud/pkg/dynamiccontroller/application"
	"github.com/kubeedge/kubeedge/cloud/pkg/dynamiccontroller/filter"
)

// FilterImpl implement enpointslice filter
type FilterImpl struct {
	NodesInformer    *application.CommonResourceEventHandler
	ServicesInformer *application.CommonResourceEventHandler
	// defaultTopology applies to Services without a service-topology annotation.
	defaultTopology string
}

const (
	resourceEpSliceName = "EndpointSlice"
	resourceEpName      = "Endpoints"
	filterName          = "EndpointResource"
)

func newEndpointsliceFilter(defaultTopology string) *FilterImpl {
	return &FilterImpl{defaultTopology: defaultTopology}
}

func Register(defaultTopology string) {
	filter.Register(newEndpointsliceFilter(defaultTopology))
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

// getService resolves the Service owning an endpoint object, preferring the
// shared informer cache and falling back to a live GET while it is unavailable
// or does not have the Service yet.
func getService(namespace, name string) (*v1.Service, error) {
	servicesGVR := v1.SchemeGroupVersion.WithResource("services")
	liveGet := func() (runtime.Object, error) {
		return client.GetDynamicClient().Resource(servicesGVR).Namespace(namespace).
			Get(context.TODO(), name, metav1.GetOptions{})
	}

	var svcRaw runtime.Object
	lister, err := filter.GetSyncedResourceLister(servicesGVR)
	if err != nil {
		klog.Infof("services lister unavailable, falling back to live get: %v", err)
		svcRaw, err = liveGet()
	} else {
		svcRaw, err = lister.ByNamespace(namespace).Get(name)
		// The Service and EndpointSlice watches are independent, so a slice can
		// arrive before the cache has its Service, for example when a Service is
		// created over pods that already run. Ask the API server before giving up.
		if apierrors.IsNotFound(err) {
			klog.V(4).Infof("service %s/%s not in cache yet, falling back to live get", namespace, name)
			svcRaw, err = liveGet()
		}
	}
	if err != nil {
		return nil, err
	}
	switch svc := svcRaw.(type) {
	case *v1.Service:
		return svc, nil
	case *unstructured.Unstructured:
		var typed v1.Service
		if err := runtime.DefaultUnstructuredConverter.FromUnstructured(svc.UnstructuredContent(), &typed); err != nil {
			return nil, err
		}
		return &typed, nil
	}
	return nil, fmt.Errorf("unexpected type %T for service %s/%s", svcRaw, namespace, name)
}

// unknownTopologyLogged holds the Services already logged for an unknown
// service-topology value, keyed by namespace/name=value. isScoped runs for
// every edge node on every event, so each value is logged once, not each time.
var unknownTopologyLogged sync.Map

// isScoped reports whether svc's endpoints are scoped to the receiving node's
// NodeGroup. The annotation wins; without it the default applies, but only to
// Services with a selector, whose endpoints carry the node they run on. An
// unknown annotation value is treated as no annotation.
func isScoped(svc *v1.Service, defaultTopology string) bool {
	switch value := svc.Annotations[nodegroup.ServiceTopologyAnnotation]; value {
	case nodegroup.ServiceTopologyRangeNodegroup:
		return true
	case nodegroup.ServiceTopologyRangeCluster:
		return false
	case "":
	default:
		key := svc.Namespace + "/" + svc.Name + "=" + value
		if _, logged := unknownTopologyLogged.LoadOrStore(key, struct{}{}); !logged {
			klog.Warningf("service %s/%s has unknown %s value %q, treating it as unset (defaultServiceTopology %q)",
				svc.Namespace, svc.Name, nodegroup.ServiceTopologyAnnotation, value, defaultTopology)
		}
	}
	return defaultTopology == nodegroup.ServiceTopologyRangeNodegroup && len(svc.Spec.Selector) > 0
}

// keepEndpointsOn returns a function that reports whether targetNode keeps an
// endpoint running on a given node, or nil when targetNode gets every endpoint
// of Service namespace/svcName.
//
// When the Service cannot be looked up and defaultTopology is range-nodegroup,
// targetNode keeps only its own endpoints: sending them all would show every
// NodeGroup's endpoints to every edge node. default/kubernetes is the exception.
// Its endpoints have no node, so this would empty them, and the defaultMaster
// filter needs them. With the default unset, the Service was most likely not
// scoped, so its endpoints are sent unfiltered.
func keepEndpointsOn(targetNode, namespace, svcName, defaultTopology string) func(nodeName string) bool {
	svc, err := getService(namespace, svcName)
	if err != nil {
		if defaultTopology != nodegroup.ServiceTopologyRangeNodegroup ||
			(namespace == metav1.NamespaceDefault && svcName == "kubernetes") {
			klog.Errorf("failed to get service %s/%s, sending its endpoints to node %s unfiltered: %v",
				namespace, svcName, targetNode, err)
			return nil
		}
		klog.Errorf("failed to get service %s/%s, sending node %s only its own endpoints: %v",
			namespace, svcName, targetNode, err)
		return func(nodeName string) bool { return nodeName == targetNode }
	}
	if !isScoped(svc, defaultTopology) {
		return nil
	}
	return filter.SameGroupAs(targetNode)
}

func filterEndpointSlice(targetNode string, obj runtime.Object, defaultTopology string) {
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
	var keep func(nodeName string) bool
	if svcName, ok := epSlice.Labels[discovery.LabelServiceName]; ok {
		keep = keepEndpointsOn(targetNode, epSlice.Namespace, svcName, defaultTopology)
	}
	if keep == nil {
		klog.V(4).Infof("skip filter for endpointSlice %v", unstruct.GetName())
		return
	}
	var epsTmp []discovery.Endpoint
	for _, ep := range epSlice.Endpoints {
		// nodeName is optional and is routinely absent while the backing pod is
		// being scheduled. Such an endpoint cannot be matched to a node group,
		// so it is skipped instead of dereferenced, as filterEndpointsAddress
		// already does for the Endpoints resource.
		if ep.NodeName == nil {
			klog.V(4).Infof("skip endpoint without nodeName in endpointSlice %v", unstruct.GetName())
			continue
		}
		if keep(*ep.NodeName) {
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

func filterEndpointsAddress(keep func(nodeName string) bool, address []v1.EndpointAddress) []v1.EndpointAddress {
	var tmpAddress []v1.EndpointAddress
	for _, addr := range address {
		if addr.NodeName == nil {
			continue
		}
		if keep(*addr.NodeName) {
			tmpAddress = append(tmpAddress, addr)
		}
	}
	return tmpAddress
}

// filterEndpoints filters a core/v1 Endpoints object as it is sent. Unlike an
// EndpointSlice, its edge copies are not filtered again when the filter inputs
// change: the refilter controller leaves Endpoints out on purpose (refilter.go).
func filterEndpoints(targetNode string, obj runtime.Object, defaultTopology string) {
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
	keep := keepEndpointsOn(targetNode, ep.Namespace, ep.GetName(), defaultTopology)
	if keep == nil {
		klog.V(4).Infof("skip filter for endpoint %v", unstruct.GetName())
		return
	}
	for i := range ep.Subsets {
		ep.Subsets[i].Addresses = filterEndpointsAddress(keep, ep.Subsets[i].Addresses)
		ep.Subsets[i].NotReadyAddresses = filterEndpointsAddress(keep, ep.Subsets[i].NotReadyAddresses)
	}
	unstrRaw, err := runtime.DefaultUnstructuredConverter.ToUnstructured(&ep)
	if err != nil {
		klog.Errorf("endpoints %v convert to unstructure error: %v", ep.Name, err)
		return
	}
	unstruct.SetUnstructuredContent(unstrRaw)
}

func (f *FilterImpl) FilterResource(targetNode string, obj runtime.Object) {
	if obj.GetObjectKind().GroupVersionKind().Kind == resourceEpSliceName {
		filterEndpointSlice(targetNode, obj, f.defaultTopology)
	} else if obj.GetObjectKind().GroupVersionKind().Kind == resourceEpName {
		filterEndpoints(targetNode, obj, f.defaultTopology)
	}
}
