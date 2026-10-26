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
	scoped := false
	if svcName, ok := epSlice.Labels[discovery.LabelServiceName]; ok {
		svcObj, err := getService(epSlice.Namespace, svcName)
		if err != nil {
			klog.Errorf("filter endpoint slice for svc %s error: %v", svcName, err)
			return
		}
		scoped = isScoped(svcObj, defaultTopology)
	}
	if !scoped {
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
	svcName := ep.GetName()
	svcObj, err := getService(ep.Namespace, svcName)
	if err != nil {
		klog.Errorf("filter endpoint for svc %s error: %v", svcName, err)
		return
	}

	if !isScoped(svcObj, defaultTopology) {
		klog.V(4).Infof("skip filter for endpoint %v", unstruct.GetName())
		return
	}
	for i := range ep.Subsets {
		ep.Subsets[i].Addresses = filterEndpointsAddress(targetNode, ep.Subsets[i].Addresses)
		ep.Subsets[i].NotReadyAddresses = filterEndpointsAddress(targetNode, ep.Subsets[i].NotReadyAddresses)
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
