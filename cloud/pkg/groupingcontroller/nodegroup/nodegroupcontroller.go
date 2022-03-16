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

	groupingv1alpha1 "github.com/kubeedge/kubeedge/pkg/apis/grouping/v1alpha1"
)

const (
	// ControllerName is the controller name that will be used when reporting events.
	ControllerName = "nodegroup-controller"

	LabelBelongingTo             = "grouping.kubeedge.io/belonging-to"
	NodeGroupControllerFinalizer = "grouping.kubeedge.io/nodegroup-controller"
)

var (
	conditionStatusReadyStatusMap = map[corev1.ConditionStatus]groupingv1alpha1.ReadyStatus{
		corev1.ConditionTrue:    groupingv1alpha1.NodeReady,
		corev1.ConditionFalse:   groupingv1alpha1.NodeNotReady,
		corev1.ConditionUnknown: groupingv1alpha1.Unknown,
		// for the convinence of processing the situation that node has no ready condition
		"": groupingv1alpha1.Unknown,
	}
)

// nodeGroupStatusSort implements sort.Interface for NodeGroupStatus
type nodeGroupStatusSort struct {
	list []groupingv1alpha1.NodeStatus
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

	nodeGroup := &groupingv1alpha1.NodeGroup{}
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

func (c *Controller) syncNodeGroup(nodeGroup *groupingv1alpha1.NodeGroup) (controllerruntime.Result, error) {
	newNodes, err := c.getNodesSelectedBy(nodeGroup)
	if err != nil {
		klog.Errorf("failed to get all new nodes, %s, continue with what have found.", err)
	}

	oldNodes, err := c.getNodesByLabels(map[string]string{
		LabelBelongingTo: nodeGroup.Name,
	})
	if err != nil {
		klog.Errorf("failed to get old nodes for nodegroup %s, %s.", nodeGroup.Name, err)
		return controllerruntime.Result{Requeue: true}, err
	}

	// delete belonging label on nodes that do not belong to this node group
	nodesDeleted, _ := nodesDiff(oldNodes, newNodes)
	if err := c.removeBelongingLabelOnNodes(nodesDeleted); err != nil {
		klog.Errorf("failed to remove label on nodes that do not belong to this nodegroup, %s", err)
		return controllerruntime.Result{Requeue: true}, err
	}

	// collect statuses of nodes
	nodeStatusList := []groupingv1alpha1.NodeStatus{}
	for _, node := range newNodes {
		nodeStatus := groupingv1alpha1.NodeStatus{
			NodeName: node.Name,
		}
		// update ReadyStatus
		nodeReadyConditionStatus, _ := getNodeReadyConditionFromNode(&node)
		nodeStatus.ReadyStatus = conditionStatusReadyStatusMap[nodeReadyConditionStatus]

		// try to add node group label to this node
		if err := c.addOrUpdateNodeLabel(&node, nodeGroup.Name); err != nil {
			klog.Errorf("failed to update belonging label for node %s in nodegroup %s, %s, continue to reconcile other nodes", node, nodeGroup.Name, err)
			nodeStatus.SelectionStatus = groupingv1alpha1.FailedSelection
			nodeStatus.SelectionStatusReason = err.Error()
		}
	}

	sort.Sort(&nodeGroupStatusSort{nodeStatusList})
	newNodeGroupStatus := groupingv1alpha1.NodeGroupStatus{NodeStatuses: nodeStatusList}
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
		For(&groupingv1alpha1.NodeGroup{}).
		Watches(&source.Kind{Type: &corev1.Node{}}, handler.EnqueueRequestsFromMapFunc(c.nodeMapFunc), nil).
		Complete(c)
}

func (c *Controller) removeBelongingLabelOnNodes(nodes []corev1.Node) error {
	errs := []error{}
	for _, node := range nodes {
		n := node.DeepCopy()
		delete(n.Labels, LabelBelongingTo)
		if err := c.Client.Update(context.TODO(), n); err != nil {
			errs = append(errs, err)
		}
	}
	return utilerrors.NewAggregate(errs)
}

func (c *Controller) removeFinalizer(nodeGroup *groupingv1alpha1.NodeGroup) error {
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

func (c *Controller) getNodesSelectedBy(nodeGroup *groupingv1alpha1.NodeGroup) ([]corev1.Node, error) {
	errs := []error{}
	allNodes, err := c.getNodesByLabels(nodeGroup.Spec.MatchLabels)
	if err != nil {
		klog.Errorf("failed to get nodes by MatchLabels %v, %s", nodeGroup.Spec.MatchLabels, err)
		errs = append(errs, err)
	}

	nodes, err := c.getNodesByNodeName(nodeGroup.Spec.Nodes)
	if err != nil {
		klog.Errorf("failed to get all nodes specified in the NodeGroup.Spec.Nodes, %s.", err)
		errs = append(errs, err)
	}
	allNodes = append(allNodes, nodes...)
	return allNodes, utilerrors.NewAggregate(errs)
}

// We can assume that one node can only be in one of following conditions:
// 1. This node is an orphan, do not and will not belong to any NodeGroup.
// 2. This node is or will be a member of one NodeGroup.
func (c *Controller) nodeMapFunc(obj client.Object) []controllerruntime.Request {
	node := obj.(*corev1.Node)
	if nodeGroupName, ok := node.Labels[LabelBelongingTo]; ok {
		return []controllerruntime.Request{
			{
				NamespacedName: types.NamespacedName{
					Name: nodeGroupName,
				},
			},
		}
	}
	// node do not have belonging label, either a new node will be add to a node group or an orphan node
	nodegroupList := &groupingv1alpha1.NodeGroupList{}
	if err := c.Client.List(context.TODO(), nodegroupList); err != nil {
		klog.Errorf("failed to list all nodegroups, %s", err)
		return nil
	}

	for _, nodegroup := range nodegroupList.Items {
		if IfMatchNodeGroup(node, &nodegroup) {
			// this node will be added into a node group
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
		newNode := node.DeepCopy()
		delete(newNode.Labels, LabelBelongingTo)
		if err := c.Client.Update(context.TODO(), newNode); err != nil {
			klog.Errorf("failed to delete node group label of %s on node %s, %s", nodeGroupName, node, err)
		}
	}
	return utilerrors.NewAggregate(errs)
}

// getNodesByLabels can get all nodes matching these labels.
func (c *Controller) getNodesByLabels(matchLabels map[string]string) ([]corev1.Node, error) {
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
	newnode := node.DeepCopy()
	newnode.Labels[LabelBelongingTo] = nodeGroupName
	if err := c.Client.Update(context.TODO(), newnode); err != nil {
		klog.Errorf("failed to add label %s=%s to node %s, %s", LabelBelongingTo, nodeGroupName, node.Name, err)
		return err
	}
	return nil
}

// IfMatchNodeGroup will check if the node is selected by the nodegroup.
func IfMatchNodeGroup(node *corev1.Node, nodegroup *groupingv1alpha1.NodeGroup) bool {
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

func nodesDiff(oldNodes []corev1.Node, newNodes []corev1.Node) ([]corev1.Node, []corev1.Node) {
	nodesDeleted, nodesAdded := []corev1.Node{}, []corev1.Node{}
	m := map[string]corev1.Node{}
	for _, n := range oldNodes {
		m[n.Name] = n
	}
	for _, n := range newNodes {
		_, exist := m[n.Name]
		if exist {
			delete(m, n.Name)
		} else {
			nodesAdded = append(nodesAdded, n)
		}
	}
	for _, n := range m {
		nodesDeleted = append(nodesDeleted, n)
	}
	return nodesDeleted, nodesAdded
}
