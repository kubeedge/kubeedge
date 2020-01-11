package synccontroller

import (
	"strconv"

	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/api/meta"
	"k8s.io/klog"

	beehiveContext "github.com/kubeedge/beehive/pkg/core/context"
	"github.com/kubeedge/beehive/pkg/core/model"
	"github.com/kubeedge/kubeedge/cloud/pkg/apis/reliablesyncs/v1alpha1"
	edgectrconst "github.com/kubeedge/kubeedge/cloud/pkg/edgecontroller/constants"
	edgectrmessagelayer "github.com/kubeedge/kubeedge/cloud/pkg/edgecontroller/messagelayer"
	commonconst "github.com/kubeedge/kubeedge/common/constants"
)

func (sctl *SyncController) managePod(sync *v1alpha1.ObjectSync) {
	pod, err := sctl.podLister.Pods(sync.Namespace).Get(sync.Spec.ObjectName)

	nodeName := getNodeName(sync.Name)

	if err != nil && apierrors.IsNotFound(err) {
		sendEvents(err, nodeName, sync.Namespace, sync.Spec.ObjectName, model.ResourceTypePod,
			"", "", nil)
		return
	}
	sendEvents(err, nodeName, sync.Namespace, sync.Spec.ObjectName, model.ResourceTypePod,
		pod.ResourceVersion, sync.Status.ObjectResourceVersion, pod)
}

func (sctl *SyncController) manageConfigMap(sync *v1alpha1.ObjectSync) {
	configmap, err := sctl.configMapLister.ConfigMaps(sync.Namespace).Get(sync.Spec.ObjectName)

	nodeName := getNodeName(sync.Name)

	if err != nil && apierrors.IsNotFound(err) {
		sendEvents(err, nodeName, sync.Namespace, sync.Spec.ObjectName, model.ResourceTypeConfigmap,
			"", "", nil)
		return
	}
	sendEvents(err, nodeName, sync.Namespace, sync.Spec.ObjectName, model.ResourceTypeConfigmap,
		configmap.ResourceVersion, sync.Status.ObjectResourceVersion, configmap)
}
func (sctl *SyncController) manageSecret(sync *v1alpha1.ObjectSync) {
	secret, err := sctl.secretLister.Secrets(sync.Namespace).Get(sync.Spec.ObjectName)

	nodeName := getNodeName(sync.Name)

	if err != nil && apierrors.IsNotFound(err) {
		sendEvents(err, nodeName, sync.Namespace, sync.Spec.ObjectName, model.ResourceTypeSecret,
			"", "", nil)
		return
	}
	sendEvents(err, nodeName, sync.Namespace, sync.Spec.ObjectName, model.ResourceTypeSecret,
		secret.ResourceVersion, sync.Status.ObjectResourceVersion, secret)
}

func (sctl *SyncController) manageService(sync *v1alpha1.ObjectSync) {
	service, err := sctl.serviceLister.Services(sync.Namespace).Get(sync.Spec.ObjectName)

	nodeName := getNodeName(sync.Name)

	if err != nil && apierrors.IsNotFound(err) {
		sendEvents(err, nodeName, sync.Namespace, sync.Spec.ObjectName, commonconst.ResourceTypeService,
			"", "", nil)
		return
	}
	sendEvents(err, nodeName, sync.Namespace, sync.Spec.ObjectName, commonconst.ResourceTypeService,
		service.ResourceVersion, sync.Status.ObjectResourceVersion, service)
}

func (sctl *SyncController) manageEndpoint(sync *v1alpha1.ObjectSync) {
	endpoint, err := sctl.endpointLister.Endpoints(sync.Namespace).Get(sync.Spec.ObjectName)

	nodeName := getNodeName(sync.Name)

	if err != nil && apierrors.IsNotFound(err) {
		sendEvents(err, nodeName, sync.Namespace, sync.Spec.ObjectName, commonconst.ResourceTypeEndpoints,
			"", "", nil)
		return
	}
	sendEvents(err, nodeName, sync.Namespace, sync.Spec.ObjectName, commonconst.ResourceTypeEndpoints,
		endpoint.ResourceVersion, sync.Status.ObjectResourceVersion, endpoint)
}

// todo: add events for devices
func (sctl *SyncController) manageDevice(sync *v1alpha1.ObjectSync) {
	//pod, err := sctl.deviceLister.Devices(sync.Namespace).Get(sync.Spec.ObjectName)

	//
	//if err != nil && apierrors.IsNotFound(err) {
	//trigger the delete event
	//}

	//if pod.ResourceVersion > sync.Status.ObjectResourceVersion {
	// trigger the update event
	//}
}

func sendEvents(err error, nodeName, namespace, objectName, resourceType string,
	objectResourceVersion, syncResourceVersion string,
	obj interface{}) {

	if err != nil && apierrors.IsNotFound(err) {
		//trigger the delete event
		klog.Infof("%s: %s has been deleted in K8s, send the delete event to edge", resourceType, objectName)
		msg := buildEdgeControllerMessage(nodeName, namespace, resourceType, objectName, model.DeleteOperation, nil)
		beehiveContext.Send(commonconst.DefaultContextSendModuleName, *msg)
		return
	}

	if CompareResourceVersion(objectResourceVersion, syncResourceVersion) > 0 {
		// trigger the update event
		klog.Infof("The resourceVersion: %s of %s in K8s is greater than in edgenode: %s, send the update event", objectResourceVersion, resourceType, syncResourceVersion)
		msg := buildEdgeControllerMessage(nodeName, namespace, model.ResourceTypePod, objectName, model.UpdateOperation, obj)
		beehiveContext.Send(commonconst.DefaultContextSendModuleName, *msg)
	}
}

func buildEdgeControllerMessage(nodeName, namespace, resourceType, resourceName, operationType string, obj interface{}) *model.Message {
	msg := model.NewMessage("")
	resource, err := edgectrmessagelayer.BuildResource(nodeName, namespace, resourceType, resourceName)
	if err != nil {
		klog.Warningf("build message resource failed with error: %s", err)
		return nil
	}
	msg.BuildRouter(edgectrconst.EdgeControllerModuleName, edgectrconst.GroupResource, resource, operationType)
	msg.Content = obj

	resourceVersion := GetObjectResourceVersion(obj)
	msg.SetResourceVersion(resourceVersion)

	return msg
}

// GetMessageUID returns the resourceVersion of the object in message
func GetObjectResourceVersion(obj interface{}) string {
	if obj == nil {
		klog.Error("object is nil")
		return ""
	}

	accessor, err := meta.Accessor(obj)
	if err != nil {
		klog.Errorf("Failed to get resourceVersion of the object: %v", obj)
		return ""
	}

	return accessor.GetResourceVersion()
}

// CompareResourceVersion compares resourceversions, resource versions are actually
// ints, so we can easily compare them.
// If rva>rvb, return 1; rva=rvb, return 0; rva<rvb, return -1
func CompareResourceVersion(rva, rvb string) int {
	a, err := strconv.ParseUint(rva, 10, 64)
	if err != nil {
		// coder error
		panic(err)
	}
	b, err := strconv.ParseUint(rvb, 10, 64)
	if err != nil {
		// coder error
		panic(err)
	}

	if a > b {
		return 1
	}
	if a == b {
		return 0
	}
	return -1
}
