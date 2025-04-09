package synccontroller

import (
	"context"
	"strings"

	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/api/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/klog/v2"

	"github.com/kubeedge/api/apis/reliablesyncs/v1alpha1"
	beehiveContext "github.com/kubeedge/beehive/pkg/core/context"
	"github.com/kubeedge/beehive/pkg/core/model"
	commonconst "github.com/kubeedge/kubeedge/common/constants"
	v2 "github.com/kubeedge/kubeedge/edge/pkg/metamanager/dao/v2"
	"github.com/kubeedge/kubeedge/pkg/metaserver/util"
)

var (
	sendToEdge = beehiveContext.Send

	buildEdgeControllerMessageFunc = buildEdgeControllerMessage

	getNodeNameFunc = getNodeName

	getObjectUIDFunc = getObjectUID

	compareResourceVersionFunc = CompareResourceVersion

	gcOrphanedClusterObjectSyncFunc = func(sctl *SyncController, sync *v1alpha1.ClusterObjectSync) {
		sctl.gcOrphanedClusterObjectSyncImpl(sync)
	}

	deleteClusterObjectSyncFunc = func(sctl *SyncController, name string) error {
		return sctl.crdclient.ReliablesyncsV1alpha1().ClusterObjectSyncs().Delete(context.Background(), name, *metav1.NewDeleteOptions(0))
	}
)

func (sctl *SyncController) reconcileClusterObjectSync(sync *v1alpha1.ClusterObjectSync) {
	var object metav1.Object

	gv, err := schema.ParseGroupVersion(sync.Spec.ObjectAPIVersion)
	if err != nil {
		return
	}
	resource := util.UnsafeKindToResource(sync.Spec.ObjectKind)
	gvr := gv.WithResource(resource)
	nodeName := getNodeNameFunc(sync.Name)
	resourceType := strings.ToLower(sync.Spec.ObjectKind)

	lister, err := sctl.informerManager.GetLister(gvr)
	if err != nil {
		return
	}

	ret, err := lister.Get(sync.Spec.ObjectName)
	if apierrors.IsNotFound(err) {
		sctl.gcOrphanedClusterObjectSync(sync)
		return
	}

	if err != nil || ret == nil {
		klog.Errorf("failed to get obj(gvr:%v, name:%v): %v", gvr, sync.Spec.ObjectName, err)
		return
	}

	object, err = meta.Accessor(ret)
	if err != nil {
		klog.Errorf("failed to Accessor obj(gvr:%v, name:%v): %v", gvr, sync.Spec.ObjectName, err)
		return
	}

	syncObjUID := getObjectUIDFunc(sync.Name)
	if syncObjUID != string(object.GetUID()) {
		sctl.gcOrphanedClusterObjectSync(sync)
		return
	}

	sendClusterObjectSyncEvent(nodeName, sync, resourceType, object.GetResourceVersion(), object)
}

func (sctl *SyncController) gcOrphanedClusterObjectSync(sync *v1alpha1.ClusterObjectSync) {
	gcOrphanedClusterObjectSyncFunc(sctl, sync)
}

// gcOrphanedClusterObjectSyncImpl try to send delete message to the edge node
// to make sure that the resource is deleted in the edge node. After the
// message ACK is received by `cloudHub`, the objectSync will be deleted
// directly in the `cloudHub`. But if message build failed, the ClusterObjectSync
// will be deleted directly in the `syncController`.
func (sctl *SyncController) gcOrphanedClusterObjectSyncImpl(sync *v1alpha1.ClusterObjectSync) {
	resourceType := strings.ToLower(sync.Spec.ObjectKind)
	nodeName := getNodeNameFunc(sync.Name)
	klog.V(4).Infof("%s: %s has been deleted in K8s, send the delete event to edge in sync loop", resourceType, sync.Spec.ObjectName)

	object := &unstructured.Unstructured{}
	object.SetName(sync.Spec.ObjectName)
	object.SetUID(types.UID(getObjectUIDFunc(sync.Name)))
	if msg := buildEdgeControllerMessageFunc(nodeName, v2.NullNamespace, resourceType, sync.Spec.ObjectName, model.DeleteOperation, object); msg != nil {
		sendToEdge(commonconst.DefaultContextSendModuleName, *msg)
	} else {
		if err := deleteClusterObjectSyncFunc(sctl, sync.Name); err != nil {
			klog.Errorf("Failed to delete clusterObjectSync %s: %v", sync.Name, err)
		}
	}
}

func sendClusterObjectSyncEvent(nodeName string, sync *v1alpha1.ClusterObjectSync, resourceType string,
	objectResourceVersion string, obj interface{}) {
	runtimeObj := obj.(runtime.Object)
	if err := util.SetMetaType(runtimeObj); err != nil {
		klog.Warningf("failed to set metatype :%v", err)
	}

	if sync.Status.ObjectResourceVersion == "" {
		klog.Errorf("The ObjectResourceVersion is empty in status of clusterObjectSync: %s", sync.Name)
		return
	}

	if compareResourceVersionFunc(objectResourceVersion, sync.Status.ObjectResourceVersion) > 0 {
		// trigger the update event
		klog.V(4).Infof("The resourceVersion: %s of %s in K8s is greater than in edgenode: %s, send the update event", objectResourceVersion, resourceType, sync.Status.ObjectResourceVersion)
		msg := buildEdgeControllerMessageFunc(nodeName, v2.NullNamespace, resourceType, sync.Spec.ObjectName, model.UpdateOperation, obj)
		sendToEdge(commonconst.DefaultContextSendModuleName, *msg)
	}
}
