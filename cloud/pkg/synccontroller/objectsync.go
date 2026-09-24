/*
Copyright 2020 The KubeEdge Authors.

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
	"fmt"
	"strconv"
	"strings"

	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/api/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/klog/v2"

	"github.com/kubeedge/api/apis/reliablesyncs/v1alpha1"
	beehiveContext "github.com/kubeedge/beehive/pkg/core/context"
	"github.com/kubeedge/beehive/pkg/core/model"
	"github.com/kubeedge/kubeedge/cloud/pkg/common/messagelayer"
	"github.com/kubeedge/kubeedge/cloud/pkg/common/modules"
	edgectrconst "github.com/kubeedge/kubeedge/cloud/pkg/edgecontroller/constants"
	commonconst "github.com/kubeedge/kubeedge/common/constants"
	"github.com/kubeedge/kubeedge/pkg/metaserver/util"
)

func (sctl *SyncController) reconcileObjectSync(sync *v1alpha1.ObjectSync) {
	var object metav1.Object

	gv, err := schema.ParseGroupVersion(sync.Spec.ObjectAPIVersion)
	if err != nil {
		return
	}
	resource := util.UnsafeKindToResource(sync.Spec.ObjectKind)
	gvr := gv.WithResource(resource)
	nodeName := getNodeName(sync.Name)
	resourceType := strings.ToLower(sync.Spec.ObjectKind)

	lister, err := sctl.informerManager.GetLister(gvr)
	if err != nil {
		return
	}

	ret, err := lister.ByNamespace(sync.Namespace).Get(sync.Spec.ObjectName)

	if apierrors.IsNotFound(err) {
		sctl.gcOrphanedObjectSync(sync)
		return
	} else if err != nil || ret == nil {
		klog.Errorf("failed to get obj(gvr:%v,namespace:%v,name:%v), %v", gvr, sync.Namespace, sync.Spec.ObjectName, err)
		return
	}

	object, err = meta.Accessor(ret)
	if err != nil {
		return
	}

	syncObjUID := getObjectUID(sync.Name)
	if syncObjUID != string(object.GetUID()) {
		sctl.gcOrphanedObjectSync(sync)
		return
	}

	sendEvents(nodeName, sync, resourceType, object.GetResourceVersion(), object)
}

// gcOrphanedObjectSync try to send delete message to the edge node
// to make sure that the resource is deleted in the edge node. After the
// message ACK is received by `cloudHub`, the objectSync will be deleted
// directly in the `cloudHub`. But if message build failed, the objectSync
// will be deleted directly in the `syncController`.
func (sctl *SyncController) gcOrphanedObjectSync(sync *v1alpha1.ObjectSync) {
	resourceType := strings.ToLower(sync.Spec.ObjectKind)
	nodeName := getNodeName(sync.Name)
	klog.V(4).Infof("%s: %s has been deleted in K8s, send the delete event to edge in sync loop", resourceType, sync.Spec.ObjectName)

	object := &unstructured.Unstructured{}
	object.SetNamespace(sync.Namespace)
	object.SetName(sync.Spec.ObjectName)
	object.SetUID(types.UID(getObjectUID(sync.Name)))
	if msg := buildEdgeControllerMessage(nodeName, sync.Namespace, resourceType, sync.Spec.ObjectName, model.DeleteOperation, object); msg != nil {
		beehiveContext.Send(commonconst.DefaultContextSendModuleName, *msg)
	} else {
		if err := sctl.crdclient.ReliablesyncsV1alpha1().ObjectSyncs(sync.Namespace).Delete(context.Background(), sync.Name, *metav1.NewDeleteOptions(0)); err != nil {
			klog.Errorf("Failed to delete objectsync %s: %v", sync.Name, err)
		}
	}
}

func sendEvents(nodeName string, sync *v1alpha1.ObjectSync, resourceType string,
	objectResourceVersion string, obj interface{}) {
	if sync.Status.ObjectResourceVersion == "" {
		klog.Errorf("The ObjectResourceVersion is empty in status of objectsync: %s", sync.Name)
		return
	}

	cmp, err := CompareResourceVersion(objectResourceVersion, sync.Status.ObjectResourceVersion)
	if err != nil {
		// ResourceVersion is opaque per the Kubernetes API spec; non-integer values are valid
		// for non-standard API servers. Treat an unparseable version as "unknown — send the
		// update" so the edge stays consistent rather than silently stalling forever.
		klog.Warningf("objectsync %s: cannot compare resourceVersions (live=%q edge=%q): %v; sending update as fail-safe",
			sync.Name, objectResourceVersion, sync.Status.ObjectResourceVersion, err)
		cmp = 1
	}
	if cmp > 0 {
		klog.V(4).Infof("The resourceVersion: %s of %s in K8s is greater than in edgenode: %s, send the update event",
			objectResourceVersion, resourceType, sync.Status.ObjectResourceVersion)
		msg := buildEdgeControllerMessage(nodeName, sync.Namespace, resourceType, sync.Spec.ObjectName, model.UpdateOperation, obj)
		beehiveContext.Send(commonconst.DefaultContextSendModuleName, *msg)
	}
}

func buildEdgeControllerMessage(nodeName, namespace, resourceType, resourceName, operationType string, obj interface{}) *model.Message {
	resource, err := messagelayer.BuildResource(nodeName, namespace, resourceType, resourceName)
	if err != nil {
		klog.Warningf("build message resource failed with error: %s", err)
		return nil
	}

	resourceVersion := GetObjectResourceVersion(obj)

	msg := model.NewMessage("").
		BuildRouter(modules.EdgeControllerModuleName, edgectrconst.GroupResource, resource, operationType).
		FillBody(obj).
		SetResourceVersion(resourceVersion)

	return msg
}

// GetObjectResourceVersion returns the resourceVersion of the object in message
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

// CompareResourceVersion compares two resourceVersion strings as uint64 integers.
// Returns 1 if rva > rvb, 0 if equal, -1 if rva < rvb.
// Returns an error when either string is not a valid non-negative integer.
// The Kubernetes API spec treats resourceVersion as an opaque string; the integer
// representation is an implementation detail of kube-apiserver that non-standard
// API servers may not follow. Callers must handle the error explicitly.
func CompareResourceVersion(rva, rvb string) (int, error) {
	a, err := strconv.ParseUint(rva, 10, 64)
	if err != nil {
		return 0, fmt.Errorf("invalid resourceVersion %q: %w", rva, err)
	}
	b, err := strconv.ParseUint(rvb, 10, 64)
	if err != nil {
		return 0, fmt.Errorf("invalid resourceVersion %q: %w", rvb, err)
	}
	if a > b {
		return 1, nil
	}
	if a == b {
		return 0, nil
	}
	return -1, nil
}
