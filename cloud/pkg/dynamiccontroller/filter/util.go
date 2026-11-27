package filter

import (
	"context"
	"fmt"

	v1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/client-go/informers"
	"k8s.io/client-go/tools/cache"
	"k8s.io/klog/v2"

	"github.com/kubeedge/kubeedge/cloud/pkg/common/client"
	commoninformers "github.com/kubeedge/kubeedge/cloud/pkg/common/informers"
	"github.com/kubeedge/kubeedge/cloud/pkg/controllermanager/nodegroup"
)

// SameGroupAs returns a function that reports whether an endpoint's node is in
// the same NodeGroup as targetNodeName. The nodes lister and the target node are
// looked up once here, so call SameGroupAs once per object, not per endpoint.
//
// The target node always matches itself. A node in no NodeGroup shares a group
// with no other node: "" must not match the "" of every ungrouped cloud node.
func SameGroupAs(targetNodeName string) func(epNodeName string) bool {
	var getNode func(string) (interface{}, error)

	// Define a function to get the node based on whether the informer is synced
	nodesGVR := v1.SchemeGroupVersion.WithResource("nodes")
	if lister, err := GetSyncedResourceLister(nodesGVR); err != nil {
		klog.Infof("nodes lister unavailable, falling back to live get: %v", err)
		getNode = func(nodeName string) (interface{}, error) {
			return client.GetDynamicClient().Resource(nodesGVR).Get(context.TODO(), nodeName, metav1.GetOptions{})
		}
	} else {
		getNode = func(nodeName string) (interface{}, error) {
			return lister.Get(nodeName)
		}
	}

	getGroup := func(nodeName string) (string, bool) {
		node, err := getNode(nodeName)
		if err != nil {
			klog.Errorf("failed to get node %s: %v", nodeName, err)
			return "", false
		}
		accessor, err := meta.Accessor(node)
		if err != nil {
			klog.Error(err)
			return "", false
		}
		return accessor.GetLabels()[nodegroup.LabelBelongingTo], true
	}

	targetGroup, _ := getGroup(targetNodeName)
	return func(epNodeName string) bool {
		if epNodeName == targetNodeName {
			return true
		}
		if targetGroup == "" {
			return false
		}
		epGroup, ok := getGroup(epNodeName)
		return ok && epGroup == targetGroup
	}
}
func GetDynamicResourceInformer(gvr schema.GroupVersionResource) informers.GenericInformer {
	return commoninformers.GetInformersManager().GetDynamicInformerFactory().ForResource(gvr)
}

// GetSyncedResourceLister returns a lister for gvr backed by an informer that
// the informers manager has actually started and synced.
//
// GetDynamicResourceInformer must not be used for built-in resources such as
// services and nodes: the informers manager serves those from the typed
// informer factory (see informers.forResource), so asking the dynamic factory
// for them lazily creates a second informer that nobody ever starts. Its cache
// stays empty for the process lifetime and every lister lookup returns NotFound,
// which silently disables any filter built on top of it.
func GetSyncedResourceLister(gvr schema.GroupVersionResource) (cache.GenericLister, error) {
	pair, err := commoninformers.GetInformersManager().GetInformerPair(gvr)
	if err != nil {
		return nil, err
	}
	if pair == nil {
		return nil, fmt.Errorf("no informer registered for %s", gvr.String())
	}
	if !pair.Informer.HasSynced() {
		return nil, fmt.Errorf("informer for %s has not synced yet", gvr.String())
	}
	return pair.Lister, nil
}
