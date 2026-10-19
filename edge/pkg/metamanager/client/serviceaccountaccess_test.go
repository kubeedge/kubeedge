/*
Copyright 2026 The KubeEdge Authors.

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
	"testing"

	corev1 "k8s.io/api/core/v1"
	rbacv1 "k8s.io/api/rbac/v1"
	"k8s.io/apimachinery/pkg/api/equality"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	apiserverserviceaccount "k8s.io/apiserver/pkg/authentication/serviceaccount"
	"k8s.io/apiserver/pkg/authorization/authorizer"
	rbacauthorizer "k8s.io/kubernetes/plugin/pkg/auth/authorizer/rbac"

	edgecorev1alpha2 "github.com/kubeedge/api/apis/componentconfig/edgecore/v1alpha2"
	policyv1alpha1 "github.com/kubeedge/api/apis/policy/v1alpha1"
	"github.com/kubeedge/beehive/pkg/core/model"
	"github.com/kubeedge/kubeedge/edge/pkg/metamanager/dao"
	"github.com/kubeedge/kubeedge/edge/pkg/metamanager/dao/dbclient"
	"github.com/kubeedge/kubeedge/edge/pkg/metamanager/dao/models"
)

func TestRoleGetterUsesRoleBindingNamespace(t *testing.T) {
	const (
		serviceAccountNamespace = "kube-system"
		serviceAccountName      = "edge-client"
		roleName                = "reader"
		localRoleName           = "local-reader"
	)
	tenantARules := []rbacv1.PolicyRule{{
		APIGroups: []string{""},
		Resources: []string{"configmaps"},
		Verbs:     []string{"get"},
	}}
	tenantBRules := []rbacv1.PolicyRule{{
		APIGroups: []string{""},
		Resources: []string{"secrets"},
		Verbs:     []string{"list"},
	}}
	localRules := []rbacv1.PolicyRule{{
		APIGroups: []string{""},
		Resources: []string{"services"},
		Verbs:     []string{"get"},
	}}
	subject := rbacv1.Subject{
		Kind:      rbacv1.ServiceAccountKind,
		Name:      serviceAccountName,
		Namespace: serviceAccountNamespace,
	}
	access := &policyv1alpha1.ServiceAccountAccess{
		ObjectMeta: metav1.ObjectMeta{Name: serviceAccountName, Namespace: serviceAccountNamespace},
		Spec: policyv1alpha1.AccessSpec{
			ServiceAccount: corev1.ServiceAccount{
				ObjectMeta: metav1.ObjectMeta{Name: serviceAccountName, Namespace: serviceAccountNamespace},
			},
			AccessRoleBinding: []policyv1alpha1.AccessRoleBinding{
				{
					RoleBinding: rbacv1.RoleBinding{
						ObjectMeta: metav1.ObjectMeta{Name: "tenant-a-reader", Namespace: "tenant-a"},
						Subjects:   []rbacv1.Subject{subject},
						RoleRef:    rbacv1.RoleRef{APIGroup: rbacv1.GroupName, Kind: roleKind, Name: roleName},
					},
					Rules: tenantARules,
				},
				{
					RoleBinding: rbacv1.RoleBinding{
						ObjectMeta: metav1.ObjectMeta{Name: "tenant-b-reader", Namespace: "tenant-b"},
						Subjects:   []rbacv1.Subject{subject},
						RoleRef:    rbacv1.RoleRef{APIGroup: rbacv1.GroupName, Kind: roleKind, Name: roleName},
					},
					Rules: tenantBRules,
				},
				{
					RoleBinding: rbacv1.RoleBinding{
						ObjectMeta: metav1.ObjectMeta{Name: "local-reader", Namespace: serviceAccountNamespace},
						Subjects:   []rbacv1.Subject{subject},
						RoleRef:    rbacv1.RoleRef{APIGroup: rbacv1.GroupName, Kind: roleKind, Name: localRoleName},
					},
					Rules: localRules,
				},
			},
		},
	}
	storeServiceAccountAccess(t, access)

	getter := &RoleGetter{}
	for _, test := range []struct {
		name      string
		namespace string
		roleName  string
		wantRules []rbacv1.PolicyRule
	}{
		{name: "tenant a", namespace: "tenant-a", roleName: roleName, wantRules: tenantARules},
		{name: "tenant b", namespace: "tenant-b", roleName: roleName, wantRules: tenantBRules},
		{name: "same namespace compatibility", namespace: serviceAccountNamespace, roleName: localRoleName, wantRules: localRules},
	} {
		t.Run(test.name, func(t *testing.T) {
			role, err := getter.GetRole(context.Background(), test.namespace, test.roleName)
			if err != nil {
				t.Fatalf("GetRole(%q, %q) returned error: %v", test.namespace, test.roleName, err)
			}
			if role.Namespace != test.namespace {
				t.Errorf("GetRole(%q, %q) namespace = %q, want %q", test.namespace, test.roleName, role.Namespace, test.namespace)
			}
			if role.Name != test.roleName {
				t.Errorf("GetRole(%q, %q) name = %q, want %q", test.namespace, test.roleName, role.Name, test.roleName)
			}
			if !equality.Semantic.DeepEqual(role.Rules, test.wantRules) {
				t.Errorf("GetRole(%q, %q) rules = %#v, want %#v", test.namespace, test.roleName, role.Rules, test.wantRules)
			}
		})
	}

	if role, err := getter.GetRole(context.Background(), serviceAccountNamespace, roleName); err == nil {
		t.Fatalf("GetRole(%q, %q) returned fabricated role %#v", serviceAccountNamespace, roleName, role)
	}

	lister := &RoleBindingLister{}
	for _, test := range []struct {
		namespace string
		name      string
	}{
		{namespace: "tenant-a", name: "tenant-a-reader"},
		{namespace: "tenant-b", name: "tenant-b-reader"},
		{namespace: serviceAccountNamespace, name: "local-reader"},
	} {
		bindings, err := lister.ListRoleBindings(context.Background(), test.namespace)
		if err != nil {
			t.Fatalf("ListRoleBindings(%q) returned error: %v", test.namespace, err)
		}
		if len(bindings) != 1 || bindings[0].Namespace != test.namespace || bindings[0].Name != test.name {
			t.Fatalf("ListRoleBindings(%q) = %#v, want %s/%s", test.namespace, bindings, test.namespace, test.name)
		}
	}
	bindings, err := lister.ListRoleBindings(context.Background(), "tenant-c")
	if err != nil {
		t.Fatalf("ListRoleBindings(%q) returned error: %v", "tenant-c", err)
	}
	if len(bindings) != 0 {
		t.Fatalf("ListRoleBindings(%q) = %#v, want no bindings", "tenant-c", bindings)
	}

	rbacAuthorizer := rbacauthorizer.New(
		&RoleGetter{},
		&RoleBindingLister{},
		&ClusterRoleGetter{},
		&ClusterRoleBindingLister{},
	)
	for _, test := range []struct {
		name          string
		userNamespace string
		verb          string
		namespace     string
		resource      string
		want          authorizer.Decision
	}{
		{
			name:          "cross namespace role allows tenant a",
			userNamespace: serviceAccountNamespace,
			verb:          "get",
			namespace:     "tenant-a",
			resource:      "configmaps",
			want:          authorizer.DecisionAllow,
		},
		{
			name:          "same role name resolves tenant b rules",
			userNamespace: serviceAccountNamespace,
			verb:          "list",
			namespace:     "tenant-b",
			resource:      "secrets",
			want:          authorizer.DecisionAllow,
		},
		{
			name:          "legacy same namespace role remains allowed",
			userNamespace: serviceAccountNamespace,
			verb:          "get",
			namespace:     serviceAccountNamespace,
			resource:      "services",
			want:          authorizer.DecisionAllow,
		},
		{
			name:          "role does not grant another namespace",
			userNamespace: serviceAccountNamespace,
			verb:          "get",
			namespace:     "tenant-c",
			resource:      "configmaps",
			want:          authorizer.DecisionNoOpinion,
		},
		{
			name:          "same service account name in another namespace is denied",
			userNamespace: "tenant-a",
			verb:          "get",
			namespace:     "tenant-a",
			resource:      "configmaps",
			want:          authorizer.DecisionNoOpinion,
		},
	} {
		t.Run("authorize "+test.name, func(t *testing.T) {
			decision, reason, err := rbacAuthorizer.Authorize(context.Background(), authorizer.AttributesRecord{
				User:            apiserverserviceaccount.UserInfo(test.userNamespace, serviceAccountName, "edge-client-uid"),
				Verb:            test.verb,
				Namespace:       test.namespace,
				APIGroup:        "",
				Resource:        test.resource,
				Name:            "demo",
				ResourceRequest: true,
			})
			if err != nil {
				t.Fatalf("Authorize() returned error: %v", err)
			}
			if decision != test.want {
				t.Errorf("Authorize() decision = %v, want %v, reason: %s", decision, test.want, reason)
			}
		})
	}

	access.Spec.AccessRoleBinding = access.Spec.AccessRoleBinding[1:]
	upsertServiceAccountAccess(t, access)
	decision, reason, err := rbacAuthorizer.Authorize(context.Background(), authorizer.AttributesRecord{
		User:            apiserverserviceaccount.UserInfo(serviceAccountNamespace, serviceAccountName, "edge-client-uid"),
		Verb:            "get",
		Namespace:       "tenant-a",
		APIGroup:        "",
		Resource:        "configmaps",
		Name:            "demo",
		ResourceRequest: true,
	})
	if err != nil {
		t.Fatalf("Authorize() after rolebinding removal returned error: %v", err)
	}
	if decision != authorizer.DecisionNoOpinion {
		t.Errorf("Authorize() after rolebinding removal decision = %v, want %v, reason: %s", decision, authorizer.DecisionNoOpinion, reason)
	}
}

func storeServiceAccountAccess(t *testing.T, access *policyv1alpha1.ServiceAccountAccess) {
	t.Helper()
	upsertServiceAccountAccess(t, access)
	key := serviceAccountAccessTestKey(access)
	metaService := dbclient.NewMetaService()
	t.Cleanup(func() {
		if err := metaService.DeleteMetaByKey(key); err != nil {
			t.Errorf("failed to delete serviceaccountaccess test data: %v", err)
		}
	})
}

func upsertServiceAccountAccess(t *testing.T, access *policyv1alpha1.ServiceAccountAccess) {
	t.Helper()
	dao.Init("file:serviceaccountaccess-client-test?mode=memory&cache=shared", &edgecorev1alpha2.MetaManager{Enable: true})
	value, err := json.Marshal(access)
	if err != nil {
		t.Fatalf("failed to marshal serviceaccountaccess: %v", err)
	}
	metaService := dbclient.NewMetaService()
	if err := metaService.InsertOrUpdate(&models.Meta{
		Key:   serviceAccountAccessTestKey(access),
		Type:  model.ResourceTypeSaAccess,
		Value: string(value),
	}); err != nil {
		t.Fatalf("failed to store serviceaccountaccess: %v", err)
	}
}

func serviceAccountAccessTestKey(access *policyv1alpha1.ServiceAccountAccess) string {
	return "serviceaccountaccess-test/" + access.Namespace + "/" + access.Name
}
