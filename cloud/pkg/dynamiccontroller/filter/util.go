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

func IsBelongToSameGroup(targetNodeName string, epNodeName string) bool {
	// Return true if both node names are the same
	if targetNodeName == epNodeName {
		return true
	}

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

	// Get the target node
	targetNode, err := getNode(targetNodeName)
	if err != nil {
		klog.Errorf("failed to get target node %s: %v", targetNodeName, err)
		return false
	}

	// Get the endpoint node
	epNode, err := getNode(epNodeName)
	if err != nil {
		klog.Errorf("failed to get endpoint node %s: %v", epNodeName, err)
		return false
	}

	targetAccessor, err := meta.Accessor(targetNode)
	if err != nil {
		klog.Error(err)
		return false
	}
	epNodeAccessor, err := meta.Accessor(epNode)
	if err != nil {
		klog.Error(err)
		return false
	}

	// Compare the labels
	return targetAccessor.GetLabels()[nodegroup.LabelBelongingTo] == epNodeAccessor.GetLabels()[nodegroup.LabelBelongingTo]
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
