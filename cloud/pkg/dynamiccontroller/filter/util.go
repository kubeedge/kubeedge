package filter

import (
	"strings"

	v1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/meta"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/client-go/informers"
	"k8s.io/klog/v2"

	commoninformers "github.com/kubeedge/kubeedge/cloud/pkg/common/informers"
	"github.com/kubeedge/kubeedge/cloud/pkg/controllermanager/nodegroup"
)

func IsBelongToSameGroup(targetNodeName string, epNodeName string) bool {
	if strings.Compare(targetNodeName, epNodeName) == 0 {
		return true
	}
	targetNode, err := GetDynamicResourceInformer(v1.SchemeGroupVersion.WithResource("nodes")).Lister().Get(targetNodeName)
	if err != nil {
		klog.Errorf("node informer get node %s error: %v", targetNodeName, err)
		return false
	}
	targetAccessor, err := meta.Accessor(targetNode)
	if err != nil {
		klog.Error(err)
		return false
	}

	epNode, err := GetDynamicResourceInformer(v1.SchemeGroupVersion.WithResource("nodes")).Lister().Get(epNodeName)
	if err != nil {
		klog.Errorf("node informer get endpoint slice belonging node %s error: %v", epNodeName, err)
		return false
	}
	epNodeAccessor, err := meta.Accessor(epNode)
	if err != nil {
		klog.Error(err)
		return false
	}

	return targetAccessor.GetLabels()[nodegroup.LabelBelongingTo] == epNodeAccessor.GetLabels()[nodegroup.LabelBelongingTo]
}

func GetDynamicResourceInformer(gvr schema.GroupVersionResource) informers.GenericInformer {
	return commoninformers.GetInformersManager().GetDynamicInformerFactory().ForResource(gvr)
}
