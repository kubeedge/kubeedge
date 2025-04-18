/*
Copyright 2025 The KubeEdge Authors.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

	http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package client

import (
	"context"
	"encoding/json"
	"errors"
	"testing"

	"github.com/agiledragon/gomonkey/v2"
	"github.com/stretchr/testify/assert"
	corev1 "k8s.io/api/core/v1"
	rbacv1 "k8s.io/api/rbac/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes/fake"

	"github.com/kubeedge/kubeedge/edge/pkg/metamanager/dao"
)

const (
	typeField               = "type"
	serviceAccountAccessVal = "serviceaccountaccess"
)

type mockServiceAccountAccess struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`
	Spec              struct {
		AccessRoleBinding []struct {
			RoleBinding rbacv1.RoleBinding  `json:"roleBinding"`
			Rules       []rbacv1.PolicyRule `json:"rules,omitempty"`
		} `json:"accessRoleBinding,omitempty"`
		AccessClusterRoleBinding []struct {
			ClusterRoleBinding rbacv1.ClusterRoleBinding `json:"clusterRoleBinding"`
			Rules              []rbacv1.PolicyRule       `json:"rules,omitempty"`
		} `json:"accessClusterRoleBinding,omitempty"`
	} `json:"spec,omitempty"`
}

func createMockResponse() []string {
	saAccess := mockServiceAccountAccess{}
	saAccess.TypeMeta = metav1.TypeMeta{
		Kind:       "ServiceAccountAccess",
		APIVersion: "policy.kubeedge.io/v1alpha1",
	}
	saAccess.ObjectMeta = metav1.ObjectMeta{
		Name:      "test-sa-access",
		Namespace: "test-namespace",
	}

	roleBinding := struct {
		RoleBinding rbacv1.RoleBinding  `json:"roleBinding"`
		Rules       []rbacv1.PolicyRule `json:"rules,omitempty"`
	}{
		RoleBinding: rbacv1.RoleBinding{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "test-rolebinding",
				Namespace: "test-namespace",
			},
			RoleRef: rbacv1.RoleRef{
				Kind: "Role",
				Name: "test-role",
			},
		},
		Rules: []rbacv1.PolicyRule{
			{
				Verbs:     []string{"get", "list"},
				APIGroups: []string{""},
				Resources: []string{"pods"},
			},
		},
	}
	saAccess.Spec.AccessRoleBinding = append(saAccess.Spec.AccessRoleBinding, roleBinding)

	clusterRoleBinding := struct {
		RoleBinding rbacv1.RoleBinding  `json:"roleBinding"`
		Rules       []rbacv1.PolicyRule `json:"rules,omitempty"`
	}{
		RoleBinding: rbacv1.RoleBinding{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "test-cluster-rolebinding",
				Namespace: "test-namespace",
			},
			RoleRef: rbacv1.RoleRef{
				Kind: "ClusterRole",
				Name: "test-clusterrole-via-rolebinding",
			},
		},
		Rules: []rbacv1.PolicyRule{
			{
				Verbs:     []string{"get", "list"},
				APIGroups: []string{"apps"},
				Resources: []string{"deployments"},
			},
		},
	}
	saAccess.Spec.AccessRoleBinding = append(saAccess.Spec.AccessRoleBinding, clusterRoleBinding)

	clusterRoleBindingEntry := struct {
		ClusterRoleBinding rbacv1.ClusterRoleBinding `json:"clusterRoleBinding"`
		Rules              []rbacv1.PolicyRule       `json:"rules,omitempty"`
	}{
		ClusterRoleBinding: rbacv1.ClusterRoleBinding{
			ObjectMeta: metav1.ObjectMeta{
				Name: "test-clusterrolebinding",
			},
			RoleRef: rbacv1.RoleRef{
				Kind: "ClusterRole",
				Name: "test-clusterrole",
			},
		},
		Rules: []rbacv1.PolicyRule{
			{
				Verbs:     []string{"get", "list"},
				APIGroups: []string{""},
				Resources: []string{"nodes"},
			},
		},
	}
	saAccess.Spec.AccessClusterRoleBinding = append(saAccess.Spec.AccessClusterRoleBinding, clusterRoleBindingEntry)

	duplicateClusterRoleBindingEntry := struct {
		ClusterRoleBinding rbacv1.ClusterRoleBinding `json:"clusterRoleBinding"`
		Rules              []rbacv1.PolicyRule       `json:"rules,omitempty"`
	}{
		ClusterRoleBinding: rbacv1.ClusterRoleBinding{
			ObjectMeta: metav1.ObjectMeta{
				Name: "test-clusterrolebinding",
			},
			RoleRef: rbacv1.RoleRef{
				Kind: "ClusterRole",
				Name: "test-clusterrole",
			},
		},
		Rules: []rbacv1.PolicyRule{
			{
				Verbs:     []string{"get", "list"},
				APIGroups: []string{""},
				Resources: []string{"nodes"},
			},
		},
	}
	saAccess.Spec.AccessClusterRoleBinding = append(saAccess.Spec.AccessClusterRoleBinding, duplicateClusterRoleBindingEntry)

	data, _ := json.Marshal(saAccess)

	return []string{string(data)}
}

func createSecondMockResponse() []string {
	saAccess := mockServiceAccountAccess{}
	saAccess.TypeMeta = metav1.TypeMeta{
		Kind:       "ServiceAccountAccess",
		APIVersion: "policy.kubeedge.io/v1alpha1",
	}
	saAccess.ObjectMeta = metav1.ObjectMeta{
		Name:      "test-sa-access-2",
		Namespace: "other-namespace",
	}

	roleBinding := struct {
		RoleBinding rbacv1.RoleBinding  `json:"roleBinding"`
		Rules       []rbacv1.PolicyRule `json:"rules,omitempty"`
	}{
		RoleBinding: rbacv1.RoleBinding{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "other-rolebinding",
				Namespace: "other-namespace",
			},
			RoleRef: rbacv1.RoleRef{
				Kind: "Role",
				Name: "other-role",
			},
		},
		Rules: []rbacv1.PolicyRule{
			{
				Verbs:     []string{"create", "update"},
				APIGroups: []string{"apps"},
				Resources: []string{"deployments"},
			},
		},
	}
	saAccess.Spec.AccessRoleBinding = append(saAccess.Spec.AccessRoleBinding, roleBinding)

	data, _ := json.Marshal(saAccess)

	return []string{string(data)}
}

func TestRoleGetter_GetRole(t *testing.T) {
	patches := gomonkey.NewPatches()
	defer patches.Reset()

	patches.ApplyFunc(dao.QueryMeta, func(field, value string) (*[]string, error) {
		if field == typeField && value == serviceAccountAccessVal {
			return &[]string{createMockResponse()[0]}, nil
		}
		return nil, errors.New("expected Role not found")
	})

	roleGetter := &RoleGetter{}

	role, err := roleGetter.GetRole("test-namespace", "test-role")
	assert.NoError(t, err)
	assert.NotNil(t, role)
	assert.Equal(t, "test-role", role.Name)
	assert.Equal(t, "test-namespace", role.Namespace)
	assert.Len(t, role.Rules, 1)
	assert.Equal(t, []string{"get", "list"}, role.Rules[0].Verbs)

	role, err = roleGetter.GetRole("test-namespace", "nonexistent-role")
	assert.Error(t, err)
	assert.Nil(t, role)

	role, err = roleGetter.GetRole("wrong-namespace", "test-role")
	assert.Error(t, err)
	assert.Nil(t, role)

	// Test with database error
	patches.Reset()
	patches.ApplyFunc(dao.QueryMeta, func(field, value string) (*[]string, error) {
		return nil, errors.New("database error")
	})

	role, err = roleGetter.GetRole("test-namespace", "test-role")
	assert.Error(t, err)
	assert.Nil(t, role)
}

func TestRoleBindingLister_ListRoleBindings(t *testing.T) {
	patches := gomonkey.NewPatches()
	defer patches.Reset()

	patches.ApplyFunc(dao.QueryMeta, func(field, value string) (*[]string, error) {
		if field == typeField && value == serviceAccountAccessVal {
			return &[]string{createMockResponse()[0]}, nil
		}
		return nil, errors.New("expected RoleBindings not found")
	})

	roleBindingLister := &RoleBindingLister{}

	roleBindings, err := roleBindingLister.ListRoleBindings("test-namespace")
	assert.NoError(t, err)
	assert.NotNil(t, roleBindings)
	assert.Len(t, roleBindings, 2)

	roleBindings, err = roleBindingLister.ListRoleBindings("wrong-namespace")
	assert.NoError(t, err)
	assert.Empty(t, roleBindings)
}

func TestClusterRoleGetter_GetClusterRole(t *testing.T) {
	patches := gomonkey.NewPatches()
	defer patches.Reset()

	patches.ApplyFunc(dao.QueryMeta, func(field, value string) (*[]string, error) {
		if field == typeField && value == serviceAccountAccessVal {
			return &[]string{createMockResponse()[0]}, nil
		}
		return nil, errors.New("expected ClusterRole not found")
	})

	clusterRoleGetter := &ClusterRoleGetter{}

	clusterRole, err := clusterRoleGetter.GetClusterRole("test-clusterrole-via-rolebinding")
	assert.NoError(t, err)
	assert.NotNil(t, clusterRole)
	assert.Equal(t, "test-clusterrole-via-rolebinding", clusterRole.Name)
	assert.Len(t, clusterRole.Rules, 1)
	assert.Equal(t, []string{"get", "list"}, clusterRole.Rules[0].Verbs)

	clusterRole, err = clusterRoleGetter.GetClusterRole("test-clusterrole")
	assert.NoError(t, err)
	assert.NotNil(t, clusterRole)
	assert.Equal(t, "test-clusterrole", clusterRole.Name)
	assert.Len(t, clusterRole.Rules, 1)
	assert.Equal(t, []string{"get", "list"}, clusterRole.Rules[0].Verbs)

	clusterRole, err = clusterRoleGetter.GetClusterRole("nonexistent-clusterrole")
	assert.Error(t, err)
	assert.Nil(t, clusterRole)
}

func TestClusterRoleBindingLister_ListClusterRoleBindings(t *testing.T) {
	patches := gomonkey.NewPatches()
	defer patches.Reset()

	patches.ApplyFunc(dao.QueryMeta, func(field, value string) (*[]string, error) {
		if field == typeField && value == serviceAccountAccessVal {
			return &[]string{createMockResponse()[0]}, nil
		}
		return nil, errors.New("expected ClusterRoleBindings not found")
	})

	clusterRoleBindingLister := &ClusterRoleBindingLister{}

	clusterRoleBindings, err := clusterRoleBindingLister.ListClusterRoleBindings()
	assert.NoError(t, err)
	assert.NotNil(t, clusterRoleBindings)
	assert.Len(t, clusterRoleBindings, 1)
}

func TestMultipleServiceAccountAccess(t *testing.T) {
	mockData := []string{
		createMockResponse()[0],
		createSecondMockResponse()[0],
	}

	patches := gomonkey.NewPatches()
	defer patches.Reset()

	patches.ApplyFunc(dao.QueryMeta, func(field, value string) (*[]string, error) {
		if field == typeField && value == serviceAccountAccessVal {
			return &mockData, nil
		}
		return nil, errors.New("expected ServiceAccountAccess not found")
	})

	roleGetter := &RoleGetter{}

	role, err := roleGetter.GetRole("test-namespace", "test-role")
	assert.NoError(t, err)
	assert.NotNil(t, role)
	assert.Equal(t, "test-role", role.Name)

	role, err = roleGetter.GetRole("other-namespace", "other-role")
	assert.NoError(t, err)
	assert.NotNil(t, role)
	assert.Equal(t, "other-role", role.Name)

	roleBindingLister := &RoleBindingLister{}

	roleBindings, err := roleBindingLister.ListRoleBindings("test-namespace")
	assert.NoError(t, err)
	assert.Len(t, roleBindings, 2)

	roleBindings, err = roleBindingLister.ListRoleBindings("other-namespace")
	assert.NoError(t, err)
	assert.Len(t, roleBindings, 1)
}

func TestGetterImplementation(t *testing.T) {
	fakeClient := fake.NewSimpleClientset()

	testSA := &corev1.ServiceAccount{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test-sa",
			Namespace: "test-namespace",
		},
	}

	testPod := &corev1.Pod{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test-pod",
			Namespace: "test-namespace",
		},
		Spec: corev1.PodSpec{
			Containers: []corev1.Container{
				{
					Name:  "test-container",
					Image: "test-image",
				},
			},
		},
	}

	testSecret := &corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test-secret",
			Namespace: "test-namespace",
		},
		Data: map[string][]byte{
			"token": []byte("test-token"),
		},
	}

	testNode := &corev1.Node{
		ObjectMeta: metav1.ObjectMeta{
			Name: "test-node",
		},
	}

	_, err := fakeClient.CoreV1().ServiceAccounts("test-namespace").Create(context.Background(), testSA, metav1.CreateOptions{})
	assert.NoError(t, err)

	_, err = fakeClient.CoreV1().Pods("test-namespace").Create(context.Background(), testPod, metav1.CreateOptions{})
	assert.NoError(t, err)

	_, err = fakeClient.CoreV1().Secrets("test-namespace").Create(context.Background(), testSecret, metav1.CreateOptions{})
	assert.NoError(t, err)

	_, err = fakeClient.CoreV1().Nodes().Create(context.Background(), testNode, metav1.CreateOptions{})
	assert.NoError(t, err)

	tokenGetter := NewGetterFromClient(fakeClient)
	assert.NotNil(t, tokenGetter)

	g, ok := tokenGetter.(getter)
	assert.True(t, ok)
	assert.NotNil(t, g.Client)

	sa, err := g.GetServiceAccount("test-namespace", "test-sa")
	assert.NoError(t, err)
	assert.NotNil(t, sa)
	assert.Equal(t, "test-sa", sa.Name)

	sa, err = g.GetServiceAccount("test-namespace", "nonexistent-sa")
	assert.Error(t, err)
	assert.Nil(t, sa)

	pod, err := g.GetPod("test-namespace", "test-pod")
	assert.NoError(t, err)
	assert.NotNil(t, pod)
	assert.Equal(t, "test-pod", pod.Name)

	pod, err = g.GetPod("test-namespace", "nonexistent-pod")
	assert.Error(t, err)
	assert.Nil(t, pod)

	secret, err := g.GetSecret("test-namespace", "test-secret")
	assert.NoError(t, err)
	assert.NotNil(t, secret)
	assert.Equal(t, "test-secret", secret.Name)

	secret, err = g.GetSecret("test-namespace", "nonexistent-secret")
	assert.Error(t, err)
	assert.Nil(t, secret)

	node, err := g.GetNode("test-node")
	assert.NoError(t, err)
	assert.NotNil(t, node)
	assert.Equal(t, "test-node", node.Name)

	node, err = g.GetNode("nonexistent-node")
	assert.Error(t, err)
	assert.Nil(t, node)
}

func TestMultipleGetterInstances(t *testing.T) {
	fakeClient1 := fake.NewSimpleClientset()
	fakeClient2 := fake.NewSimpleClientset()

	node1 := &corev1.Node{
		ObjectMeta: metav1.ObjectMeta{
			Name: "node-client1",
		},
	}

	node2 := &corev1.Node{
		ObjectMeta: metav1.ObjectMeta{
			Name: "node-client2",
		},
	}

	_, err := fakeClient1.CoreV1().Nodes().Create(context.Background(), node1, metav1.CreateOptions{})
	assert.NoError(t, err)

	_, err = fakeClient2.CoreV1().Nodes().Create(context.Background(), node2, metav1.CreateOptions{})
	assert.NoError(t, err)

	getter1 := NewGetterFromClient(fakeClient1)
	getter2 := NewGetterFromClient(fakeClient2)

	node, err := getter1.GetNode("node-client1")
	assert.NoError(t, err)
	assert.Equal(t, "node-client1", node.Name)

	node, err = getter1.GetNode("node-client2")
	assert.Error(t, err)

	node, err = getter2.GetNode("node-client2")
	assert.NoError(t, err)
	assert.Equal(t, "node-client2", node.Name)

	node, err = getter2.GetNode("node-client1")
	assert.Error(t, err)
}
