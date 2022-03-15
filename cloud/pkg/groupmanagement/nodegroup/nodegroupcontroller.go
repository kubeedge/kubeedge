package nodegroup

import (
	"context"
	"fmt"
	"sort"

	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/equality"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/apimachinery/pkg/types"
	utilerrors "k8s.io/apimachinery/pkg/util/errors"
	"k8s.io/klog/v2"
	controllerruntime "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
	"sigs.k8s.io/controller-runtime/pkg/handler"
	"sigs.k8s.io/controller-runtime/pkg/source"

	groupmanagementv1alpha1 "github.com/kubeedge/kubeedge/cloud/pkg/apis/groupmanagement/v1alpha1"
)

const (
	// ControllerName is the controller name that will be used when reporting events.
	ControllerName = "nodegroup-controller"

	LabelBelongingTo             = "groupmanagement.kubeedge.io/belonging-to"
	NodeGroupControllerFinalizer = "groupmanagement.kubeedge.io/nodegroup-controller"
)

var (
	conditionStatusReadyStatusMap = map[corev1.ConditionStatus]groupmanagementv1alpha1.ReadyStatus{
		corev1.ConditionTrue:    groupmanagementv1alpha1.NodeReady,
		corev1.ConditionFalse:   groupmanagementv1alpha1.NodeNotReady,
		corev1.ConditionUnknown: groupmanagementv1alpha1.Unknown,
		// for the convinence of processing the situation that node has no ready condition
		"": groupmanagementv1alpha1.Unknown,
	}
)

// nodeGroupStatusSort implements sort.Interface for NodeGroupStatus
type nodeGroupStatusSort struct {
	list []groupmanagementv1alpha1.NodeStatus
}

func (n *nodeGroupStatusSort) Len() int           { return len(n.list) }
func (n *nodeGroupStatusSort) Less(i, j int) bool { return n.list[i].NodeName < n.list[j].NodeName }
func (n *nodeGroupStatusSort) Swap(i, j int) {
	tmp := n.list[i].DeepCopy()
	n.list[i] = *n.list[j].DeepCopy()
	n.list[j] = *tmp
}

// Controller is to sync NodeGroup.
type Controller struct {
	client.Client
}

// Reconcile performs a full reconciliation for the object referred to by the Request.
// The Controller will requeue the Request to be processed again if an error is non-nil or
// Result.Requeue is true, otherwise upon completion it will remove the work from the queue.
func (c *Controller) Reconcile(ctx context.Context, req controllerruntime.Request) (controllerruntime.Result, error) {
	klog.Infof("Reconciling nodeGroup %s", req.NamespacedName.Name)

	nodeGroup := &groupmanagementv1alpha1.NodeGroup{}
	if err := c.Client.Get(context.TODO(), req.NamespacedName, nodeGroup); err != nil {
		// The resource may no longer exist, in which case we stop processing.
		if apierrors.IsNotFound(err) {
			return controllerruntime.Result{}, nil
		}

		return controllerruntime.Result{Requeue: true}, err
	}

	if !nodeGroup.DeletionTimestamp.IsZero() {
		// remove labels it added to nodes before deleting this NodeGroup
		klog.Infof("begin to remove node group label on nodes selected by nodegroup %s", nodeGroup.Name)
		if err := c.removeBelongingLabelOfNodeGroup(nodeGroup.Name); err != nil {
			return controllerruntime.Result{Requeue: true}, err
		}
		// this NodeGroup can be deleted now
		if err := c.removeFinalizer(nodeGroup); err != nil {
			return controllerruntime.Result{Requeue: true}, err
		}
		return controllerruntime.Result{}, nil
	}

	if !controllerutil.ContainsFinalizer(nodeGroup, NodeGroupControllerFinalizer) {
		controllerutil.AddFinalizer(nodeGroup, NodeGroupControllerFinalizer)
		if err := c.Client.Update(context.TODO(), nodeGroup); err != nil {
			klog.Errorf("failed to add finalizer for nodegroup %s, %s", nodeGroup.Name, err)
			return controllerruntime.Result{Requeue: true}, err
		}
	}

	return c.syncNodeGroup(nodeGroup)
}

func (c *Controller) syncNodeGroup(nodeGroup *groupmanagementv1alpha1.NodeGroup) (controllerruntime.Result, error) {
	allNodes, err := c.getNodesByMatchLabels(nodeGroup.Spec.MatchLabels)
	if err != nil {
		klog.Errorf("failed to get nodes by MatchLabels, %s. Continue reconciliation with the specification of node names", err)
	}

	nodes, err := c.getNodesByNodeName(nodeGroup.Spec.Nodes)
	if err != nil {
		klog.Errorf("failed to get all nodes specified in the NodeGroup.Spec.Nodes, %s. Continue reconciliation with what has found.", err)
	}
	allNodes = append(allNodes, nodes...)

	// collect statuses of nodes
	nodeStatusList := []groupmanagementv1alpha1.NodeStatus{}
	for _, node := range allNodes {
		nodeStatus := groupmanagementv1alpha1.NodeStatus{
			NodeName: node.Name,
		}
		// update ReadyStatus
		nodeReadyConditionStatus, _ := getNodeReadyConditionFromNode(&node)
		nodeStatus.ReadyStatus = conditionStatusReadyStatusMap[nodeReadyConditionStatus]

		// try to add node group label to this node
		if err := c.addOrUpdateNodeLabel(&node, nodeGroup.Name); err != nil {
			klog.Errorf("failed to update belonging label for node %s, %s, continue to reconcile other nodes", node, err)
			nodeStatus.SelectionStatus = groupmanagementv1alpha1.FailedSelection
			nodeStatus.SelectionStatusReason = err.Error()
		}
	}

	sort.Sort(&nodeGroupStatusSort{nodeStatusList})
	newNodeGroupStatus := groupmanagementv1alpha1.NodeGroupStatus{NodeStatuses: nodeStatusList}
	if !equality.Semantic.DeepEqual(nodeGroup.Status, newNodeGroupStatus) {
		// the status of this NodeGroup has changed, try to update status
		nodeGroup.Status = newNodeGroupStatus
		if err := c.Status().Update(context.TODO(), nodeGroup); err != nil {
			klog.Errorf("failed to update status for nodegroup %s, %s", nodeGroup.Name, err)
			return controllerruntime.Result{Requeue: true}, nil
		}
	}

	return controllerruntime.Result{}, nil
}

// SetupWithManager creates a controller and register to controller manager.
func (c *Controller) SetupWithManager(mgr controllerruntime.Manager) error {
	return controllerruntime.NewControllerManagedBy(mgr).
		For(&groupmanagementv1alpha1.NodeGroup{}).
		Watches(&source.Kind{Type: &corev1.Node{}}, handler.EnqueueRequestsFromMapFunc(c.nodeMapFunc), nil).
		Complete(c)
}

func (c *Controller) removeFinalizer(nodeGroup *groupmanagementv1alpha1.NodeGroup) error {
	if !controllerutil.ContainsFinalizer(nodeGroup, LabelBelongingTo) {
		return nil
	}
	controllerutil.RemoveFinalizer(nodeGroup, NodeGroupControllerFinalizer)
	if err := c.Client.Update(context.TODO(), nodeGroup); err != nil {
		klog.Errorf("failed to remove finalizer on nodegroup %s, %s", nodeGroup, err)
		return err
	}
	return nil
}

// We can assume that one node can only be in one of following conditions:
// 1. This node is an orphan, do not and will not belong to any NodeGroup.
// 2. This node is or will be a member of one NodeGroup.
func (c *Controller) nodeMapFunc(obj client.Object) []controllerruntime.Request {
	node := obj.(*corev1.Node)
	nodegroupList := &groupmanagementv1alpha1.NodeGroupList{}
	if err := c.Client.List(context.TODO(), nodegroupList); err != nil {
		klog.Errorf("failed to list all nodegroups, %s", err)
		return nil
	}

	for _, nodegroup := range nodegroupList.Items {
		if IfMatchNodeGroup(node, &nodegroup) && c.ifReconcile(node, &nodegroup) {
			return []controllerruntime.Request{
				{
					NamespacedName: types.NamespacedName{
						Name: nodegroup.Name,
					},
				},
			}
		}
	}

	// an orphan node, do not reconcile
	return []controllerruntime.Request{}
}

func (c *Controller) removeBelongingLabelOfNodeGroup(nodeGroupName string) error {
	selector := labels.SelectorFromSet(labels.Set(
		map[string]string{LabelBelongingTo: nodeGroupName},
	))
	nodeList := &corev1.NodeList{}
	err := c.Client.List(context.TODO(), nodeList, &client.ListOptions{LabelSelector: selector})
	if err != nil {
		return err
	}

	errs := []error{}
	for _, node := range nodeList.Items {
		delete(node.Labels, LabelBelongingTo)
		if err := c.Client.Update(context.TODO(), &node); err != nil {
			klog.Errorf("failed to delete node group label of %s on node %s, %s", nodeGroupName, node, err)
		}
	}
	return utilerrors.NewAggregate(errs)
}

// getNodesByMatchLabels can get all nodes matching these labels.
func (c *Controller) getNodesByMatchLabels(matchLabels map[string]string) ([]corev1.Node, error) {
	selector := labels.SelectorFromSet(labels.Set(matchLabels))
	nodeList := &corev1.NodeList{}
	err := c.Client.List(context.TODO(), nodeList, &client.ListOptions{LabelSelector: selector})
	if err != nil {
		return nil, err
	}
	return nodeList.Items, nil
}

// getNodesByNodeName can get all nodes specified by node names.
func (c *Controller) getNodesByNodeName(nodeNames []string) ([]corev1.Node, error) {
	errs := []error{}
	nodes := []corev1.Node{}
	for _, name := range nodeNames {
		node := &corev1.Node{}
		if err := c.Client.Get(context.TODO(), types.NamespacedName{Name: name}, node); err != nil {
			klog.Errorf("failed to get node %s, %s", name, err)
			errs = append(errs, err)
			continue
		}
		nodes = append(nodes, *node)
	}

	return nodes, utilerrors.NewAggregate(errs)
}

func (c *Controller) addOrUpdateNodeLabel(node *corev1.Node, nodeGroupName string) error {
	nodeLabels := node.Labels
	v, ok := nodeLabels[LabelBelongingTo]
	if ok && v == nodeGroupName {
		// nothing to do
		return nil
	}
	if ok && v != nodeGroupName {
		return fmt.Errorf("node %s has already belonged to NodeGroup %s", node.Name, nodeGroupName)
	}

	// !ok
	// add new label to this node
	node.Labels[LabelBelongingTo] = nodeGroupName
	if err := c.Client.Update(context.TODO(), node); err != nil {
		klog.Errorf("failed to add label %s=%s to node %s, %s", LabelBelongingTo, nodeGroupName, node.Name, err)
		return err
	}
	return nil
}

func (c *Controller) ifReconcile(node *corev1.Node, nodeGroup *groupmanagementv1alpha1.NodeGroup) bool {
	currentNode := &corev1.Node{}
	if err := c.Client.Get(context.TODO(), types.NamespacedName{Name: currentNode.Name}, currentNode); err != nil {
		if apierrors.IsNotFound(err) {
			// this node has been deleted, need to update its nodegroup status.
			return true
		}
		klog.Errorf("failed to check if node %s has been deleted, %s", node.Name, err)
	}

	// NodeGroup of this node has changed, or
	// this node will be added to a NodeGroup
	if oldNodeGroup, exist := node.Labels[LabelBelongingTo]; oldNodeGroup != nodeGroup.Name || !exist {
		return true
	}
	// This controller also cares about the update of node status.
	findNode := false
	nodeReadyConditionStatus, hasReadyCondition := getNodeReadyConditionFromNode(node)
	for _, nodeStatus := range nodeGroup.Status.NodeStatuses {
		if nodeStatus.NodeName == node.Name {
			findNode = true
			oldReadyStatus := nodeStatus.ReadyStatus
			newReadyStatus := conditionStatusReadyStatusMap[nodeReadyConditionStatus]
			if oldReadyStatus != newReadyStatus {
				return true
			}
			break
		}
	}

	// findNode == false && hasReadyCondition == true: node starts to post ready condition
	// findNode == true && hasReadyCondition == false: normally impossible, reconcile it in case
	return findNode != hasReadyCondition
}

// IfMatchNodeGroup will check if the node is selected by the nodegroup.
func IfMatchNodeGroup(node *corev1.Node, nodegroup *groupmanagementv1alpha1.NodeGroup) bool {
	// check if nodename is in the nodegroup.Spec.Nodes
	for _, nodeName := range nodegroup.Spec.Nodes {
		if nodeName == node.Name {
			return true
		}
	}
	// check if labels of this node selected by nodegroup.Spec.MatchLabels
	selector := labels.SelectorFromSet(labels.Set(nodegroup.Spec.MatchLabels))
	return selector.Matches(labels.Set(node.Labels))
}

func getNodeReadyConditionFromNode(node *corev1.Node) (corev1.ConditionStatus, bool) {
	for _, condition := range node.Status.Conditions {
		if condition.Type == corev1.NodeReady {
			return condition.Status, true
		}
	}
	return "", false
}
