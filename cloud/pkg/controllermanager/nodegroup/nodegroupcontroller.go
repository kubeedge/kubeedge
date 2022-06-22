package nodegroup

import (
	"context"
	"fmt"
	"sort"

	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/equality"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/fields"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/apimachinery/pkg/types"
	utilerrors "k8s.io/apimachinery/pkg/util/errors"
	"k8s.io/apimachinery/pkg/util/sets"
	"k8s.io/klog/v2"
	controllerruntime "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
	"sigs.k8s.io/controller-runtime/pkg/handler"
	"sigs.k8s.io/controller-runtime/pkg/source"

	appsv1alpha1 "github.com/kubeedge/kubeedge/pkg/apis/apps/v1alpha1"
)

const (
	// ControllerName is the controller name that will be used when reporting events.
	ControllerName = "nodegroup-controller"

	LabelBelongingTo              = "apps.kubeedge.io/belonging-to"
	NodeGroupControllerFinalizer  = "apps.kubeedge.io/nodegroup-controller"
	ServiceTopologyAnnotation     = "apps.kubeedge.io/service-topology"
	ServiceTopologyRangeNodegroup = "range-nodegroup"
)

var (
	conditionStatusReadyStatusMap = map[corev1.ConditionStatus]appsv1alpha1.ReadyStatus{
		corev1.ConditionTrue:    appsv1alpha1.NodeReady,
		corev1.ConditionFalse:   appsv1alpha1.NodeNotReady,
		corev1.ConditionUnknown: appsv1alpha1.Unknown,
		// for the convinence of processing the situation that node has no ready condition
		"": appsv1alpha1.Unknown,
	}
)

// Controller is to sync NodeGroup.
type Controller struct {
	client.Client
}

// Reconcile performs a full reconciliation for the object referred to by the Request.
// The Controller will requeue the Request to be processed again if an error is non-nil or
// Result.Requeue is true, otherwise upon completion it will remove the work from the queue.
func (c *Controller) Reconcile(ctx context.Context, req controllerruntime.Request) (controllerruntime.Result, error) {
	klog.Infof("Reconciling nodeGroup %s", req.NamespacedName.Name)

	nodeGroup := &appsv1alpha1.NodeGroup{}
	if err := c.Client.Get(ctx, req.NamespacedName, nodeGroup); err != nil {
		// The resource may no longer exist, in which case we stop processing.
		if apierrors.IsNotFound(err) {
			return controllerruntime.Result{}, nil
		}

		return controllerruntime.Result{Requeue: true}, err
	}

	if !nodeGroup.DeletionTimestamp.IsZero() {
		// remove labels it added to nodes before deleting this NodeGroup
		klog.Infof("begin to remove node group label on nodes selected by nodegroup %s", nodeGroup.Name)
		if err := c.evictNodesInNodegroup(ctx, nodeGroup.Name); err != nil {
			return controllerruntime.Result{Requeue: true}, err
		}
		// this NodeGroup can be deleted now
		if err := c.removeFinalizer(ctx, nodeGroup); err != nil {
			return controllerruntime.Result{Requeue: true}, err
		}
		return controllerruntime.Result{}, nil
	}

	if !controllerutil.ContainsFinalizer(nodeGroup, NodeGroupControllerFinalizer) {
		controllerutil.AddFinalizer(nodeGroup, NodeGroupControllerFinalizer)
		if err := c.Client.Update(ctx, nodeGroup); err != nil {
			klog.Errorf("failed to add finalizer for nodegroup %s, %s", nodeGroup.Name, err)
			return controllerruntime.Result{Requeue: true}, err
		}
	}

	return c.syncNodeGroup(ctx, nodeGroup)
}

func (c *Controller) syncNodeGroup(ctx context.Context, nodeGroup *appsv1alpha1.NodeGroup) (controllerruntime.Result, error) {
	debugLogNodes := func(msg string, nodes []corev1.Node) {
		if klog.V(4).Enabled() {
			if len(nodes) == 0 {
				klog.Infof("%s: get no nodes when syncing nodegroup %s", msg, nodeGroup.Name)
				return
			}
			nodeNames := []string{}
			for i := range nodes {
				nodeNames = append(nodeNames, nodes[i].Name)
			}
			klog.Infof("%s: get %d nodes %v when syncing nodegroup %s", msg, len(nodes), nodeNames, nodeGroup.Name)
		}
	}

	newNodes, err := c.getNodesSelectedBy(ctx, nodeGroup)
	if err != nil {
		klog.Errorf("failed to get all new nodes, %s, continue with what have found.", err)
	}
	debugLogNodes("get new nodes", newNodes)

	oldNodes, err := c.getNodesByLabels(ctx, map[string]string{LabelBelongingTo: nodeGroup.Name})
	if err != nil {
		klog.Errorf("failed to get old nodes for nodegroup %s, %s.", nodeGroup.Name, err)
		return controllerruntime.Result{Requeue: true}, err
	}
	debugLogNodes("get current nodes", oldNodes)

	// delete belonging label on nodes that do not belong to this node group
	nodesDeleted, _ := nodesDiff(oldNodes, newNodes)
	debugLogNodes("get nodes to delete label", nodesDeleted)

	if err := c.evictNodes(ctx, nodesDeleted); err != nil {
		klog.Errorf("failed to evict nodes that do not belong to this nodegroup anymore, %s", err)
		return controllerruntime.Result{Requeue: true}, err
	}

	// This loop will
	// 1. add or update belonging label for nodes
	// 2. prepare NodeStatus for NodeGroup
	nodeStatusList := []appsv1alpha1.NodeStatus{}
	existingNodes := sets.NewString()
	for _, node := range newNodes {
		existingNodes = existingNodes.Insert(node.Name)
		nodeStatus := appsv1alpha1.NodeStatus{
			NodeName: node.Name,
		}
		// update ReadyStatus
		nodeReadyConditionStatus, _ := getNodeReadyConditionFromNode(&node)
		nodeStatus.ReadyStatus = conditionStatusReadyStatusMap[nodeReadyConditionStatus]
		klog.V(4).Infof("get status %s for node %s, when reconciling nodegroup %s", nodeStatus.ReadyStatus, node.Name, nodeGroup.Name)

		// try to add node group label to this node
		if err := c.addOrUpdateNodeLabel(ctx, &node, nodeGroup.Name); err != nil {
			klog.Errorf("failed to update belonging label for node %s in nodegroup %s, %s, continue to reconcile other nodes", node.Name, nodeGroup.Name, err)
			nodeStatus.SelectionStatus = appsv1alpha1.FailedSelection
			nodeStatus.SelectionStatusReason = err.Error()
		} else {
			nodeStatus.SelectionStatus = appsv1alpha1.SucceededSelection
		}
		nodeStatusList = append(nodeStatusList, nodeStatus)
	}
	// update status for nodes that do not exist but specified by node name.
	nonExistingNodes := sets.NewString(nodeGroup.Spec.Nodes...).Difference(existingNodes)
	for node := range nonExistingNodes {
		nodeStatusList = append(nodeStatusList, appsv1alpha1.NodeStatus{
			NodeName:              node,
			SelectionStatus:       appsv1alpha1.FailedSelection,
			SelectionStatusReason: "node does not exist",
			ReadyStatus:           appsv1alpha1.Unknown,
		})
	}
	sort.Slice(nodeStatusList, func(i, j int) bool {
		return nodeStatusList[i].NodeName < nodeStatusList[j].NodeName
	})
	if equality.Semantic.DeepEqual(nodeGroup.Status.NodeStatuses, nodeStatusList) {
		klog.V(4).Infof("status of nodegroup is unchanged, skip update")
		return controllerruntime.Result{}, nil
	}
	klog.V(4).Infof("status of nodegroup has changed, old: %v, new: %v", nodeGroup.Status.NodeStatuses, nodeStatusList)
	nodeGroup.Status.NodeStatuses = nodeStatusList
	if err := c.Status().Update(ctx, nodeGroup); err != nil {
		klog.Errorf("failed to update status for nodegroup %s, %s", nodeGroup.Name, err)
		return controllerruntime.Result{Requeue: true}, nil
	}
	return controllerruntime.Result{}, nil
}

// SetupWithManager creates a controller and register to controller manager.
func (c *Controller) SetupWithManager(ctx context.Context, mgr controllerruntime.Manager) error {
	if err := mgr.GetFieldIndexer().IndexField(ctx, &corev1.Pod{}, "spec.nodeName", func(o client.Object) []string {
		pod := o.(*corev1.Pod)
		return []string{pod.Spec.NodeName}
	}); err != nil {
		return fmt.Errorf("failed to set nodeName field selector for manager, %v", err)
	}
	return controllerruntime.NewControllerManagedBy(mgr).
		For(&appsv1alpha1.NodeGroup{}).
		Watches(&source.Kind{Type: &corev1.Node{}}, handler.EnqueueRequestsFromMapFunc(c.nodeMapFunc)).
		Complete(c)
}

// evictNodes will remove the belonging-to label from nodes and evict pods
// that should run in the nodegroup which the node was used to belong to.
func (c *Controller) evictNodes(ctx context.Context, nodes []corev1.Node) error {
	errs := []error{}
	for _, node := range nodes {
		n := node.DeepCopy()
		ng := n.Labels[LabelBelongingTo]
		delete(n.Labels, LabelBelongingTo)
		if err := c.Client.Patch(ctx, n, client.MergeFrom(&node)); err != nil {
			klog.Errorf("failed to remove belonging label of nodegroup %s on node %s, %v", ng, node.Name, err)
			errs = append(errs, err)
		}

		if err := c.evictPodsShouldNotRunOnNode(ctx, n, ng); err != nil {
			klog.Errorf("failed to evict pods running on node %s in nodegroup %s, %v", node.Name, ng, err)
			errs = append(errs, err)
		}
	}
	return utilerrors.NewAggregate(errs)
}

func (c *Controller) evictPodsShouldNotRunOnNode(ctx context.Context, node *corev1.Node, nodegroup string) error {
	// find all pods running on this node
	runningPods := &corev1.PodList{}
	nodeNameSelector := fields.OneTermEqualSelector("spec.nodeName", node.Name)
	if err := c.Client.List(ctx, runningPods, client.MatchingFieldsSelector{Selector: nodeNameSelector}); err != nil {
		return fmt.Errorf("failed to get pods running on node %s, %v", node.Name, err)
	}

	// evict pods
	errs := []error{}
	for i := range runningPods.Items {
		pod := &runningPods.Items[i]
		nodeSelector := pod.Spec.NodeSelector
		if v, ok := nodeSelector[LabelBelongingTo]; ok && v == nodegroup {
			// TODO: in an async way?
			// Delete pod seems to block until the pod has actually stopped
			if err := c.Client.Delete(ctx, pod); err != nil {
				errs = append(errs, fmt.Errorf("failed to delete pod %s/%s, %v", pod.Namespace, pod.Name, err))
			}
		}
	}
	return utilerrors.NewAggregate(errs)
}

func (c *Controller) removeFinalizer(ctx context.Context, nodeGroup *appsv1alpha1.NodeGroup) error {
	if !controllerutil.ContainsFinalizer(nodeGroup, NodeGroupControllerFinalizer) {
		return nil
	}
	controllerutil.RemoveFinalizer(nodeGroup, NodeGroupControllerFinalizer)
	if err := c.Client.Update(ctx, nodeGroup); err != nil {
		klog.Errorf("failed to remove finalizer on nodegroup %s, %s", nodeGroup.Name, err)
		return err
	}
	return nil
}

func (c *Controller) getNodesSelectedBy(ctx context.Context, nodeGroup *appsv1alpha1.NodeGroup) ([]corev1.Node, error) {
	errs := []error{}
	nodesByLabel, err := c.getNodesByLabels(ctx, nodeGroup.Spec.MatchLabels)
	if err != nil {
		klog.Errorf("failed to get nodes by MatchLabels %v, %s", nodeGroup.Spec.MatchLabels, err)
		errs = append(errs, err)
	}
	klog.V(4).Infof("get %d nodes that match labels in nodegroup %s", len(nodesByLabel), nodeGroup.Name)

	nodesByName, err := c.getNodesByNodeName(ctx, nodeGroup.Spec.Nodes)
	if err != nil {
		klog.Errorf("failed to get all nodes specified in the NodeGroup.Spec.Nodes, %s.", err)
		errs = append(errs, err)
	}
	klog.V(4).Infof("get %d nodes that specified by name in nodegroup %s", len(nodesByName), nodeGroup.Name)
	// remove duplicate nodes
	return nodesUnion(nodesByLabel, nodesByName), utilerrors.NewAggregate(errs)
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
	nodegroupList := &appsv1alpha1.NodeGroupList{}
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

func (c *Controller) evictNodesInNodegroup(ctx context.Context, nodeGroupName string) error {
	selector := labels.SelectorFromSet(labels.Set(
		map[string]string{LabelBelongingTo: nodeGroupName},
	))
	nodeList := &corev1.NodeList{}
	err := c.Client.List(ctx, nodeList, &client.ListOptions{LabelSelector: selector})
	if err != nil {
		return err
	}
	return c.evictNodes(ctx, nodeList.Items)
}

// getNodesByLabels can get all nodes matching these labels.
func (c *Controller) getNodesByLabels(ctx context.Context, matchLabels map[string]string) ([]corev1.Node, error) {
	if matchLabels == nil {
		// Return empty when matchLabels is nil
		// Otherwise, it will select all nodes, it's not what we want
		return []corev1.Node{}, nil
	}
	selector := labels.SelectorFromSet(labels.Set(matchLabels))
	nodeList := &corev1.NodeList{}
	err := c.Client.List(ctx, nodeList, &client.ListOptions{LabelSelector: selector})
	if err != nil {
		return nil, err
	}
	return nodeList.Items, nil
}

// getNodesByNodeName can get all nodes specified by node names.
func (c *Controller) getNodesByNodeName(ctx context.Context, nodeNames []string) ([]corev1.Node, error) {
	errs := []error{}
	nodes := []corev1.Node{}
	for _, name := range nodeNames {
		node := &corev1.Node{}
		if err := c.Client.Get(ctx, types.NamespacedName{Name: name}, node); err != nil {
			klog.Errorf("failed to get node %s, %s", name, err)
			errs = append(errs, err)
			continue
		}
		nodes = append(nodes, *node)
	}

	return nodes, utilerrors.NewAggregate(errs)
}

func (c *Controller) addOrUpdateNodeLabel(ctx context.Context, node *corev1.Node, nodeGroupName string) error {
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
	if newnode.Labels == nil {
		newnode.Labels = map[string]string{}
	}
	newnode.Labels[LabelBelongingTo] = nodeGroupName
	if err := c.Client.Patch(ctx, newnode, client.MergeFrom(node)); err != nil {
		klog.Errorf("failed to add label %s=%s to node %s, %s", LabelBelongingTo, nodeGroupName, node.Name, err)
		return err
	}
	return nil
}

// IfMatchNodeGroup will check if the node is selected by the nodegroup.
func IfMatchNodeGroup(node *corev1.Node, nodegroup *appsv1alpha1.NodeGroup) bool {
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

func nodesUnion(a []corev1.Node, b []corev1.Node) []corev1.Node {
	nodesMap := map[string]*corev1.Node{}
	for i := range a {
		nodesMap[a[i].Name] = &a[i]
	}
	for i := range b {
		nodesMap[b[i].Name] = &b[i]
	}

	nodes := []corev1.Node{}
	for _, node := range nodesMap {
		nodes = append(nodes, *node)
	}
	return nodes
}
