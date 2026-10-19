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

package metamanager

import (
	"context"
	"testing"

	corev1 "k8s.io/api/core/v1"
	rbacv1 "k8s.io/api/rbac/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	apiserverserviceaccount "k8s.io/apiserver/pkg/authentication/serviceaccount"
	"k8s.io/apiserver/pkg/authorization/authorizer"
	rbacauthorizer "k8s.io/kubernetes/plugin/pkg/auth/authorizer/rbac"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"

	edgecorev1alpha2 "github.com/kubeedge/api/apis/componentconfig/edgecore/v1alpha2"
	policyv1alpha1 "github.com/kubeedge/api/apis/policy/v1alpha1"
	"github.com/kubeedge/beehive/pkg/core/model"
	policycontroller "github.com/kubeedge/kubeedge/cloud/pkg/policycontroller/manager"
	metaclient "github.com/kubeedge/kubeedge/edge/pkg/metamanager/client"
	"github.com/kubeedge/kubeedge/edge/pkg/metamanager/dao"
)

func TestCrossNamespaceServiceAccountAuthorizationFlow(t *testing.T) {
	const (
		serviceAccountNamespace = "kube-system"
		serviceAccountName      = "edge-client-integration"
		targetNamespace         = "tenant-a"
		roleName                = "configmap-reader"
	)
	rules := []rbacv1.PolicyRule{{
		APIGroups: []string{""},
		Resources: []string{"configmaps"},
		Verbs:     []string{"get"},
	}}
	serviceAccount := corev1.ServiceAccount{
		ObjectMeta: metav1.ObjectMeta{
			Name:      serviceAccountName,
			Namespace: serviceAccountNamespace,
			UID:       "edge-client-integration-uid",
		},
	}
	role := &rbacv1.Role{
		ObjectMeta: metav1.ObjectMeta{Name: roleName, Namespace: targetNamespace},
		Rules:      rules,
	}
	roleBinding := &rbacv1.RoleBinding{
		ObjectMeta: metav1.ObjectMeta{Name: "allow-edge-client", Namespace: targetNamespace},
		Subjects: []rbacv1.Subject{{
			Kind:      rbacv1.ServiceAccountKind,
			Name:      serviceAccountName,
			Namespace: serviceAccountNamespace,
		}},
		RoleRef: rbacv1.RoleRef{
			APIGroup: rbacv1.GroupName,
			Kind:     "Role",
			Name:     roleName,
		},
	}

	cloudScheme := runtime.NewScheme()
	if err := rbacv1.AddToScheme(cloudScheme); err != nil {
		t.Fatalf("failed to add rbacv1 scheme: %v", err)
	}
	cloudClient := fake.NewClientBuilder().WithScheme(cloudScheme).WithObjects(role, roleBinding).Build()
	cloudController := &policycontroller.Controller{Client: cloudClient, Reader: cloudClient}
	access := &policyv1alpha1.ServiceAccountAccess{
		ObjectMeta: metav1.ObjectMeta{Name: serviceAccountName, Namespace: serviceAccountNamespace},
		Spec:       policyv1alpha1.AccessSpec{ServiceAccount: serviceAccount},
	}
	cloudController.VisitRulesFor(
		context.Background(),
		apiserverserviceaccount.UserInfo(serviceAccountNamespace, serviceAccountName, string(serviceAccount.UID)),
		serviceAccountNamespace,
		access,
	)
	if len(access.Spec.AccessRoleBinding) != 1 {
		t.Fatalf("CloudCore collected %d rolebindings, want 1: %#v", len(access.Spec.AccessRoleBinding), access.Spec.AccessRoleBinding)
	}
	collected := access.Spec.AccessRoleBinding[0]
	if collected.RoleBinding.Namespace != targetNamespace || collected.RoleBinding.RoleRef.Name != roleName {
		t.Fatalf("CloudCore collected rolebinding %#v, want namespace %q and role %q", collected.RoleBinding, targetNamespace, roleName)
	}

	dao.Init("file:serviceaccountaccess-integration-test?mode=memory&cache=shared", &edgecorev1alpha2.MetaManager{Enable: true})
	manager := newMetaManager(true)
	resource := serviceAccountNamespace + "/" + model.ResourceTypeSaAccess + "/" + serviceAccountName
	t.Cleanup(func() {
		if err := manager.metaService.DeleteMetaByKey(resource); err != nil {
			t.Errorf("failed to clean serviceaccountaccess integration data: %v", err)
		}
	})
	insertMessage := model.NewMessage("").
		BuildRouter(CloudControllerModel, GroupResource, resource, model.InsertOperation).
		FillBody(access)
	if err := manager.handleMessage(insertMessage); err != nil {
		t.Fatalf("EdgeCore failed to store CloudCore serviceaccountaccess: %v", err)
	}

	rbacAuthorizer := rbacauthorizer.New(
		&metaclient.RoleGetter{},
		&metaclient.RoleBindingLister{},
		&metaclient.ClusterRoleGetter{},
		&metaclient.ClusterRoleBindingLister{},
	)
	userInfo := apiserverserviceaccount.UserInfo(serviceAccountNamespace, serviceAccountName, string(serviceAccount.UID))
	authorize := func(namespace string) authorizer.Decision {
		t.Helper()
		decision, reason, err := rbacAuthorizer.Authorize(context.Background(), authorizer.AttributesRecord{
			User:            userInfo,
			Verb:            "get",
			Namespace:       namespace,
			APIGroup:        "",
			Resource:        "configmaps",
			Name:            "demo",
			ResourceRequest: true,
		})
		if err != nil {
			t.Fatalf("Authorize(%q) returned error: %v", namespace, err)
		}
		t.Logf("Authorize(%q) reason: %s", namespace, reason)
		return decision
	}
	if decision := authorize(targetNamespace); decision != authorizer.DecisionAllow {
		t.Fatalf("Authorize(%q) = %v, want %v", targetNamespace, decision, authorizer.DecisionAllow)
	}
	if decision := authorize("tenant-b"); decision != authorizer.DecisionNoOpinion {
		t.Fatalf("Authorize(%q) = %v, want %v", "tenant-b", decision, authorizer.DecisionNoOpinion)
	}

	deleteMessage := model.NewMessage("").
		BuildRouter(CloudControllerModel, GroupResource, resource, model.DeleteOperation)
	if err := manager.handleMessage(deleteMessage); err != nil {
		t.Fatalf("EdgeCore failed to delete serviceaccountaccess: %v", err)
	}
	if decision := authorize(targetNamespace); decision != authorizer.DecisionNoOpinion {
		t.Fatalf("Authorize(%q) after delete = %v, want %v", targetNamespace, decision, authorizer.DecisionNoOpinion)
	}
}
