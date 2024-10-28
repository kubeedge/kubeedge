package controller

import (
	"context"
	"fmt"
	"reflect"
	"sort"

	corev1 "k8s.io/api/core/v1"
	rbacv1 "k8s.io/api/rbac/v1"
	"k8s.io/apimachinery/pkg/api/equality"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/fields"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/apiserver/pkg/authentication/serviceaccount"
	"k8s.io/apiserver/pkg/authentication/user"
	"k8s.io/klog/v2"
	controllerruntime "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/builder"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/handler"
	"sigs.k8s.io/controller-runtime/pkg/predicate"

	policyv1alpha1 "github.com/kubeedge/api/apis/policy/v1alpha1"
	"github.com/kubeedge/beehive/pkg/core/model"
	"github.com/kubeedge/kubeedge/cloud/pkg/common/messagelayer"
	"github.com/kubeedge/kubeedge/cloud/pkg/common/modules"
	"github.com/kubeedge/kubeedge/cloud/pkg/edgecontroller/constants"
	commonconstants "github.com/kubeedge/kubeedge/common/constants"
)

type Controller struct {
	client.Client
	MessageLayer messagelayer.MessageLayer
}

func (c *Controller) Reconcile(ctx context.Context, request controllerruntime.Request) (controllerruntime.Result, error) {
	acc := &policyv1alpha1.ServiceAccountAccess{}
	if err := c.Client.Get(ctx, request.NamespacedName, acc); err != nil {
		if apierrors.IsNotFound(err) {
			return controllerruntime.Result{}, nil
		}
		klog.Errorf("failed to get serviceaccountaccess %s/%s, %v", request.Namespace, request.Name, err)
		return controllerruntime.Result{Requeue: true}, err
	}
	if !acc.GetDeletionTimestamp().IsZero() {
		return controllerruntime.Result{}, nil
	}
	return c.syncRules(ctx, acc)
}

func (c *Controller) filterResource(ctx context.Context, object client.Object) bool {
	var p = &PolicyMatcher{}
	matchTarget(ctx, c.Client, object, p.isMatchServiceAccount)
	klog.V(4).Infof("filter resource %s/%s, %v", object.GetNamespace(), object.GetName(), p.match)
	return p.match
}

func isMatchedRoleRef(roleRef rbacv1.RoleRef, bindingNamespace string, object client.Object) bool {
	if reflect.TypeOf(object).Elem().Name() != roleRef.Kind {
		return false
	}
	if roleRef.Kind == "ClusterRole" {
		return object.GetName() == roleRef.Name
	} else if roleRef.Kind == "Role" {
		return object.GetName() == roleRef.Name && bindingNamespace == object.GetNamespace()
	}
	return false
}

type PolicyMatcher struct {
	match bool
}

func (pm *PolicyMatcher) isMatchServiceAccount(*policyv1alpha1.ServiceAccountAccess) bool {
	pm.match = true
	return false
}

type PolicyRequestVisitor struct {
	AuthPolicy []controllerruntime.Request
}

func (p *PolicyRequestVisitor) matchRuntimeRequest(acc *policyv1alpha1.ServiceAccountAccess) bool {
	p.AuthPolicy = append(p.AuthPolicy, controllerruntime.Request{NamespacedName: client.ObjectKey{Namespace: acc.Namespace, Name: acc.Name}})
	return true
}

func matchTarget(ctx context.Context, cli client.Client, object client.Object, visitor func(*policyv1alpha1.ServiceAccountAccess) bool) {
	accList := &policyv1alpha1.ServiceAccountAccessList{}
	if err := cli.List(ctx, accList); err != nil {
		klog.Errorf("failed to list serviceaccountaccess, %v", err)
		return
	}

	crbl := &rbacv1.ClusterRoleBindingList{}
	if err := cli.List(ctx, crbl); err != nil {
		klog.Errorf("failed to list clusterrolebindings, %v", err)
		return
	}

	for _, am := range accList.Items {
		userInfo := serviceaccount.UserInfo(am.Spec.ServiceAccount.Namespace, am.Spec.ServiceAccount.Name, string(am.Spec.ServiceAccount.UID))
		switch obj := object.(type) {
		case *rbacv1.ClusterRoleBinding:
			_, applies := appliesTo(userInfo, obj.Subjects, "")
			if applies && !visitor(&am) {
				return
			}
		case *rbacv1.RoleBinding:
			_, applies := appliesTo(userInfo, obj.Subjects, obj.Namespace)
			if applies && !visitor(&am) {
				return
			}
		case *rbacv1.ClusterRole:
			for _, crb := range crbl.Items {
				if !isMatchedRoleRef(crb.RoleRef, "", obj) {
					continue
				}
				_, applies := appliesTo(userInfo, crb.Subjects, "")
				if applies && !visitor(&am) {
					return
				}
			}
			var roleBindingList = &rbacv1.RoleBindingList{}
			if err := cli.List(ctx, roleBindingList, &client.ListOptions{Namespace: am.Spec.ServiceAccount.Namespace}); err != nil {
				klog.Errorf("failed to list rolebindings, %v", err)
				return
			}
			for _, rb := range roleBindingList.Items {
				if !isMatchedRoleRef(rb.RoleRef, rb.Namespace, obj) {
					continue
				}
				_, applies := appliesTo(userInfo, rb.Subjects, rb.Namespace)
				if applies && !visitor(&am) {
					return
				}
			}
		case *rbacv1.Role:
			var roleBindingList = &rbacv1.RoleBindingList{}
			if err := cli.List(ctx, roleBindingList, &client.ListOptions{Namespace: am.Spec.ServiceAccount.Namespace}); err != nil {
				klog.Errorf("failed to list rolebindings, %v", err)
				return
			}
			for _, rb := range roleBindingList.Items {
				if !isMatchedRoleRef(rb.RoleRef, rb.Namespace, obj) {
					continue
				}
				_, applies := appliesTo(userInfo, rb.Subjects, rb.Namespace)
				if applies && !visitor(&am) {
					return
				}
			}
		}
	}
}

func (c *Controller) mapRolesFunc(_ context.Context, object client.Object) []controllerruntime.Request {
	var p = PolicyRequestVisitor{}
	matchTarget(context.Background(), c.Client, object, p.matchRuntimeRequest)
	klog.V(4).Infof("filter resource %s/%s, %v", object.GetNamespace(), object.GetName(), p.AuthPolicy)
	return p.AuthPolicy
}

func newSaAccessObject(sa corev1.ServiceAccount) *policyv1alpha1.ServiceAccountAccess {
	return &policyv1alpha1.ServiceAccountAccess{
		ObjectMeta: metav1.ObjectMeta{
			Name:      sa.GetName(),
			Namespace: sa.GetNamespace(),
		},
		Spec: policyv1alpha1.AccessSpec{
			ServiceAccount: sa,
		},
	}
}

func (c *Controller) mapObjectFunc(_ context.Context, object client.Object) []controllerruntime.Request {
	accList := &policyv1alpha1.ServiceAccountAccessList{}
	if err := c.Client.List(context.Background(), accList, &client.ListOptions{Namespace: object.GetNamespace()}); err != nil {
		klog.Errorf("failed to list serviceaccountaccess, %v", err)
		return nil
	}
	switch obj := object.(type) {
	case *corev1.Pod:
		sa := obj.Spec.ServiceAccountName
		for _, am := range accList.Items {
			if am.Spec.ServiceAccount.Name == sa && am.Spec.ServiceAccount.Namespace == object.GetNamespace() {
				if obj.GetDeletionTimestamp() == nil {
					// won't reconcile pod update event
					return []controllerruntime.Request{}
				}
				klog.V(4).Infof("reconcile pod deleting %s/%s", obj.Namespace, obj.Name)
				return []controllerruntime.Request{{NamespacedName: client.ObjectKey{Namespace: am.Namespace, Name: am.Name}}}
			}
		}
		// already deleted serviceaccountaccess
		if obj.GetDeletionTimestamp() != nil {
			return []controllerruntime.Request{}
		}
		// create serviceaccountaccess if not exist when pod event triggered
		newSaa := newSaAccessObject(corev1.ServiceAccount{
			ObjectMeta: metav1.ObjectMeta{
				Name:      sa,
				Namespace: object.GetNamespace(),
			},
		})
		klog.V(4).Infof("create serviceaccountaccess %s/%s for pod %s/%s", newSaa.Namespace, newSaa.Name, obj.Namespace, obj.Name)
		if err := c.Client.Create(context.Background(), newSaa); err != nil {
			klog.Errorf("failed to create serviceaccountaccess, %v", err)
			return nil
		}
		// return empty request to avoid reconcile conflict with serviceaccountaccess resource
		return []controllerruntime.Request{}
	case *corev1.ServiceAccount:
		return []controllerruntime.Request{{NamespacedName: client.ObjectKey{Namespace: object.GetNamespace(), Name: object.GetName()}}}
	}

	return []controllerruntime.Request{}
}

func (c *Controller) filterObject(ctx context.Context, object client.Object) bool {
	switch obj := object.(type) {
	case *corev1.Pod:
		node := obj.Spec.NodeName
		if obj.Spec.ServiceAccountName == "" || node == "" || !isEdgeNode(ctx, c.Client, node) {
			return false
		}
		return true
	case *corev1.ServiceAccount:
		accList := &policyv1alpha1.ServiceAccountAccessList{}
		if err := c.Client.List(ctx, accList, &client.ListOptions{Namespace: object.GetNamespace()}); err != nil {
			klog.Errorf("failed to list serviceaccountaccess, %v", err)
			return false
		}
		for _, am := range accList.Items {
			if am.Spec.ServiceAccount.Name == object.GetName() && am.Spec.ServiceAccount.Namespace == object.GetNamespace() {
				return true
			}
		}
	}
	return false
}

// SetupWithManager creates a controller and register to controller manager.
func (c *Controller) SetupWithManager(ctx context.Context, mgr controllerruntime.Manager) error {
	if err := mgr.GetFieldIndexer().IndexField(ctx, &corev1.Pod{}, "spec.serviceAccountName", func(o client.Object) []string {
		pod := o.(*corev1.Pod)
		return []string{pod.Spec.ServiceAccountName}
	}); err != nil {
		return fmt.Errorf("failed to set ServiceAccountName field selector for manager, %v", err)
	}
	return controllerruntime.NewControllerManagedBy(mgr).
		For(&policyv1alpha1.ServiceAccountAccess{}).
		Watches(&rbacv1.ClusterRoleBinding{}, handler.EnqueueRequestsFromMapFunc(c.mapRolesFunc), builder.WithPredicates(predicate.NewPredicateFuncs(func(object client.Object) bool {
			return c.filterResource(ctx, object)
		}))).
		Watches(&rbacv1.RoleBinding{}, handler.EnqueueRequestsFromMapFunc(c.mapRolesFunc), builder.WithPredicates(predicate.NewPredicateFuncs(func(object client.Object) bool {
			return c.filterResource(ctx, object)
		}))).
		Watches(&rbacv1.ClusterRole{}, handler.EnqueueRequestsFromMapFunc(c.mapRolesFunc), builder.WithPredicates(predicate.NewPredicateFuncs(func(object client.Object) bool {
			return c.filterResource(ctx, object)
		}))).
		Watches(&rbacv1.Role{}, handler.EnqueueRequestsFromMapFunc(c.mapRolesFunc), builder.WithPredicates(predicate.NewPredicateFuncs(func(object client.Object) bool {
			return c.filterResource(ctx, object)
		}))).
		Watches(&corev1.ServiceAccount{}, handler.EnqueueRequestsFromMapFunc(c.mapObjectFunc), builder.WithPredicates(predicate.NewPredicateFuncs(func(object client.Object) bool {
			return c.filterObject(ctx, object)
		}))).
		Watches(&corev1.Pod{}, handler.EnqueueRequestsFromMapFunc(c.mapObjectFunc), builder.WithPredicates(predicate.NewPredicateFuncs(func(object client.Object) bool {
			return c.filterObject(ctx, object)
		}))).
		Complete(c)
}

func isEdgeNode(ctx context.Context, cli client.Client, name string) bool {
	set := labels.Set{commonconstants.EdgeNodeRoleKey: commonconstants.EdgeNodeRoleValue}
	selector := labels.SelectorFromSet(set)
	var edgeNodeList = &corev1.NodeList{}
	if err := cli.List(ctx, edgeNodeList, &client.ListOptions{LabelSelector: selector}); err != nil {
		klog.Errorf("failed to list edge nodes, %v", err)
		return false
	}
	for _, node := range edgeNodeList.Items {
		if node.Name == name {
			return true
		}
	}
	return false
}

func getNodeListOfServiceAccountAccess(ctx context.Context, cli client.Client, acc *policyv1alpha1.ServiceAccountAccess) ([]string, error) {
	var nodeList []string
	podList := &corev1.PodList{}
	saNameSelector := fields.OneTermEqualSelector("spec.serviceAccountName", acc.Spec.ServiceAccount.Name)
	if err := cli.List(ctx, podList, client.MatchingFieldsSelector{Selector: saNameSelector}, &client.ListOptions{Namespace: acc.Namespace}); err != nil {
		klog.Errorf("failed to list pods through field selector serviceAccountName, %v", err)
		return nil, err
	}

	var nodeMap = make(map[string]bool)
	for _, pod := range podList.Items {
		if pod.Spec.NodeName == "" {
			continue
		}
		if nodeMap[pod.Spec.NodeName] {
			continue
		}
		if !isEdgeNode(ctx, cli, pod.Spec.NodeName) {
			continue
		}
		nodeMap[pod.Spec.NodeName] = true
		nodeList = append(nodeList, pod.Spec.NodeName)
	}
	return nodeList, nil
}

func intersectSlice(old, new []string) []string {
	var intersect = []string{}
	var oldMap = make(map[string]bool)
	for _, oldItem := range old {
		oldMap[oldItem] = true
	}
	for _, newItem := range new {
		if oldMap[newItem] {
			intersect = append(intersect, newItem)
		}
	}
	return intersect
}

func subtractSlice(source, subTarget []string) []string {
	var subtract = []string{}
	var oldMap = make(map[string]bool)
	for _, oldItem := range source {
		oldMap[oldItem] = true
	}
	for _, newItem := range subTarget {
		if !oldMap[newItem] {
			subtract = append(subtract, newItem)
		}
	}
	return subtract
}

func (c *Controller) send2Edge(acc *policyv1alpha1.ServiceAccountAccess, targets []string, opr string) {
	sendObj := acc.DeepCopy()
	for _, node := range targets {
		resource, err := messagelayer.BuildResource(node, sendObj.Namespace, model.ResourceTypeSaAccess, sendObj.Name)
		if err != nil {
			klog.Warningf("built message resource failed with error: %s", err)
			continue
		}
		// filter out the node list data
		sendObj.Status.NodeList = []string{}
		msg := model.NewMessage("").
			SetResourceVersion(sendObj.ResourceVersion).
			FillBody(sendObj).BuildRouter(modules.PolicyControllerModuleName, constants.GroupResource, resource, opr)
		if err := c.MessageLayer.Send(*msg); err != nil {
			klog.Warningf("send message %s failed with error: %s", resource, err)
			continue
		}
	}
}

func (c *Controller) syncRules(ctx context.Context, acc *policyv1alpha1.ServiceAccountAccess) (controllerruntime.Result, error) {
	var newSA = &corev1.ServiceAccount{}
	err := c.Client.Get(ctx, types.NamespacedName{Namespace: acc.Namespace, Name: acc.Spec.ServiceAccount.Name}, newSA)
	if (err != nil && apierrors.IsNotFound(err)) || (err == nil && newSA.DeletionTimestamp != nil) {
		klog.V(4).Infof("serviceaccount %s/%s is removed and delete the policy resource", acc.Namespace, acc.Spec.ServiceAccount.Name)
		copyObj := acc.DeepCopy()
		if err := c.Client.Delete(ctx, copyObj); err != nil {
			klog.Errorf("failed to delete serviceaccountaccess %s/%s, %v", copyObj.Namespace, copyObj.Name, err)
			return controllerruntime.Result{Requeue: true}, err
		}
		c.send2Edge(copyObj, copyObj.Status.NodeList, model.DeleteOperation)
		return controllerruntime.Result{}, nil
	} else if err != nil {
		klog.Errorf("failed to get serviceaccount %s/%s, %v", acc.Namespace, acc.Spec.ServiceAccount.Name, err)
		return controllerruntime.Result{Requeue: true}, err
	}
	userInfo := serviceaccount.UserInfo(newSA.Namespace, newSA.Name, string(newSA.UID))
	var currentAcc = &policyv1alpha1.ServiceAccountAccess{}
	c.VisitRulesFor(ctx, userInfo, acc.Namespace, currentAcc)
	nodes, err := getNodeListOfServiceAccountAccess(ctx, c.Client, acc)
	if err != nil {
		klog.Errorf("failed to get node list of serviceaccountaccess %s/%s, %v", acc.Namespace, acc.Name, err)
		return controllerruntime.Result{Requeue: true}, err
	}
	currentAcc.Spec.ServiceAccount = *newSA
	currentAcc.Spec.ServiceAccountUID = newSA.UID
	if len(nodes) == 0 && len(acc.Status.NodeList) == 0 {
		klog.Warningf("no nodes found for serviceaccountaccess %s/%s", acc.Namespace, acc.Name)
		return controllerruntime.Result{}, nil
	}
	deleteNodes := subtractSlice(nodes, acc.Status.NodeList)
	if len(deleteNodes) != 0 {
		// no nodes in the current acc status, delete the acc
		if len(nodes) == 0 {
			if err = c.Client.Delete(ctx, acc); err != nil {
				klog.Errorf("failed to delete serviceaccountaccess %s/%s, %v", acc.Namespace, acc.Name, err)
				return controllerruntime.Result{Requeue: true}, err
			}
			klog.V(4).Infof("delete serviceaccountaccess %s/%s", acc.Namespace, acc.Name)
			c.send2Edge(acc, deleteNodes, model.DeleteOperation)
			return controllerruntime.Result{}, nil
		}
		c.send2Edge(acc, deleteNodes, model.DeleteOperation)
	}
	sort.Slice(currentAcc.Spec.AccessRoleBinding, func(i, j int) bool {
		return currentAcc.Spec.AccessRoleBinding[i].RoleBinding.Name < currentAcc.Spec.AccessRoleBinding[j].RoleBinding.Name
	})
	sort.Slice(currentAcc.Spec.AccessClusterRoleBinding, func(i, j int) bool {
		return currentAcc.Spec.AccessClusterRoleBinding[i].ClusterRoleBinding.Name < currentAcc.Spec.AccessClusterRoleBinding[j].ClusterRoleBinding.Name
	})
	sort.Slice(nodes, func(i, j int) bool {
		return nodes[i] < nodes[j]
	})
	if !equalAccessBindingSlice(acc.Spec.AccessClusterRoleBinding, currentAcc.Spec.AccessClusterRoleBinding) ||
		!equalAccessBindingSlice(acc.Spec.AccessRoleBinding, currentAcc.Spec.AccessRoleBinding) ||
		!equalServiceAccount(&acc.Spec.ServiceAccount, &currentAcc.Spec.ServiceAccount) ||
		acc.Spec.ServiceAccountUID != currentAcc.Spec.ServiceAccountUID {
		acc.Spec = *currentAcc.Spec.DeepCopy()
		if err := c.Client.Update(ctx, acc); err != nil {
			klog.Errorf("failed to update serviceaccountaccess %s/%s, %v", acc.Namespace, acc.Name, err)
			return controllerruntime.Result{Requeue: true}, err
		}
		if !equality.Semantic.DeepEqual(acc.Status.NodeList, nodes) {
			acc.Status.NodeList = append([]string{}, nodes...)
			if err := c.Client.Status().Update(ctx, acc); err != nil {
				klog.Errorf("failed to update serviceaccountaccess status %s/%s, %v", acc.Namespace, acc.Name, err)
				return controllerruntime.Result{Requeue: true}, err
			}
		}
		c.send2Edge(acc, nodes, model.UpdateOperation)
	} else {
		addNodes := subtractSlice(acc.Status.NodeList, nodes)
		klog.V(4).Infof("serviceaccountaccess spec %s/%s is up to date", acc.Namespace, acc.Name)
		if len(addNodes) != 0 {
			acc.Status.NodeList = append([]string{}, nodes...)
			if err := c.Client.Status().Update(ctx, acc); err != nil {
				klog.Errorf("failed to update serviceaccountaccess status %s/%s, %v", acc.Namespace, acc.Name, err)
				return controllerruntime.Result{Requeue: true}, err
			}
			c.send2Edge(acc, addNodes, model.InsertOperation)
		}
	}
	return controllerruntime.Result{}, nil
}

func equalAccessBindingSlice(a, b interface{}) bool {
	aBindings, aOk := a.([]policyv1alpha1.AccessRoleBinding)
	bBindings, bOk := b.([]policyv1alpha1.AccessRoleBinding)
	if !aOk || !bOk {
		aClusterBindings, aClusterOk := a.([]policyv1alpha1.AccessClusterRoleBinding)
		bClusterBindings, bClusterOk := b.([]policyv1alpha1.AccessClusterRoleBinding)
		if !aClusterOk || !bClusterOk {
			return false
		}
		if len(aClusterBindings) != len(bClusterBindings) {
			return false
		}
		for i := range aClusterBindings {
			if aClusterBindings[i].ClusterRoleBinding.Name != bClusterBindings[i].ClusterRoleBinding.Name ||
				!equality.Semantic.DeepEqual(aClusterBindings[i].Rules, bClusterBindings[i].Rules) ||
				!equality.Semantic.DeepEqual(aClusterBindings[i].ClusterRoleBinding.Labels, bClusterBindings[i].ClusterRoleBinding.Labels) ||
				!equality.Semantic.DeepEqual(aClusterBindings[i].ClusterRoleBinding.Annotations, bClusterBindings[i].ClusterRoleBinding.Annotations) ||
				!equality.Semantic.DeepEqual(aClusterBindings[i].ClusterRoleBinding.Subjects, bClusterBindings[i].ClusterRoleBinding.Subjects) ||
				!equality.Semantic.DeepEqual(aClusterBindings[i].ClusterRoleBinding.RoleRef, bClusterBindings[i].ClusterRoleBinding.RoleRef) {
				return false
			}
		}
	} else {
		if len(aBindings) != len(bBindings) {
			return false
		}
		for i := range aBindings {
			if aBindings[i].RoleBinding.Name != bBindings[i].RoleBinding.Name ||
				aBindings[i].RoleBinding.Namespace != bBindings[i].RoleBinding.Namespace ||
				!equality.Semantic.DeepEqual(aBindings[i].Rules, bBindings[i].Rules) ||
				!equality.Semantic.DeepEqual(aBindings[i].RoleBinding.Labels, bBindings[i].RoleBinding.Labels) ||
				!equality.Semantic.DeepEqual(aBindings[i].RoleBinding.Annotations, bBindings[i].RoleBinding.Annotations) ||
				!equality.Semantic.DeepEqual(aBindings[i].RoleBinding.RoleRef, bBindings[i].RoleBinding.RoleRef) ||
				!equality.Semantic.DeepEqual(aBindings[i].RoleBinding.Subjects, bBindings[i].RoleBinding.Subjects) {
				return false
			}
		}
	}
	return true
}

func equalServiceAccount(a, b *corev1.ServiceAccount) bool {
	if a == nil || b == nil {
		return false
	}
	aCopy := a.DeepCopy()
	bCopy := b.DeepCopy()
	// ignore metadata because it is not be allowed to update in crd
	aCopy.ObjectMeta = bCopy.ObjectMeta
	return equality.Semantic.DeepEqual(aCopy, bCopy)
}

func appliesToUser(user user.Info, subject rbacv1.Subject, namespace string) bool {
	switch subject.Kind {
	case rbacv1.UserKind:
		return user.GetName() == subject.Name

	case rbacv1.GroupKind:
		return has(user.GetGroups(), subject.Name)

	case rbacv1.ServiceAccountKind:
		// default the namespace to namespace we're working in if its available.  This allows rolebindings that reference
		// SAs in th local namespace to avoid having to qualify them.
		saNamespace := namespace
		if len(subject.Namespace) > 0 {
			saNamespace = subject.Namespace
		}
		if len(saNamespace) == 0 {
			return false
		}
		// use a more efficient comparison for RBAC checking
		return serviceaccount.MatchesUsername(saNamespace, subject.Name, user.GetName())
	default:
		return false
	}
}

// appliesTo returns whether any of the bindingSubjects applies to the specified subject,
// and if true, the index of the first subject that applies
func appliesTo(user user.Info, bindingSubjects []rbacv1.Subject, namespace string) (int, bool) {
	for i, bindingSubject := range bindingSubjects {
		if appliesToUser(user, bindingSubject, namespace) {
			return i, true
		}
	}
	return 0, false
}

func has(set []string, ele string) bool {
	for _, s := range set {
		if s == ele {
			return true
		}
	}
	return false
}

// GetRoleReferenceRules attempts to resolve the RoleBinding or ClusterRoleBinding.
func (c *Controller) GetRoleReferenceRules(ctx context.Context, roleRef rbacv1.RoleRef, bindingNamespace string) ([]rbacv1.PolicyRule, error) {
	switch roleRef.Kind {
	case "Role":
		var role = &rbacv1.Role{}
		err := c.Client.Get(ctx, types.NamespacedName{Namespace: bindingNamespace, Name: roleRef.Name}, role)
		if err != nil {
			return nil, err
		}
		return role.Rules, nil

	case "ClusterRole":
		var clusterRole = &rbacv1.ClusterRole{}
		err := c.Client.Get(ctx, types.NamespacedName{Name: roleRef.Name}, clusterRole)
		if err != nil {
			return nil, err
		}
		return clusterRole.Rules, nil

	default:
		return nil, fmt.Errorf("unsupported role reference kind: %q", roleRef.Kind)
	}
}

func (c *Controller) VisitRulesFor(ctx context.Context, user user.Info, namespace string, acc *policyv1alpha1.ServiceAccountAccess) {
	crbl := &rbacv1.ClusterRoleBindingList{}
	if err := c.Client.List(ctx, crbl); err != nil {
		klog.Errorf("failed to list clusterrolebindings, %v", err)
		return
	}
	for _, crb := range crbl.Items {
		_, applies := appliesTo(user, crb.Subjects, "")
		if !applies {
			continue
		}
		rules, err := c.GetRoleReferenceRules(ctx, crb.RoleRef, "")
		if err != nil {
			klog.Errorf("failed to get rules for clusterrolebinding %s, %v", crb.Name, err)
			return
		}
		var accessClusterRoleBinding = policyv1alpha1.AccessClusterRoleBinding{
			ClusterRoleBinding: crb,
			Rules:              rules,
		}
		acc.Spec.AccessClusterRoleBinding = append(acc.Spec.AccessClusterRoleBinding, accessClusterRoleBinding)
	}

	if len(namespace) > 0 {
		var roleBindingList = &rbacv1.RoleBindingList{}
		if err := c.Client.List(ctx, roleBindingList, &client.ListOptions{Namespace: namespace}); err != nil {
			klog.Errorf("failed to list rolebindings, %v", err)
			return
		}
		for _, roleBinding := range roleBindingList.Items {
			_, applies := appliesTo(user, roleBinding.Subjects, namespace)
			if !applies {
				continue
			}
			rules, err := c.GetRoleReferenceRules(ctx, roleBinding.RoleRef, namespace)
			if err != nil {
				klog.Errorf("failed to get rules for rolebinding %s, %v", roleBinding.Name, err)
				return
			}
			var accessRoleBinding = policyv1alpha1.AccessRoleBinding{
				RoleBinding: roleBinding,
				Rules:       rules,
			}
			acc.Spec.AccessRoleBinding = append(acc.Spec.AccessRoleBinding, accessRoleBinding)
		}
	}
}
