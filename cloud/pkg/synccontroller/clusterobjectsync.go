/*
Copyright 2021 The KubeEdge Authors.

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
	"github.com/kubeedge/kubeedge/edge/pkg/metamanager/dao/models"
	"github.com/kubeedge/kubeedge/pkg/metaserver/util"
)

var (
	sendToEdge = beehiveContext.Send

	buildEdgeControllerMessageFunc = buildEdgeControllerMessage

	getNodeNameFunc = getNodeName

	getObjectUIDFunc = getObjectUID

	// compareResourceVersionFunc is a variable to allow test injection.
	// Its signature matches CompareResourceVersion.
	compareResourceVersionFunc func(string, string) (int, error) = CompareResourceVersion

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
// message ACK is received by `cloudHub`, the ClusterObjectSync will be deleted
// directly in the `cloudHub`. But if message build failed, the ClusterObjectSync
// will be deleted directly in the `syncController`.
func (sctl *SyncController) gcOrphanedClusterObjectSyncImpl(sync *v1alpha1.ClusterObjectSync) {
	resourceType := strings.ToLower(sync.Spec.ObjectKind)
	nodeName := getNodeNameFunc(sync.Name)
	klog.V(4).Infof("%s: %s has been deleted in K8s, send the delete event to edge in sync loop", resourceType, sync.Spec.ObjectName)

	object := &unstructured.Unstructured{}
	object.SetName(sync.Spec.ObjectName)
	object.SetUID(types.UID(getObjectUIDFunc(sync.Name)))
	if msg := buildEdgeControllerMessageFunc(nodeName, models.NullNamespace, resourceType, sync.Spec.ObjectName, model.DeleteOperation, object); msg != nil {
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

	cmp, err := compareResourceVersionFunc(objectResourceVersion, sync.Status.ObjectResourceVersion)
	if err != nil {
		// ResourceVersion is opaque per the Kubernetes API spec; non-integer values are valid
		// for non-standard API servers. Treat an unparseable version as "unknown — send the
		// update" so the edge stays consistent rather than silently stalling forever.
		klog.Warningf("clusterObjectSync %s: cannot compare resourceVersions (live=%q edge=%q): %v; sending update as fail-safe",
			sync.Name, objectResourceVersion, sync.Status.ObjectResourceVersion, err)
		cmp = 1
	}
	if cmp > 0 {
		klog.V(4).Infof("The resourceVersion: %s of %s in K8s is greater than in edgenode: %s, send the update event",
			objectResourceVersion, resourceType, sync.Status.ObjectResourceVersion)
		msg := buildEdgeControllerMessageFunc(nodeName, models.NullNamespace, resourceType, sync.Spec.ObjectName, model.UpdateOperation, obj)
		sendToEdge(commonconst.DefaultContextSendModuleName, *msg)
	}
}
