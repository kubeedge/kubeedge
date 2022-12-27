package filter

import (
	"strings"

	v1 "k8s.io/api/core/v1"
	"k8s.io/client-go/tools/cache"
	"k8s.io/klog/v2"

	"github.com/kubeedge/kubeedge/cloud/pkg/controllermanager/nodegroup"
)

func IsBelongToSameGroup(targetNodeName, epNodeName string, nodeLister cache.GenericLister) bool {
	if strings.Compare(targetNodeName, epNodeName) == 0 {
		return true
	}

	targetNode, err := getNode(targetNodeName, nodeLister)
	if err != nil {
		klog.Errorf("node lister get node %s error: %v", targetNodeName, err)
		return false
	}

	epNode, err := getNode(epNodeName, nodeLister)
	if err != nil {
		klog.Errorf("node lister get endpoint slice belonging node %s error: %v", epNodeName, err)
		return false
	}

	return targetNode.GetLabels()[nodegroup.LabelBelongingTo] == epNode.GetLabels()[nodegroup.LabelBelongingTo]
}

func getNode(name string, nodeLister cache.GenericLister) (*v1.Node, error) {
	ret, err := nodeLister.Get(name)
	if err != nil {
		return nil, err
	}
	return ret.(*v1.Node), err
}
