package client

import (
	"context"
	"encoding/json"
	"fmt"

	corev1 "k8s.io/api/core/v1"
	rbacv1 "k8s.io/api/rbac/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	clientset "k8s.io/client-go/kubernetes"
	"k8s.io/client-go/tools/cache"
	"k8s.io/klog/v2"
	"k8s.io/kubernetes/pkg/serviceaccount"

	policyv1alpha1 "github.com/kubeedge/api/apis/policy/v1alpha1"
	"github.com/kubeedge/beehive/pkg/core/model"
	"github.com/kubeedge/kubeedge/edge/pkg/metamanager/dao"
)

const (
	roleKind        = "Role"
	clusterRoleKind = "ClusterRole"
)

type RoleGetter struct {
}

func (g *RoleGetter) GetRole(namespace, name string) (*rbacv1.Role, error) {
	rst, err := dao.QueryMeta("type", model.ResourceTypeSaAccess)
	if err != nil {
		return nil, err
	}
	for _, v := range *rst {
		var saAccess policyv1alpha1.ServiceAccountAccess
		err = json.Unmarshal([]byte(v), &saAccess)
		if err != nil {
			klog.Errorf("failed to unmarshal saAccess %v", err)
			return nil, err
		}
		for _, rb := range saAccess.Spec.AccessRoleBinding {
			if rb.RoleBinding.RoleRef.Kind == roleKind && rb.RoleBinding.RoleRef.Name == name &&
				saAccess.Namespace == namespace {
				return &rbacv1.Role{
					ObjectMeta: metav1.ObjectMeta{
						Name:      rb.RoleBinding.RoleRef.Name,
						Namespace: saAccess.Namespace,
					},
					Rules: rb.Rules,
				}, nil
			}
		}
	}
	return nil, fmt.Errorf("role %s/%s not found", namespace, name)
}

type RoleBindingLister struct {
}

func (l *RoleBindingLister) ListRoleBindings(namespace string) ([]*rbacv1.RoleBinding, error) {
	rst, err := dao.QueryMeta("type", model.ResourceTypeSaAccess)
	if err != nil {
		return nil, err
	}
	var items = make(map[string]struct{})
	var res []*rbacv1.RoleBinding
	for _, v := range *rst {
		var saAccess policyv1alpha1.ServiceAccountAccess
		err = json.Unmarshal([]byte(v), &saAccess)
		if err != nil {
			klog.Errorf("failed to unmarshal saAccess %v", err)
			return nil, err
		}
		for _, rb := range saAccess.Spec.AccessRoleBinding {
			var tmp = rb.RoleBinding
			if tmp.Namespace == namespace {
				key, err := cache.MetaNamespaceKeyFunc(&tmp)
				if err != nil {
					continue
				}
				if _, ok := items[key]; ok {
					continue
				}
				items[key] = struct{}{}
				res = append(res, &tmp)
			}
		}
	}
	return res, nil
}

type ClusterRoleGetter struct {
}

func (g *ClusterRoleGetter) GetClusterRole(name string) (*rbacv1.ClusterRole, error) {
	rst, err := dao.QueryMeta("type", model.ResourceTypeSaAccess)
	if err != nil {
		return nil, err
	}
	for _, v := range *rst {
		var saAccess policyv1alpha1.ServiceAccountAccess
		err = json.Unmarshal([]byte(v), &saAccess)
		if err != nil {
			klog.Errorf("failed to unmarshal saAccess %v", err)
			return nil, err
		}
		for _, rb := range saAccess.Spec.AccessRoleBinding {
			if rb.RoleBinding.RoleRef.Kind == clusterRoleKind && rb.RoleBinding.RoleRef.Name == name {
				return &rbacv1.ClusterRole{
					ObjectMeta: metav1.ObjectMeta{
						Name: rb.RoleBinding.RoleRef.Name,
					},
					Rules: rb.Rules,
				}, nil
			}
		}
		for _, crb := range saAccess.Spec.AccessClusterRoleBinding {
			if crb.ClusterRoleBinding.RoleRef.Kind == clusterRoleKind && crb.ClusterRoleBinding.RoleRef.Name == name {
				return &rbacv1.ClusterRole{
					ObjectMeta: metav1.ObjectMeta{
						Name: crb.ClusterRoleBinding.RoleRef.Name,
					},
					Rules: crb.Rules,
				}, nil
			}
		}
	}
	return nil, fmt.Errorf("clusterrole %s not found", name)
}

type ClusterRoleBindingLister struct {
}

func (l *ClusterRoleBindingLister) ListClusterRoleBindings() ([]*rbacv1.ClusterRoleBinding, error) {
	rst, err := dao.QueryMeta("type", model.ResourceTypeSaAccess)
	if err != nil {
		klog.Errorf("failed to query meta %v", err)
		return nil, err
	}
	var items = make(map[string]struct{})
	var res []*rbacv1.ClusterRoleBinding
	for _, v := range *rst {
		var saAccess policyv1alpha1.ServiceAccountAccess
		err = json.Unmarshal([]byte(v), &saAccess)
		if err != nil {
			klog.Errorf("failed to unmarshal saAccess %v", err)
			return nil, err
		}
		for _, crb := range saAccess.Spec.AccessClusterRoleBinding {
			var tmp = crb.ClusterRoleBinding
			key, err := cache.MetaNamespaceKeyFunc(&tmp)
			if err != nil {
				klog.Warningf("failed to get key for clusterrolebinding %v", err)
				continue
			}
			if _, ok := items[key]; ok {
				continue
			}
			items[key] = struct{}{}
			res = append(res, &tmp)
		}
	}
	return res, nil
}

// getter implements ServiceAccountTokenGetter using a clientset.Interface
type getter struct {
	Client clientset.Interface
}

// NewGetterFromClient returns a ServiceAccountTokenGetter that
// uses the specified Client to retrieve service accounts and secrets.
// The Client should NOT authenticate using a service account token
// the returned getter will be used to retrieve, or recursion will result.
func NewGetterFromClient(client clientset.Interface) serviceaccount.ServiceAccountTokenGetter {
	return getter{Client: client}
}

func (c getter) GetServiceAccount(namespace, name string) (*corev1.ServiceAccount, error) {
	return c.Client.CoreV1().ServiceAccounts(namespace).Get(context.Background(), name, metav1.GetOptions{})
}

func (c getter) GetPod(namespace, name string) (*corev1.Pod, error) {
	return c.Client.CoreV1().Pods(namespace).Get(context.Background(), name, metav1.GetOptions{})
}

func (c getter) GetSecret(namespace, name string) (*corev1.Secret, error) {
	return c.Client.CoreV1().Secrets(namespace).Get(context.Background(), name, metav1.GetOptions{})
}

func (c getter) GetNode(name string) (*corev1.Node, error) {
	return c.Client.CoreV1().Nodes().Get(context.Background(), name, metav1.GetOptions{})
}
