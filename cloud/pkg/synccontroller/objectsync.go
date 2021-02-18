package synccontroller

import (
	"context"
	"strconv"

	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/api/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime"
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
	"github.com/kubeedge/kubeedge/pkg/metaserver/util"
)

func (sctl *SyncController) manageObject(sync *v1alpha1.ObjectSync) {
	var object metav1.Object

	gv, err := schema.ParseGroupVersion(sync.Spec.ObjectAPIVersion)
	if err != nil {
		return
	}
	gvr := gv.WithResource(sync.Spec.ObjectKind)

	//ret, err := informers.GetInformersManager().GetDynamicSharedInformerFactory().ForResource(gvr).Lister().ByNamespace(sync.Namespace).Get(sync.Spec.ObjectName)
	ret, err := sctl.kubeclient.Resource(gvr).Namespace(sync.Namespace).Get(context.TODO(), sync.Spec.ObjectName, metav1.GetOptions{})
	if err != nil {
		klog.Errorf("failed to get obj(gvr:%v,namespace:%v,name:%v), %v", gvr, sync.Namespace, sync.Spec.ObjectName, err)
		return
	}

	nodeName := getNodeName(sync.Name)
	if ret != nil {
		object, err = meta.Accessor(ret)
		if err != nil {
			return
		}

		syncObjUID := getObjectUID(sync.Name)
		if syncObjUID != string(object.GetUID()) {
			err = apierrors.NewNotFound(schema.GroupResource{
				Group:    "",
				Resource: sync.Spec.ObjectKind,
			}, sync.Spec.ObjectName)
		}
	}

	if err != nil {
		if apierrors.IsNotFound(err) {
			obj := &unstructured.Unstructured{}
			obj.SetName(sync.Spec.ObjectName)
			obj.SetUID(types.UID(getObjectUID(sync.Name)))
			obj.SetNamespace(sync.Namespace)
		} else {
			klog.Errorf("Failed to manage pod sync of %s in namespace %s: %v", sync.Name, sync.Namespace, err)
			return
		}
	}
	sendEvents(err, nodeName, sync, sync.Spec.ObjectKind, object.GetResourceVersion(), object)
}

func sendEvents(err error, nodeName string, sync *v1alpha1.ObjectSync, resourceType string,
	objectResourceVersion string, obj interface{}) {
	runtimeObj := obj.(runtime.Object)
	if err := util.SetMetaType(runtimeObj); err != nil {
		klog.Warningf("failed to set metatype :%v", err)
	}
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
