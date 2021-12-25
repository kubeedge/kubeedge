package edged

import (
	"fmt"
	"sort"

	coordinationv1 "k8s.io/api/coordination/v1"
	v1 "k8s.io/api/core/v1"
	apiequality "k8s.io/apimachinery/pkg/api/equality"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/clock"
	"k8s.io/klog/v2"
	"k8s.io/utils/pointer"

	"github.com/kubeedge/beehive/pkg/core"
	beehiveContext "github.com/kubeedge/beehive/pkg/core/context"
	"github.com/kubeedge/beehive/pkg/core/model"
	"github.com/kubeedge/kubeedge/common/constants"
	"github.com/kubeedge/kubeedge/edge/pkg/common/message"
	"github.com/kubeedge/kubeedge/edge/pkg/common/modules"
	"github.com/kubeedge/kubeedge/edge/pkg/edged/config"
)

var baseLease coordinationv1.Lease

func (e *edged) nodeStatusHasChanged(currentStatus *v1.NodeStatus) bool {
	originalStatusCopy := e.lastReportStatus.DeepCopy()
	currentStatusCopy := currentStatus.DeepCopy()

	// compare NodeStatus.Condition
	if len(originalStatusCopy.Conditions) != len(currentStatusCopy.Conditions) {
		return true
	}
	sort.SliceStable(originalStatusCopy.Conditions, func(i, j int) bool {
		return originalStatusCopy.Conditions[i].Type < originalStatusCopy.Conditions[j].Type
	})
	sort.SliceStable(currentStatusCopy.Conditions, func(i, j int) bool {
		return currentStatusCopy.Conditions[i].Type < currentStatusCopy.Conditions[j].Type
	})
	emptyHeartbeatTime := metav1.Time{}
	for i := range currentStatusCopy.Conditions {
		// ignore difference of LastHeartbeatTime
		originalStatusCopy.Conditions[i].LastHeartbeatTime = emptyHeartbeatTime
		currentStatusCopy.Conditions[i].LastHeartbeatTime = emptyHeartbeatTime
		if !apiequality.Semantic.DeepEqual(originalStatusCopy.Conditions[i], currentStatusCopy.Conditions[i]) {
			return true
		}
	}

	// compare other fields of NodeStatus
	originalStatusCopy.Conditions, currentStatusCopy.Conditions = nil, nil
	return !apiequality.Semantic.DeepEqual(originalStatusCopy, currentStatusCopy)
}

func (e *edged) newLease(base *coordinationv1.Lease) *coordinationv1.Lease {
	var lease *coordinationv1.Lease
	if base == nil {
		lease = &coordinationv1.Lease{
			TypeMeta: metav1.TypeMeta{
				Kind:       "Lease",
				APIVersion: "coordination.k8s.io/v1",
			},
			ObjectMeta: metav1.ObjectMeta{
				Name:      e.nodeName,
				Namespace: v1.NamespaceNodeLease,
			},
			Spec: coordinationv1.LeaseSpec{
				HolderIdentity:       pointer.StringPtr(e.nodeName),
				LeaseDurationSeconds: pointer.Int32(e.nodeLeaseDurationSeconds),
			},
		}
	} else {
		lease = base.DeepCopy()
	}
	lease.Spec.RenewTime = &metav1.MicroTime{Time: clock.RealClock{}.Now()}

	return lease
}

func (e *edged) createNodeLease() error {
	lease := e.newLease(nil)
	if !config.Config.RegisterNode {
		klog.V(2).Info("register-node is set to false, do not create node lease")
		return nil
	}
	klog.Infof("Attempting to create node lease %s", lease.Name)

	resource := fmt.Sprintf("%s/%s/%s", lease.Namespace, model.ResourceTypeLease, lease.Name)
	nodeleaseMsg := message.BuildMsg(modules.MetaGroup, "", modules.EdgedModuleName, resource, model.InsertOperation, lease)

	var resp model.Message
	var err error
	if _, ok := core.GetModules()[modules.EdgeHubModuleName]; ok {
		resp, err = beehiveContext.SendSync(modules.EdgeHubModuleName, *nodeleaseMsg, e.nodeStatusUpdateFrequency)
	} else {
		resp, err = beehiveContext.SendSync(EdgeController, *nodeleaseMsg, e.nodeStatusUpdateFrequency)
	}

	if err != nil {
		return err
	}

	if resp.Content != constants.MessageSuccessfulContent {
		return fmt.Errorf("create node lease failed, resp: %v", resp.Content)
	}

	klog.Infof("successfully create node lease %s", lease.Name)
	e.nodeLeaseCreated = true
	baseLease = *lease.DeepCopy()
	return nil
}

func (e *edged) updateNodeLease() error {
	lease := e.newLease(&baseLease)
	err := e.metaClient.Lease(lease.Namespace).Update(lease)
	if err != nil {
		klog.Errorf("update node lease failed, error: %v", err)
		return err
	}
	baseLease = *lease.DeepCopy()
	return nil
}

func (e *edged) syncNodeLease() {
	if !e.nodeLeaseCreated {
		if err := e.createNodeLease(); err != nil {
			klog.Errorf("create nodelease failed: %v", err)
			return
		}
	}

	if err := e.updateNodeLease(); err != nil {
		klog.Errorf("unable to update nodelease: %v", err)
	}
}
