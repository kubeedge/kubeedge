package synccontroller

import (
	"strconv"

	v1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/api/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/klog/v2"

	beehiveContext "github.com/kubeedge/beehive/pkg/core/context"
	"github.com/kubeedge/beehive/pkg/core/model"
	"github.com/kubeedge/kubeedge/cloud/pkg/apis/reliablesyncs/v1alpha1"
	"github.com/kubeedge/kubeedge/cloud/pkg/common/modules"
	edgectrconst "github.com/kubeedge/kubeedge/cloud/pkg/edgecontroller/constants"
	edgectrmessagelayer "github.com/kubeedge/kubeedge/cloud/pkg/edgecontroller/messagelayer"
	commonconst "github.com/kubeedge/kubeedge/common/constants"
)

func (sctl *SyncController) managePod(sync *v1alpha1.ObjectSync) {
	pod, err := sctl.podLister.Pods(sync.Namespace).Get(sync.Spec.ObjectName)

	nodeName := getNodeName(sync.Name)
	if pod != nil {
		syncObjUID := getObjectUID(sync.Name)
		if syncObjUID != string(pod.GetUID()) {
			err = apierrors.NewNotFound(schema.GroupResource{
				Group:    "",
				Resource: "pods",
			}, sync.Spec.ObjectName)
		}
	}

	if err != nil {
		if apierrors.IsNotFound(err) {
			pod = &v1.Pod{
				ObjectMeta: metav1.ObjectMeta{
					Name:      sync.Spec.ObjectName,
					Namespace: sync.Namespace,
					UID:       types.UID(getObjectUID(sync.Name)),
				},
			}
		} else {
			klog.Errorf("Failed to manage pod sync of %s in namespace %s: %v", sync.Name, sync.Namespace, err)
			return
		}
	}
	sendEvents(err, nodeName, sync, model.ResourceTypePod, pod.ResourceVersion, pod)
}

func (sctl *SyncController) manageConfigMap(sync *v1alpha1.ObjectSync) {
	configmap, err := sctl.configMapLister.ConfigMaps(sync.Namespace).Get(sync.Spec.ObjectName)

	nodeName := getNodeName(sync.Name)
	if configmap != nil {
		syncObjUID := getObjectUID(sync.Name)
		if syncObjUID != string(configmap.GetUID()) {
			err = apierrors.NewNotFound(schema.GroupResource{
				Group:    "",
				Resource: "configmaps",
			}, sync.Spec.ObjectName)
		}
	}

	if err != nil {
		if apierrors.IsNotFound(err) {
			configmap = &v1.ConfigMap{
				ObjectMeta: metav1.ObjectMeta{
					Name:      sync.Spec.ObjectName,
					Namespace: sync.Namespace,
					UID:       types.UID(getObjectUID(sync.Name)),
				},
			}
		} else {
			klog.Errorf("Failed to manage configMap sync of %s in namespace %s: %v", sync.Name, sync.Namespace, err)
			return
		}
	}
	sendEvents(err, nodeName, sync, model.ResourceTypeConfigmap, configmap.ResourceVersion, configmap)
}

func (sctl *SyncController) manageSecret(sync *v1alpha1.ObjectSync) {
	secret, err := sctl.secretLister.Secrets(sync.Namespace).Get(sync.Spec.ObjectName)

	nodeName := getNodeName(sync.Name)

	if secret != nil {
		syncObjUID := getObjectUID(sync.Name)
		if syncObjUID != string(secret.GetUID()) {
			err = apierrors.NewNotFound(schema.GroupResource{
				Group:    "",
				Resource: "secrets",
			}, sync.Spec.ObjectName)
		}
	}

	if err != nil {
		if apierrors.IsNotFound(err) {
			secret = &v1.Secret{
				ObjectMeta: metav1.ObjectMeta{
					Name:      sync.Spec.ObjectName,
					Namespace: sync.Namespace,
					UID:       types.UID(getObjectUID(sync.Name)),
				},
			}
		} else {
			klog.Errorf("Failed to manage secret sync of %s in namespace %s: %v", sync.Name, sync.Namespace, err)
			return
		}
	}
	sendEvents(err, nodeName, sync, model.ResourceTypeSecret, secret.ResourceVersion, secret)
}

func (sctl *SyncController) manageService(sync *v1alpha1.ObjectSync) {
	service, err := sctl.serviceLister.Services(sync.Namespace).Get(sync.Spec.ObjectName)

	nodeName := getNodeName(sync.Name)
	if service != nil {
		syncObjUID := getObjectUID(sync.Name)
		if syncObjUID != string(service.GetUID()) {
			err = apierrors.NewNotFound(schema.GroupResource{
				Group:    "",
				Resource: "services",
			}, sync.Spec.ObjectName)
		}
	}

	if err != nil {
		if apierrors.IsNotFound(err) {
			service = &v1.Service{
				ObjectMeta: metav1.ObjectMeta{
					Name:      sync.Spec.ObjectName,
					Namespace: sync.Namespace,
					UID:       types.UID(getObjectUID(sync.Name)),
				},
			}
		} else {
			klog.Errorf("Failed to manage service sync of %s in namespace %s: %v", sync.Name, sync.Namespace, err)
			return
		}
	}
	sendEvents(err, nodeName, sync, commonconst.ResourceTypeService, service.ResourceVersion, service)
}

func (sctl *SyncController) manageEndpoint(sync *v1alpha1.ObjectSync) {
	endpoint, err := sctl.endpointLister.Endpoints(sync.Namespace).Get(sync.Spec.ObjectName)

	nodeName := getNodeName(sync.Name)
	
	if endpoint != nil {
		syncObjUID := getObjectUID(sync.Name)
		if syncObjUID != string(endpoint.GetUID()) {
			err = apierrors.NewNotFound(schema.GroupResource{
				Group:    "",
				Resource: "endpoints",
			}, sync.Spec.ObjectName)
		}
	}

	if err != nil {
		if apierrors.IsNotFound(err) {
			endpoint = &v1.Endpoints{
				ObjectMeta: metav1.ObjectMeta{
					Name:      sync.Spec.ObjectName,
					Namespace: sync.Namespace,
					UID:       types.UID(getObjectUID(sync.Name)),
				},
			}
		} else {
			klog.Errorf("Failed to manage endpoint sync of %s in namespace %s: %v", sync.Name, sync.Namespace, err)
			return
		}
	}
	sendEvents(err, nodeName, sync, commonconst.ResourceTypeEndpoints, endpoint.ResourceVersion, endpoint)
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

func sendEvents(err error, nodeName string, sync *v1alpha1.ObjectSync, resourceType string,
	objectResourceVersion string, obj interface{}) {
	if err != nil && apierrors.IsNotFound(err) {
		//trigger the delete event
		klog.Infof("%s: %s has been deleted in K8s, send the delete event to edge", resourceType, sync.Spec.ObjectName)
		msg := buildEdgeControllerMessage(nodeName, sync.Namespace, resourceType, sync.Spec.ObjectName, model.DeleteOperation, obj)
		beehiveContext.Send(commonconst.DefaultContextSendModuleName, *msg)
		return
	}

	if sync.Status.ObjectResourceVersion == "" {
		klog.Errorf("The ObjectResourceVersion is empty in status of objectsync: %s", sync.Name)
		return
	}

	if CompareResourceVersion(objectResourceVersion, sync.Status.ObjectResourceVersion) > 0 {
		// trigger the update event
		klog.V(4).Infof("The resourceVersion: %s of %s in K8s is greater than in edgenode: %s, send the update event", objectResourceVersion, resourceType, sync.Status.ObjectResourceVersion)
		msg := buildEdgeControllerMessage(nodeName, sync.Namespace, resourceType, sync.Spec.ObjectName, model.UpdateOperation, obj)
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
	msg.BuildRouter(modules.EdgeControllerModuleName, edgectrconst.GroupResource, resource, operationType)
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
