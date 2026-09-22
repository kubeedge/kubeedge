package controller

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"reflect"
	"sort"
	"sync"
	"testing"
	"time"

	v1 "k8s.io/api/core/v1"
	rbacv1 "k8s.io/api/rbac/v1"
	"k8s.io/apimachinery/pkg/api/equality"
	apierror "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/apiserver/pkg/authentication/user"
	"k8s.io/client-go/util/workqueue"
	controllerruntime "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
	"sigs.k8s.io/controller-runtime/pkg/event"
	"sigs.k8s.io/controller-runtime/pkg/handler"

	policyv1alpha1 "github.com/kubeedge/api/apis/policy/v1alpha1"
	"github.com/kubeedge/beehive/pkg/common"
	beehiveContext "github.com/kubeedge/beehive/pkg/core/context"
	"github.com/kubeedge/beehive/pkg/core/model"
	"github.com/kubeedge/kubeedge/cloud/pkg/common/messagelayer"
	"github.com/kubeedge/kubeedge/cloud/pkg/common/modules"
)

type recordingMessageLayer struct {
	messages []model.Message
}

func (m *recordingMessageLayer) Send(message model.Message) error {
	m.messages = append(m.messages, message)
	return nil
}

func (*recordingMessageLayer) Receive() (model.Message, error) {
	return model.Message{}, nil
}

func (*recordingMessageLayer) Response(model.Message) error {
	return nil
}

func TestIntersectSlice(t *testing.T) {
	tests := []struct {
		name string
		a    []string
		b    []string
		want []string
	}{
		{
			name: "test1",
			a:    []string{"a", "b", "c"},
			b:    []string{"b", "c", "d"},
			want: []string{"b", "c"},
		},
		{
			name: "test2",
			a:    []string{"a", "b", "c"},
			b:    []string{"d", "e", "f"},
			want: []string{},
		},
		{
			name: "test3",
			a:    []string{"a", "b", "c"},
			b:    []string{"a", "b", "c"},
			want: []string{"a", "b", "c"},
		},
		{
			name: "test4",
			a:    []string{},
			b:    []string{"a", "b", "c"},
			want: []string{},
		},
		{
			name: "test5",
			a:    []string{"a", "b", "c"},
			b:    []string{},
			want: []string{},
		},
		{
			name: "test6",
			a:    []string{},
			b:    []string{},
			want: []string{},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := intersectSlice(tt.a, tt.b); !reflect.DeepEqual(got, tt.want) {
				t.Errorf("intersectSlice() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestSubtractSlice(t *testing.T) {
	tests := []struct {
		name string
		a    []string
		b    []string
		want []string
	}{
		{
			name: "test1",
			a:    []string{"a", "b", "c"},
			b:    []string{"b", "c", "d"},
			want: []string{"a"},
		},
		{
			name: "test2",
			a:    []string{"a", "b", "c"},
			b:    []string{"d", "e", "f"},
			want: []string{"a", "b", "c"},
		},
		{
			name: "test3",
			a:    []string{"a", "b", "c"},
			b:    []string{"a", "b", "c"},
			want: []string{},
		},
		{
			name: "test4",
			a:    []string{},
			b:    []string{"a", "b", "c"},
			want: []string{},
		},
		{
			name: "test5",
			a:    []string{"a", "b", "c"},
			b:    []string{},
			want: []string{"a", "b", "c"},
		},
		{
			name: "test6",
			a:    []string{},
			b:    []string{},
			want: []string{},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := subtractSlice(tt.b, tt.a); !reflect.DeepEqual(got, tt.want) {
				t.Errorf("subtractSlice() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestSend2EdgeSetsServiceAccountAccessGVK(t *testing.T) {
	messageLayer := &recordingMessageLayer{}
	controller := &Controller{MessageLayer: messageLayer}
	access := &policyv1alpha1.ServiceAccountAccess{
		ObjectMeta: metav1.ObjectMeta{
			Name:            "cilium",
			Namespace:       "kube-system",
			UID:             "cilium-uid",
			ResourceVersion: "2",
		},
		Spec: policyv1alpha1.AccessSpec{
			ServiceAccount: v1.ServiceAccount{ObjectMeta: metav1.ObjectMeta{Name: "cilium", Namespace: "kube-system"}},
		},
	}

	controller.send2Edge(access, []string{"edge-node"}, model.UpdateOperation)

	if len(messageLayer.messages) != 1 {
		t.Fatalf("send2Edge() sent %d messages, want 1", len(messageLayer.messages))
	}
	sentAccess, ok := messageLayer.messages[0].Content.(*policyv1alpha1.ServiceAccountAccess)
	if !ok {
		t.Fatalf("send2Edge() content type = %T, want *ServiceAccountAccess", messageLayer.messages[0].Content)
	}
	wantGVK := policyv1alpha1.SchemeGroupVersion.WithKind("ServiceAccountAccess")
	if got := sentAccess.GroupVersionKind(); got != wantGVK {
		t.Fatalf("send2Edge() content GVK = %s, want %s", got, wantGVK)
	}
	if got := access.GroupVersionKind(); !got.Empty() {
		t.Fatalf("send2Edge() mutated source GVK to %s", got)
	}
}

func TestSortAccessRoleBindings(t *testing.T) {
	bindings := []policyv1alpha1.AccessRoleBinding{
		{RoleBinding: rbacv1.RoleBinding{ObjectMeta: metav1.ObjectMeta{Name: "reader", Namespace: "tenant-b"}}},
		{RoleBinding: rbacv1.RoleBinding{ObjectMeta: metav1.ObjectMeta{Name: "writer", Namespace: "tenant-a"}}},
		{RoleBinding: rbacv1.RoleBinding{ObjectMeta: metav1.ObjectMeta{Name: "reader", Namespace: "tenant-a"}}},
		{RoleBinding: rbacv1.RoleBinding{ObjectMeta: metav1.ObjectMeta{Name: "admin", Namespace: "tenant-a"}}},
	}

	sortAccessRoleBindings(bindings)

	got := make([]string, 0, len(bindings))
	for _, binding := range bindings {
		got = append(got, binding.RoleBinding.Namespace+"/"+binding.RoleBinding.Name)
	}
	want := []string{
		"tenant-a/admin",
		"tenant-a/reader",
		"tenant-a/writer",
		"tenant-b/reader",
	}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("sortAccessRoleBindings() = %v, want %v", got, want)
	}
}

func TestAppliesTo(t *testing.T) {
	tests := []struct {
		subjects  []rbacv1.Subject
		user      user.Info
		namespace string
		appliesTo bool
		index     int
		testCase  string
	}{
		{
			subjects: []rbacv1.Subject{
				{Kind: rbacv1.UserKind, Name: "foobar"},
			},
			user:      &user.DefaultInfo{Name: "foobar"},
			appliesTo: true,
			index:     0,
			testCase:  "single subject that matches username",
		},
		{
			subjects: []rbacv1.Subject{
				{Kind: rbacv1.UserKind, Name: "barfoo"},
				{Kind: rbacv1.UserKind, Name: "foobar"},
			},
			user:      &user.DefaultInfo{Name: "foobar"},
			appliesTo: true,
			index:     1,
			testCase:  "multiple subjects, one that matches username",
		},
		{
			subjects: []rbacv1.Subject{
				{Kind: rbacv1.UserKind, Name: "barfoo"},
				{Kind: rbacv1.UserKind, Name: "foobar"},
			},
			user:      &user.DefaultInfo{Name: "zimzam"},
			appliesTo: false,
			testCase:  "multiple subjects, none that match username",
		},
		{
			subjects: []rbacv1.Subject{
				{Kind: rbacv1.UserKind, Name: "barfoo"},
				{Kind: rbacv1.GroupKind, Name: "foobar"},
			},
			user:      &user.DefaultInfo{Name: "zimzam", Groups: []string{"foobar"}},
			appliesTo: true,
			index:     1,
			testCase:  "multiple subjects, one that match group",
		},
		{
			subjects: []rbacv1.Subject{
				{Kind: rbacv1.UserKind, Name: "barfoo"},
				{Kind: rbacv1.GroupKind, Name: "foobar"},
			},
			user:      &user.DefaultInfo{Name: "zimzam", Groups: []string{"foobar"}},
			namespace: "namespace1",
			appliesTo: true,
			index:     1,
			testCase:  "multiple subjects, one that match group, should ignore namespace",
		},
		{
			subjects: []rbacv1.Subject{
				{Kind: rbacv1.UserKind, Name: "barfoo"},
				{Kind: rbacv1.GroupKind, Name: "foobar"},
				{Kind: rbacv1.ServiceAccountKind, Namespace: "kube-system", Name: "default"},
			},
			user:      &user.DefaultInfo{Name: "system:serviceaccount:kube-system:default"},
			namespace: "default",
			appliesTo: true,
			index:     2,
			testCase:  "multiple subjects with a service account that matches",
		},
		{
			subjects: []rbacv1.Subject{
				{Kind: rbacv1.ServiceAccountKind, Name: "default"},
			},
			user:      &user.DefaultInfo{Name: "system:serviceaccount:tenant-a:default"},
			namespace: "tenant-a",
			appliesTo: true,
			index:     0,
			testCase:  "service account subject without namespace defaults to rolebinding namespace",
		},
		{
			subjects: []rbacv1.Subject{
				{Kind: rbacv1.ServiceAccountKind, Name: "default"},
			},
			user:      &user.DefaultInfo{Name: "system:serviceaccount:kube-system:default"},
			namespace: "tenant-a",
			appliesTo: false,
			testCase:  "service account subject without namespace does not match another namespace",
		},
		{
			subjects: []rbacv1.Subject{
				{Kind: rbacv1.UserKind, Name: "*"},
			},
			user:      &user.DefaultInfo{Name: "foobar"},
			namespace: "default",
			appliesTo: false,
			testCase:  "* user subject name doesn't match all users",
		},
		{
			subjects: []rbacv1.Subject{
				{Kind: rbacv1.GroupKind, Name: user.AllAuthenticated},
				{Kind: rbacv1.GroupKind, Name: user.AllUnauthenticated},
			},
			user:      &user.DefaultInfo{Name: "foobar", Groups: []string{user.AllAuthenticated}},
			namespace: "default",
			appliesTo: true,
			index:     0,
			testCase:  "binding to all authenticated and unauthenticated subjects matches authenticated user",
		},
		{
			subjects: []rbacv1.Subject{
				{Kind: rbacv1.GroupKind, Name: user.AllAuthenticated},
				{Kind: rbacv1.GroupKind, Name: user.AllUnauthenticated},
			},
			user:      &user.DefaultInfo{Name: "system:anonymous", Groups: []string{user.AllUnauthenticated}},
			namespace: "default",
			appliesTo: true,
			index:     1,
			testCase:  "binding to all authenticated and unauthenticated subjects matches anonymous user",
		},
	}

	for _, tc := range tests {
		gotIndex, got := appliesTo(tc.user, tc.subjects, tc.namespace)
		if got != tc.appliesTo {
			t.Errorf("case %q want appliesTo=%t, got appliesTo=%t", tc.testCase, tc.appliesTo, got)
		}
		if gotIndex != tc.index {
			t.Errorf("case %q want index %d, got %d", tc.testCase, tc.index, gotIndex)
		}
	}
}

func newServiceAccount() *v1.ServiceAccount {
	return &v1.ServiceAccount{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "sa1",
			Namespace: "ns1",
		},
	}
}

func TestNewSaAccessObject(t *testing.T) {
	tests := []struct {
		name   string
		sa     *v1.ServiceAccount
		result *policyv1alpha1.ServiceAccountAccess
	}{
		{
			name: "test NewSaAccessObject",
			sa:   newServiceAccount(),
			result: &policyv1alpha1.ServiceAccountAccess{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "sa1",
					Namespace: "ns1",
				},
				Spec: policyv1alpha1.AccessSpec{
					ServiceAccount: *newServiceAccount(),
				},
			},
		},
	}
	for _, tc := range tests {
		got := newSaAccessObject(*tc.sa)
		if !reflect.DeepEqual(got, tc.result) {
			t.Errorf("case %q want=%v, got=%v", tc.name, tc.result, got)
		}
	}
}

var podStr1 = `{
  "apiVersion": "v1",
  "kind": "Pod",
  "metadata": {
    "name": "pod1",
    "namespace": "my-namespace"
  },
  "spec": {
    "serviceAccountName": "sa1",
    "nodeName": "my-node",
    "containers": [
      {
        "name": "my-container",
        "image": "my-image",
        "ports": [
          {
            "containerPort": 80,
            "protocol": "TCP"
          }
        ]
      }
    ]
  }
}`

var podDelStr1 = `{
	"apiVersion": "v1",
	"kind": "Pod",
	"metadata": {
	  "name": "podDel",
	  "namespace": "my-namespace",
	  "deletionTimestamp": "2022-01-01T00:00:00Z"
	},
	"spec": {
	  "serviceAccountName": "sa1",
	  "nodeName": "my-node",
	  "containers": [
		{
		  "name": "my-container",
		  "image": "my-image",
		  "ports": [
			{
			  "containerPort": 80,
			  "protocol": "TCP"
			}
		  ]
		}
	  ]
	}
  }`

var podStr2 = `{
  "apiVersion": "v1",
  "kind": "Pod",
  "metadata": {
    "name": "pod2",
    "namespace": "my-namespace"
  },
  "spec": {
    "serviceAccountName": "sa1",
    "nodeName": "my-node-2",
    "containers": [
      {
        "name": "my-container-2",
        "image": "my-image",
        "ports": [
          {
            "containerPort": 80,
            "protocol": "TCP"
          }
        ]
      }
    ]
  }
}`

var podStrWithoutNodeName = `{
  "apiVersion": "v1",
  "kind": "Pod",
  "metadata": {
    "name": "podwnn1",
    "namespace": "my-namespace"
  },
  "spec": {
    "serviceAccountName": "sa1",
    "containers": [
      {
        "name": "my-container",
        "image": "my-image",
        "ports": [
          {
            "containerPort": 80,
            "protocol": "TCP"
          }
        ]
      }
    ]
  }
}`

var podStrWithoutSa = `{
  "apiVersion": "v1",
  "kind": "Pod",
  "metadata": {
    "name": "podwnn1",
    "namespace": "my-namespace"
  },
  "spec": {
    "nodeName": "my-node",
    "containers": [
      {
        "name": "my-container",
        "image": "my-image",
        "ports": [
          {
            "containerPort": 80,
            "protocol": "TCP"
          }
        ]
      }
    ]
  }
}`

var saStr1 = `{
    "apiVersion": "v1",
    "kind": "ServiceAccount",
    "metadata": {
        "name": "sa1",
        "namespace": "my-namespace",
        "resourceVersion": "999"
    }
}`

var saStr2 = `{
    "apiVersion": "v1",
    "kind": "ServiceAccount",
    "metadata": {
        "name": "sa2",
        "namespace": "my-namespace",
        "resourceVersion": "999"
    }
}`

var saNsStr = `{
    "apiVersion": "v1",
    "kind": "ServiceAccount",
    "metadata": {
        "name": "sa1",
        "namespace": "my-namespace2"
    }
}`

var roleStr1 = `{
    "apiVersion": "rbac.authorization.k8s.io/v1",
    "kind": "Role",
    "metadata": {
        "name": "role1",
        "namespace": "my-namespace",
        "resourceVersion": "999"
    },
    "rules": [
        {
            "apiGroups": [""],
            "resources": ["pods"],
            "verbs": ["get", "list", "watch"]
        },
        {
            "apiGroups": ["apps"],
            "resources": ["deployments"],
            "verbs": ["get", "list", "watch"]
        }
    ]
}`

var roleNsStr1 = `{
    "apiVersion": "rbac.authorization.k8s.io/v1",
    "kind": "Role",
    "metadata": {
        "name": "role1",
        "namespace": "my-namespace2",
        "resourceVersion": "999"
    },
    "rules": [
        {
            "apiGroups": [""],
            "resources": ["pods"],
            "verbs": ["get", "list", "watch"]
        },
        {
            "apiGroups": ["apps"],
            "resources": ["deployments"],
            "verbs": ["get", "list", "watch"]
        }
    ]
}`

var roleStr2 = `{
    "apiVersion": "rbac.authorization.k8s.io/v1",
    "kind": "Role",
    "metadata": {
        "name": "role2",
        "namespace": "my-namespace",
        "resourceVersion": "999"
    },
    "rules": [
        {
            "apiGroups": [""],
            "resources": ["pods"],
            "verbs": ["get", "list", "watch"]
        },
        {
            "apiGroups": ["apps"],
            "resources": ["configmaps"],
            "verbs": ["get", "list", "watch"]
        }
    ]
}`

var rbStr1 = `{
    "apiVersion": "rbac.authorization.k8s.io/v1",
    "kind": "RoleBinding",
    "metadata": {
        "name": "rb1",
        "namespace": "my-namespace",
        "resourceVersion": "999"
    },
    "roleRef": {
        "apiGroup": "rbac.authorization.k8s.io",
        "kind": "Role",
        "name": "role1"
    },
    "subjects": [
        {
            "kind": "ServiceAccount",
            "name": "sa1",
            "namespace": "my-namespace"
        }
    ]
}`

var rbStr2 = `{
    "apiVersion": "rbac.authorization.k8s.io/v1",
    "kind": "RoleBinding",
    "metadata": {
        "name": "rb2",
        "namespace": "my-namespace",
        "resourceVersion": "999"
    },
    "roleRef": {
        "apiGroup": "rbac.authorization.k8s.io",
        "kind": "Role",
        "name": "role2"
    },
    "subjects": [
        {
            "kind": "ServiceAccount",
            "name": "sa1",
            "namespace": "my-namespace"
        }
    ]
}`

var rbWithCrStr = `{
    "apiVersion": "rbac.authorization.k8s.io/v1",
    "kind": "RoleBinding",
    "metadata": {
        "name": "rbWithCr",
        "namespace": "my-namespace",
        "resourceVersion": "999"
    },
    "roleRef": {
        "apiGroup": "rbac.authorization.k8s.io",
        "kind": "ClusterRole",
        "name": "cr1"
    },
    "subjects": [
        {
            "kind": "ServiceAccount",
            "name": "sa1",
            "namespace": "my-namespace"
        }
    ]
}`

var crStr1 = `{
    "apiVersion": "rbac.authorization.k8s.io/v1",
    "kind": "ClusterRole",
    "metadata": {
        "name": "cr1",
        "resourceVersion": "999"
    },
    "rules": [
        {
            "apiGroups": [""],
            "resources": ["pods"],
            "verbs": ["get", "list", "watch"]
        },
        {
            "apiGroups": ["apps"],
            "resources": ["deployments"],
            "verbs": ["get", "list", "watch"]
        }
    ]
}`

var crbStr1 = `{
    "apiVersion": "rbac.authorization.k8s.io/v1",
    "kind": "ClusterRoleBinding",
    "metadata": {
        "name": "crb1",
        "resourceVersion": "999"
    },
    "roleRef": {
        "apiGroup": "rbac.authorization.k8s.io",
        "kind": "ClusterRole",
        "name": "cr1"
    },
    "subjects": [
        {
            "kind": "ServiceAccount",
            "name": "sa1",
            "namespace": "my-namespace"
        }
    ]
}`

var crbStr2 = `{
    "apiVersion": "rbac.authorization.k8s.io/v1",
    "kind": "ClusterRoleBinding",
    "metadata": {
        "name": "crb2",
        "resourceVersion": "999"
    },
    "roleRef": {
        "apiGroup": "rbac.authorization.k8s.io",
        "kind": "ClusterRole",
        "name": "cr1"
    },
    "subjects": [
        {
            "kind": "ServiceAccount",
            "name": "sa1",
            "namespace": "my-namespace"
        }
    ]
}`

func TestMapRolesFuncAndFilterObject(t *testing.T) {
	var sa1 v1.ServiceAccount
	err := json.Unmarshal([]byte(saStr1), &sa1)
	if err != nil {
		t.Errorf("Failed to unmarshal sa1: %v", err)
	}
	var sa2 v1.ServiceAccount
	err = json.Unmarshal([]byte(saStr2), &sa2)
	if err != nil {
		t.Errorf("Failed to unmarshal sa2: %v", err)
	}
	var saNs v1.ServiceAccount
	err = json.Unmarshal([]byte(saNsStr), &saNs)
	if err != nil {
		t.Errorf("Failed to unmarshal sa2: %v", err)
	}
	var role1 rbacv1.Role
	err = json.Unmarshal([]byte(roleStr1), &role1)
	if err != nil {
		t.Errorf("Failed to unmarshal role1: %v", err)
	}
	var roleNs rbacv1.Role
	err = json.Unmarshal([]byte(roleNsStr1), &roleNs)
	if err != nil {
		t.Errorf("Failed to unmarshal roleNs: %v", err)
	}
	var role2 rbacv1.Role
	err = json.Unmarshal([]byte(roleStr2), &role2)
	if err != nil {
		t.Errorf("Failed to unmarshal role2: %v", err)
	}
	var rb1 rbacv1.RoleBinding
	err = json.Unmarshal([]byte(rbStr1), &rb1)
	if err != nil {
		t.Errorf("Failed to unmarshal rb1: %v", err)
	}
	var rb2 rbacv1.RoleBinding
	err = json.Unmarshal([]byte(rbStr2), &rb2)
	if err != nil {
		t.Errorf("Failed to unmarshal rb2: %v", err)
	}
	var rbWithCr rbacv1.RoleBinding
	err = json.Unmarshal([]byte(rbWithCrStr), &rbWithCr)
	if err != nil {
		t.Errorf("Failed to unmarshal rbWithCr: %v", err)
	}
	crossNamespaceRoleBinding := rb1.DeepCopy()
	crossNamespaceRoleBinding.Name = "rb-cross-namespace"
	crossNamespaceRoleBinding.Namespace = roleNs.Namespace
	crossNamespaceRoleBinding.RoleRef.Name = roleNs.Name
	crossNamespaceRoleBinding.Subjects[0].Namespace = sa1.Namespace
	crossNamespaceClusterRoleBinding := rbWithCr.DeepCopy()
	crossNamespaceClusterRoleBinding.Name = "rb-cross-namespace-cluster-role"
	crossNamespaceClusterRoleBinding.Namespace = roleNs.Namespace
	crossNamespaceClusterRoleBinding.Subjects[0].Namespace = sa1.Namespace
	var cr1 rbacv1.ClusterRole
	err = json.Unmarshal([]byte(crStr1), &cr1)
	if err != nil {
		t.Errorf("Failed to unmarshal cr1: %v", err)
	}
	var crb1 rbacv1.ClusterRoleBinding
	err = json.Unmarshal([]byte(crbStr1), &crb1)
	if err != nil {
		t.Errorf("Failed to unmarshal crb1: %v", err)
	}
	var crb2 rbacv1.ClusterRoleBinding
	err = json.Unmarshal([]byte(crbStr2), &crb2)
	if err != nil {
		t.Errorf("Failed to unmarshal crb2: %v", err)
	}
	var pod1 v1.Pod
	err = json.Unmarshal([]byte(podStr1), &pod1)
	if err != nil {
		t.Errorf("Failed to unmarshal pod1: %v", err)
	}
	var podNoNodeName v1.Pod
	err = json.Unmarshal([]byte(podStrWithoutNodeName), &podNoNodeName)
	if err != nil {
		t.Errorf("Failed to unmarshal podNoNodeName: %v", err)
	}
	var podNoSa v1.Pod
	err = json.Unmarshal([]byte(podStrWithoutSa), &podNoSa)
	if err != nil {
		t.Errorf("Failed to unmarshal podNoSa: %v", err)
	}
	nodeList := &v1.NodeList{
		Items: []v1.Node{
			{
				ObjectMeta: metav1.ObjectMeta{
					Name: "my-node",
					Labels: map[string]string{
						"node-role.kubernetes.io/edge": "",
					},
				},
			},
			{
				ObjectMeta: metav1.ObjectMeta{
					Name: "node-2",
					Labels: map[string]string{
						"node-role.kubernetes.io/edge": "",
					},
				},
			},
		},
	}
	var tests = []struct {
		name            string
		input           []client.Object
		rbacObj         client.Object
		obj             client.Object
		reconcileResult []controllerruntime.Request
		rbacResult      bool
		objResult       bool
	}{
		{
			name: "filter role or serviceaccount success",
			input: []client.Object{&policyv1alpha1.ServiceAccountAccess{
				ObjectMeta: metav1.ObjectMeta{Name: "sa1", Namespace: "my-namespace"},
				Spec: policyv1alpha1.AccessSpec{
					ServiceAccount: sa1,
					AccessRoleBinding: []policyv1alpha1.AccessRoleBinding{
						{RoleBinding: rb1}, {RoleBinding: rb2}},
					AccessClusterRoleBinding: []policyv1alpha1.AccessClusterRoleBinding{
						{ClusterRoleBinding: crb1}}},
			}, &crb1, &rb1, &rb2},
			rbacObj:    &role1,
			obj:        &sa1,
			rbacResult: true,
			objResult:  true,
			reconcileResult: []controllerruntime.Request{
				{
					NamespacedName: types.NamespacedName{
						Name:      "sa1",
						Namespace: "my-namespace",
					},
				},
			},
		},
		{
			name: "filter role or serviceaccount failed",
			input: []client.Object{&policyv1alpha1.ServiceAccountAccess{
				ObjectMeta: metav1.ObjectMeta{Name: "sa1", Namespace: "my-namespace"},
				Spec: policyv1alpha1.AccessSpec{
					ServiceAccount: sa1,
					AccessRoleBinding: []policyv1alpha1.AccessRoleBinding{
						{RoleBinding: rb1}},
					AccessClusterRoleBinding: []policyv1alpha1.AccessClusterRoleBinding{
						{ClusterRoleBinding: crb1}}},
			}, &crb1, &rb1},
			rbacObj:         &role2,
			obj:             &sa2,
			rbacResult:      false,
			objResult:       false,
			reconcileResult: []controllerruntime.Request{},
		},
		{
			name: "filter role failed with nil role",
			input: []client.Object{&policyv1alpha1.ServiceAccountAccess{
				ObjectMeta: metav1.ObjectMeta{Name: "sa1", Namespace: "my-namespace"},
				Spec: policyv1alpha1.AccessSpec{
					ServiceAccount: sa1,
					AccessClusterRoleBinding: []policyv1alpha1.AccessClusterRoleBinding{
						{ClusterRoleBinding: crb1}}},
			}, &crb1},
			rbacObj:         &role2,
			rbacResult:      false,
			reconcileResult: []controllerruntime.Request{},
		},
		{
			name: "filter role or serviceaccount failed with different namespace",
			input: []client.Object{&policyv1alpha1.ServiceAccountAccess{
				ObjectMeta: metav1.ObjectMeta{Name: "sa1", Namespace: "my-namespace"},
				Spec: policyv1alpha1.AccessSpec{
					ServiceAccount: sa1,
					AccessRoleBinding: []policyv1alpha1.AccessRoleBinding{
						{RoleBinding: rb1},
					},
					AccessClusterRoleBinding: []policyv1alpha1.AccessClusterRoleBinding{
						{ClusterRoleBinding: crb1}}},
			}, &crb1, &rb1},
			rbacObj:         &roleNs,
			obj:             &saNs,
			rbacResult:      false,
			objResult:       false,
			reconcileResult: []controllerruntime.Request{},
		},
		{
			name: "map role referenced by cross namespace rolebinding",
			input: []client.Object{&policyv1alpha1.ServiceAccountAccess{
				ObjectMeta: metav1.ObjectMeta{Name: "sa1", Namespace: "my-namespace"},
				Spec:       policyv1alpha1.AccessSpec{ServiceAccount: sa1},
			}, crossNamespaceRoleBinding},
			rbacObj:    &roleNs,
			rbacResult: true,
			reconcileResult: []controllerruntime.Request{
				{NamespacedName: types.NamespacedName{Name: "sa1", Namespace: "my-namespace"}},
			},
		},
		{
			name: "filter role failed with nil rolebinding and clusterrolebinding",
			input: []client.Object{&policyv1alpha1.ServiceAccountAccess{
				ObjectMeta: metav1.ObjectMeta{Name: "sa1", Namespace: "my-namespace"},
				Spec:       policyv1alpha1.AccessSpec{ServiceAccount: sa1},
			}},
			rbacObj:         &role2,
			rbacResult:      false,
			reconcileResult: []controllerruntime.Request{},
		},
		{
			name: "filter rolebinding success",
			input: []client.Object{&policyv1alpha1.ServiceAccountAccess{
				ObjectMeta: metav1.ObjectMeta{Name: "sa1", Namespace: "my-namespace"},
				Spec: policyv1alpha1.AccessSpec{
					ServiceAccount: sa1,
					AccessRoleBinding: []policyv1alpha1.AccessRoleBinding{
						{RoleBinding: rb1}},
					AccessClusterRoleBinding: []policyv1alpha1.AccessClusterRoleBinding{
						{ClusterRoleBinding: crb1}}},
			}, &crb1, &rb1},
			rbacObj:    &rb1,
			rbacResult: true,
			reconcileResult: []controllerruntime.Request{
				{NamespacedName: types.NamespacedName{Name: "sa1", Namespace: "my-namespace"}},
			},
		},
		{
			name: "filter rolebinding failed",
			input: []client.Object{&policyv1alpha1.ServiceAccountAccess{
				ObjectMeta: metav1.ObjectMeta{Name: "sa1", Namespace: "my-namespace"},
				Spec: policyv1alpha1.AccessSpec{
					ServiceAccount: sa1,
					AccessRoleBinding: []policyv1alpha1.AccessRoleBinding{
						{RoleBinding: rb1}},
					AccessClusterRoleBinding: []policyv1alpha1.AccessClusterRoleBinding{
						{ClusterRoleBinding: crb1}}},
			}, &crb1, &rb1},
			rbacObj:    &rb2,
			rbacResult: true,
			reconcileResult: []controllerruntime.Request{
				{NamespacedName: types.NamespacedName{Name: "sa1", Namespace: "my-namespace"}},
			},
		},
		{
			name: "filter rolebinding failed with nil rolebinding",
			input: []client.Object{&policyv1alpha1.ServiceAccountAccess{
				ObjectMeta: metav1.ObjectMeta{Name: "sa1", Namespace: "my-namespace"},
				Spec: policyv1alpha1.AccessSpec{
					ServiceAccount: sa1,
					AccessClusterRoleBinding: []policyv1alpha1.AccessClusterRoleBinding{
						{ClusterRoleBinding: crb1}}},
			}, &crb1},
			rbacObj:    &rb2,
			rbacResult: true,
			reconcileResult: []controllerruntime.Request{
				{NamespacedName: types.NamespacedName{Name: "sa1", Namespace: "my-namespace"}},
			},
		},
		{
			name: "filter clusterrolebinding success",
			input: []client.Object{&policyv1alpha1.ServiceAccountAccess{
				ObjectMeta: metav1.ObjectMeta{Name: "sa1", Namespace: "my-namespace"},
				Spec: policyv1alpha1.AccessSpec{
					ServiceAccount: sa1,
					AccessRoleBinding: []policyv1alpha1.AccessRoleBinding{
						{RoleBinding: rb1}},
					AccessClusterRoleBinding: []policyv1alpha1.AccessClusterRoleBinding{
						{ClusterRoleBinding: crb1}}},
			}, &crb1, &rb1},
			rbacObj:    &crb1,
			rbacResult: true,
			reconcileResult: []controllerruntime.Request{
				{
					NamespacedName: types.NamespacedName{
						Name:      "sa1",
						Namespace: "my-namespace",
					},
				},
			},
		},
		{
			name: "filter clusterrole success",
			input: []client.Object{&policyv1alpha1.ServiceAccountAccess{
				ObjectMeta: metav1.ObjectMeta{Name: "sa1", Namespace: "my-namespace"},
				Spec: policyv1alpha1.AccessSpec{
					ServiceAccount: sa1,
					AccessRoleBinding: []policyv1alpha1.AccessRoleBinding{
						{RoleBinding: rb1},
					},
					AccessClusterRoleBinding: []policyv1alpha1.AccessClusterRoleBinding{
						{ClusterRoleBinding: crb1}}},
			}, &crb1, &rb1},
			rbacObj:    &cr1,
			rbacResult: true,
			reconcileResult: []controllerruntime.Request{
				{
					NamespacedName: types.NamespacedName{
						Name:      "sa1",
						Namespace: "my-namespace",
					},
				},
			},
		},
		{
			name: "filter rolebinding bind cluster role success",
			input: []client.Object{&policyv1alpha1.ServiceAccountAccess{
				ObjectMeta: metav1.ObjectMeta{Name: "sa1", Namespace: "my-namespace"},
				Spec: policyv1alpha1.AccessSpec{
					ServiceAccount: sa1,
					AccessRoleBinding: []policyv1alpha1.AccessRoleBinding{
						{RoleBinding: rbWithCr}}},
			}, &rbWithCr},
			rbacObj:    &cr1,
			rbacResult: true,
			reconcileResult: []controllerruntime.Request{
				{
					NamespacedName: types.NamespacedName{
						Name:      "sa1",
						Namespace: "my-namespace",
					},
				},
			},
		},
		{
			name: "map clusterrole referenced by cross namespace rolebinding",
			input: []client.Object{&policyv1alpha1.ServiceAccountAccess{
				ObjectMeta: metav1.ObjectMeta{Name: "sa1", Namespace: "my-namespace"},
				Spec:       policyv1alpha1.AccessSpec{ServiceAccount: sa1},
			}, crossNamespaceClusterRoleBinding},
			rbacObj:    &cr1,
			rbacResult: true,
			reconcileResult: []controllerruntime.Request{
				{NamespacedName: types.NamespacedName{Name: "sa1", Namespace: "my-namespace"}},
			},
		},
		{
			name: "filter pod success",
			input: []client.Object{&policyv1alpha1.ServiceAccountAccess{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "sa1",
					Namespace: "my-namespace",
				},
				Spec: policyv1alpha1.AccessSpec{
					ServiceAccount: sa1,
				},
			}},
			obj:       &pod1,
			objResult: true,
		},
		{
			name: "filter pod failed for without service account",
			input: []client.Object{&policyv1alpha1.ServiceAccountAccess{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "sa1",
					Namespace: "my-namespace",
				},
				Spec: policyv1alpha1.AccessSpec{
					ServiceAccount: sa1,
				},
			}},
			obj:       &podNoSa,
			objResult: false,
		},
		{
			name: "filter pod failed for without node name",
			input: []client.Object{&policyv1alpha1.ServiceAccountAccess{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "sa1",
					Namespace: "my-namespace",
				},
				Spec: policyv1alpha1.AccessSpec{
					ServiceAccount: sa1,
				},
			}},
			obj:       &podNoNodeName,
			objResult: false,
		},
	}
	var accessScheme = runtime.NewScheme()
	if err := policyv1alpha1.AddToScheme(accessScheme); err != nil {
		t.Errorf("Failed to add access scheme: %v", err)
	}
	if err := v1.AddToScheme(accessScheme); err != nil {
		t.Errorf("Failed to add v1 scheme: %v", err)
	}
	if err := rbacv1.AddToScheme(accessScheme); err != nil {
		t.Errorf("Failed to add rbacv1 scheme: %v", err)
	}
	for _, tc := range tests {
		fakeClient := fake.NewClientBuilder().WithScheme(accessScheme).WithObjects(tc.input...).WithLists(nodeList).Build()
		ctr := &Controller{
			Client: fakeClient,
			Reader: fakeClient,
		}
		if tc.rbacObj != nil {
			got2 := ctr.mapRolesFunc(context.Background(), tc.rbacObj)
			if !equality.Semantic.DeepEqual(got2, tc.reconcileResult) {
				t.Errorf("case %q want=%v, got=%v", tc.name, tc.reconcileResult, got2)
			}
			if got := len(got2) > 0; got != tc.rbacResult {
				t.Errorf("case %q want match=%v, got match=%v", tc.name, tc.rbacResult, got)
			}
		}
		if tc.obj != nil {
			got1 := ctr.filterObject(context.Background(), tc.obj)
			if !reflect.DeepEqual(got1, tc.objResult) {
				t.Errorf("case %q want=%v, got=%v", tc.name, tc.objResult, got1)
			}
		}
	}
}

func TestRoleBindingEventsMapAffectedServiceAccount(t *testing.T) {
	serviceAccount := v1.ServiceAccount{
		ObjectMeta: metav1.ObjectMeta{Name: "old-sa", Namespace: "workloads"},
	}
	access := &policyv1alpha1.ServiceAccountAccess{
		ObjectMeta: metav1.ObjectMeta{Name: serviceAccount.Name, Namespace: serviceAccount.Namespace},
		Spec:       policyv1alpha1.AccessSpec{ServiceAccount: serviceAccount},
	}
	oldRoleBinding := &rbacv1.RoleBinding{
		ObjectMeta: metav1.ObjectMeta{Name: "reader", Namespace: "tenant-a"},
		Subjects: []rbacv1.Subject{{
			Kind:      rbacv1.ServiceAccountKind,
			Name:      serviceAccount.Name,
			Namespace: serviceAccount.Namespace,
		}},
		RoleRef: rbacv1.RoleRef{APIGroup: rbacv1.GroupName, Kind: "Role", Name: "reader"},
	}
	newRoleBinding := oldRoleBinding.DeepCopy()
	newRoleBinding.Subjects = nil

	testScheme := runtime.NewScheme()
	if err := policyv1alpha1.AddToScheme(testScheme); err != nil {
		t.Fatalf("failed to add policyv1alpha1 scheme: %v", err)
	}
	fakeClient := fake.NewClientBuilder().WithScheme(testScheme).WithObjects(access).Build()
	controller := &Controller{Client: fakeClient, Reader: fakeClient}
	eventHandler := handler.EnqueueRequestsFromMapFunc(controller.mapRolesFunc)
	queue := workqueue.NewTypedRateLimitingQueue(workqueue.DefaultTypedControllerRateLimiter[controllerruntime.Request]())
	defer queue.ShutDown()
	want := controllerruntime.Request{
		NamespacedName: types.NamespacedName{Name: access.Name, Namespace: access.Namespace},
	}
	assertQueued := func(eventName string) {
		t.Helper()
		if queue.Len() != 1 {
			t.Fatalf("rolebinding %s enqueued %d requests, want 1", eventName, queue.Len())
		}
		request, shutdown := queue.Get()
		if shutdown {
			t.Fatalf("workqueue shut down before rolebinding %s request was read", eventName)
		}
		queue.Done(request)
		if request != want {
			t.Errorf("rolebinding %s enqueued %#v, want %#v", eventName, request, want)
		}
	}

	eventHandler.Create(context.Background(), event.CreateEvent{Object: oldRoleBinding}, queue)
	assertQueued("create")
	eventHandler.Update(context.Background(), event.UpdateEvent{
		ObjectOld: oldRoleBinding,
		ObjectNew: newRoleBinding,
	}, queue)
	assertQueued("subject removal")
	eventHandler.Delete(context.Background(), event.DeleteEvent{Object: oldRoleBinding}, queue)
	assertQueued("delete")
}

func TestMapObjectFunc(t *testing.T) {
	var pod1 v1.Pod
	err := json.Unmarshal([]byte(podStr1), &pod1)
	if err != nil {
		t.Errorf("Failed to unmarshal pod1: %v", err)
	}
	var podDel v1.Pod
	err = json.Unmarshal([]byte(podDelStr1), &podDel)
	if err != nil {
		t.Errorf("Failed to unmarshal podDel: %v", err)
	}
	var rb2 rbacv1.RoleBinding
	err = json.Unmarshal([]byte(rbStr2), &rb2)
	if err != nil {
		t.Errorf("Failed to unmarshal rb2: %v", err)
	}
	var crb1 rbacv1.ClusterRoleBinding
	err = json.Unmarshal([]byte(crbStr1), &crb1)
	if err != nil {
		t.Errorf("Failed to unmarshal crb1: %v", err)
	}
	var sa1 v1.ServiceAccount
	err = json.Unmarshal([]byte(saStr1), &sa1)
	if err != nil {
		t.Errorf("Failed to unmarshal sa1: %v", err)
	}
	var sa2 v1.ServiceAccount
	err = json.Unmarshal([]byte(saStr2), &sa2)
	if err != nil {
		t.Errorf("Failed to unmarshal sa2: %v", err)
	}
	var cr1 rbacv1.ClusterRole
	err = json.Unmarshal([]byte(crStr1), &cr1)
	if err != nil {
		t.Errorf("Failed to unmarshal cr1: %v", err)
	}
	var rb1 rbacv1.RoleBinding
	err = json.Unmarshal([]byte(rbStr1), &rb1)
	if err != nil {
		t.Errorf("Failed to unmarshal rb1: %v", err)
	}
	var role1 rbacv1.Role
	err = json.Unmarshal([]byte(roleStr1), &role1)
	if err != nil {
		t.Errorf("Failed to unmarshal role1: %v", err)
	}
	var tests = []struct {
		name            string
		input           *policyv1alpha1.ServiceAccountAccess
		obj             client.Object
		reconcileResult []controllerruntime.Request
		output          *[]policyv1alpha1.ServiceAccountAccess
	}{
		{
			name: "match pod success and won't reconcile",
			input: &policyv1alpha1.ServiceAccountAccess{
				ObjectMeta: metav1.ObjectMeta{Name: "sa1", Namespace: "my-namespace"},
				Spec: policyv1alpha1.AccessSpec{
					ServiceAccount:           sa1,
					AccessRoleBinding:        []policyv1alpha1.AccessRoleBinding{{RoleBinding: rb1}, {RoleBinding: rb2}},
					AccessClusterRoleBinding: []policyv1alpha1.AccessClusterRoleBinding{{ClusterRoleBinding: crb1}},
				},
			},
			obj: &pod1,
			reconcileResult: []controllerruntime.Request{
				{NamespacedName: types.NamespacedName{Name: "sa1", Namespace: "my-namespace"}},
			},
			output: &[]policyv1alpha1.ServiceAccountAccess{{
				ObjectMeta: metav1.ObjectMeta{Name: "sa1", Namespace: "my-namespace"},
				Spec: policyv1alpha1.AccessSpec{
					ServiceAccount:           sa1,
					AccessRoleBinding:        []policyv1alpha1.AccessRoleBinding{{RoleBinding: rb1}, {RoleBinding: rb2}},
					AccessClusterRoleBinding: []policyv1alpha1.AccessClusterRoleBinding{{ClusterRoleBinding: crb1}},
				},
			}},
		},
		{
			name: "match deleting pod success and reconcile",
			input: &policyv1alpha1.ServiceAccountAccess{
				ObjectMeta: metav1.ObjectMeta{Name: "sa1", Namespace: "my-namespace"},
				Spec: policyv1alpha1.AccessSpec{
					ServiceAccount:           sa1,
					AccessRoleBinding:        []policyv1alpha1.AccessRoleBinding{{RoleBinding: rb1}, {RoleBinding: rb2}},
					AccessClusterRoleBinding: []policyv1alpha1.AccessClusterRoleBinding{{ClusterRoleBinding: crb1}},
				},
			},
			obj: &podDel,
			reconcileResult: []controllerruntime.Request{
				{NamespacedName: types.NamespacedName{Name: "sa1", Namespace: "my-namespace"}},
			},
			output: &[]policyv1alpha1.ServiceAccountAccess{{
				ObjectMeta: metav1.ObjectMeta{Name: "sa1", Namespace: "my-namespace"},
				Spec: policyv1alpha1.AccessSpec{
					ServiceAccount:           sa1,
					AccessRoleBinding:        []policyv1alpha1.AccessRoleBinding{{RoleBinding: rb1}, {RoleBinding: rb2}},
					AccessClusterRoleBinding: []policyv1alpha1.AccessClusterRoleBinding{{ClusterRoleBinding: crb1}},
				},
			}},
		},
		{
			name: "match pod not exist in access list",
			input: &policyv1alpha1.ServiceAccountAccess{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "sa2",
					Namespace: "my-namespace",
				},
				Spec: policyv1alpha1.AccessSpec{
					ServiceAccount: sa2,
					AccessRoleBinding: []policyv1alpha1.AccessRoleBinding{
						{
							RoleBinding: rb2,
						},
					},
				},
			},
			obj:             &pod1,
			reconcileResult: []controllerruntime.Request{},
			output: &[]policyv1alpha1.ServiceAccountAccess{
				{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "sa2",
						Namespace: "my-namespace",
					},
					Spec: policyv1alpha1.AccessSpec{
						ServiceAccount: sa2,
						AccessRoleBinding: []policyv1alpha1.AccessRoleBinding{
							{
								RoleBinding: rb2,
							},
						},
					},
				},
				{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "sa1",
						Namespace: "my-namespace",
					},
					Spec: policyv1alpha1.AccessSpec{
						ServiceAccount: v1.ServiceAccount{
							ObjectMeta: metav1.ObjectMeta{
								Name:      "sa1",
								Namespace: "my-namespace",
							},
						},
					},
				},
			},
		},
		{
			name: "match serviceaccount success",
			input: &policyv1alpha1.ServiceAccountAccess{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "sa1",
					Namespace: "my-namespace",
				},
				Spec: policyv1alpha1.AccessSpec{
					ServiceAccount: sa1,
					AccessRoleBinding: []policyv1alpha1.AccessRoleBinding{
						{
							RoleBinding: rb1,
						},
						{
							RoleBinding: rb2,
						},
					},
					AccessClusterRoleBinding: []policyv1alpha1.AccessClusterRoleBinding{
						{
							ClusterRoleBinding: crb1,
						},
					},
				},
			},
			obj: &sa1,
			reconcileResult: []controllerruntime.Request{
				{
					NamespacedName: types.NamespacedName{
						Name:      "sa1",
						Namespace: "my-namespace",
					},
				},
			},
			output: &[]policyv1alpha1.ServiceAccountAccess{{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "sa1",
					Namespace: "my-namespace",
				},
				Spec: policyv1alpha1.AccessSpec{
					ServiceAccount: sa1,
					AccessRoleBinding: []policyv1alpha1.AccessRoleBinding{
						{
							RoleBinding: rb1,
						},
						{
							RoleBinding: rb2,
						},
					},
					AccessClusterRoleBinding: []policyv1alpha1.AccessClusterRoleBinding{
						{
							ClusterRoleBinding: crb1,
						},
					},
				},
			}},
		},
	}
	var accessScheme = runtime.NewScheme()
	if err := policyv1alpha1.AddToScheme(accessScheme); err != nil {
		t.Errorf("Failed to add access scheme: %v", err)
	}
	for _, tc := range tests {
		fakeClient := fake.NewClientBuilder().WithScheme(accessScheme).WithObjects(tc.input).Build()
		ctr := &Controller{
			Client: fakeClient,
			Reader: fakeClient,
		}
		got := ctr.mapObjectFunc(context.Background(), tc.obj)
		if !equality.Semantic.DeepEqual(got, tc.reconcileResult) {
			t.Errorf("case %q, mapObjectFunc() = %v, want %v", tc.name, got, tc.reconcileResult)
		}
		sort.Slice(*tc.output, func(i, j int) bool {
			return (*tc.output)[i].Name < (*tc.output)[j].Name
		})
		accList := &policyv1alpha1.ServiceAccountAccessList{}
		if err := ctr.Client.List(context.Background(), accList, &client.ListOptions{Namespace: tc.obj.GetNamespace()}); err != nil {
			t.Errorf("Failed to list access: %v", err)
		}
		sort.Slice(accList.Items, func(i, j int) bool {
			return accList.Items[i].Name < accList.Items[j].Name
		})
		for i := range accList.Items {
			if accList.Items[i].Name != (*tc.output)[i].Name {
				t.Errorf("case %q, got %v, want %v", tc.name, accList.Items[i].Name, (*tc.output)[i].Name)
			}
			if accList.Items[i].Namespace != (*tc.output)[i].Namespace {
				t.Errorf("case %q, got %v, want %v", tc.name, accList.Items[i].Namespace, (*tc.output)[i].Namespace)
			}
			if !equality.Semantic.DeepEqual(accList.Items[i].Spec, (*tc.output)[i].Spec) {
				t.Errorf("case %q, got %v, want %v", tc.name, accList.Items[i].Spec, (*tc.output)[i].Spec)
			}
		}
	}
}

func TestGetNodeListOfServiceAccountAccess(t *testing.T) {
	// Create a sample ServiceAccountAccess object
	saa := &policyv1alpha1.ServiceAccountAccess{
		Spec: policyv1alpha1.AccessSpec{
			ServiceAccount: v1.ServiceAccount{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "test-sa",
					Namespace: "test-ns",
				},
			},
		},
	}

	nodeList := &v1.NodeList{
		Items: []v1.Node{
			{
				ObjectMeta: metav1.ObjectMeta{
					Name: "node-1",
					Labels: map[string]string{
						"node-role.kubernetes.io/edge": "",
					},
				},
			},
			{
				ObjectMeta: metav1.ObjectMeta{
					Name: "node-2",
					Labels: map[string]string{
						"node-role.kubernetes.io/edge": "",
					},
				},
			},
		},
	}

	// Create a sample PodList object with two pods on different nodes
	podList := &v1.PodList{
		Items: []v1.Pod{
			{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "pod-1",
					Namespace: "test-ns",
				},
				Spec: v1.PodSpec{
					NodeName:           "node-1",
					ServiceAccountName: "test-sa",
				},
			},
			{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "pod-2",
					Namespace: "test-ns",
				},
				Spec: v1.PodSpec{
					NodeName:           "node-2",
					ServiceAccountName: "test-sa",
				},
			},
			{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "pod-3",
					Namespace: "test-ns",
				},
				Spec: v1.PodSpec{
					NodeName:           "node-2",
					ServiceAccountName: "test-sa",
				},
			},
			{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "pod-4",
					Namespace: "test-ns",
				},
				Spec: v1.PodSpec{
					NodeName:           "node-4",
					ServiceAccountName: "test-sa2",
				},
			},
		},
	}
	pdStrategyTypeIndexer := func(obj client.Object) []string {
		pd, ok := obj.(*v1.Pod)
		if !ok {
			panic(fmt.Errorf("indexer function for type %T's spec.strategy.type field received"+
				" object of type %T, this should never happen", v1.Pod{}, obj))
		}
		serviceAccountName := ""
		if pd != nil {
			serviceAccountName = pd.Spec.ServiceAccountName
		}
		return []string{serviceAccountName}
	}
	var v1Scheme = runtime.NewScheme()
	if err := v1.AddToScheme(v1Scheme); err != nil {
		t.Errorf("Failed to add access scheme: %v", err)
	}
	withScheme := fake.NewClientBuilder().WithScheme(v1Scheme).WithIndex(&v1.Pod{}, "spec.serviceAccountName", pdStrategyTypeIndexer)
	fakeClient := withScheme.Build()
	got, err := getNodeListOfServiceAccountAccess(context.Background(), fakeClient, saa)
	if err != nil {
		t.Errorf("fakeClient get node list error = %v", err)
	}
	if !equality.Semantic.DeepEqual(got, []string{}) {
		t.Errorf("testcase 1 got %v, want %v", got, []string{})
	}
	fakeClient2 := withScheme.WithObjects(&podList.Items[0]).WithLists(nodeList).Build()
	got2, err := getNodeListOfServiceAccountAccess(context.Background(), fakeClient2, saa)
	if err != nil {
		t.Errorf("fakeClient2 get node list error = %v", err)
	}
	if !equality.Semantic.DeepEqual(got2, []string{"node-1"}) {
		t.Errorf("testcase 2 got %v, want %v", got2, []string{"node-1"})
	}
	fakeClient3 := withScheme.WithObjects(&podList.Items[1]).WithObjects(&podList.Items[2]).Build()
	got3, err := getNodeListOfServiceAccountAccess(context.Background(), fakeClient3, saa)
	if err != nil {
		t.Errorf("fakeClient3 get node list error = %v", err)
	}
	if !equality.Semantic.DeepEqual(got3, []string{"node-1", "node-2"}) {
		t.Errorf("testcase 3 got %v, want %v", got3, []string{"node-1", "node-2"})
	}
}

func TestVisitRulesForCollectsCrossNamespaceRoleBindings(t *testing.T) {
	const (
		serviceAccountNamespace = "kube-system"
		serviceAccountName      = "edge-client"
	)

	roleRules := []rbacv1.PolicyRule{{
		APIGroups: []string{""},
		Resources: []string{"configmaps"},
		Verbs:     []string{"get"},
	}}
	clusterRoleRules := []rbacv1.PolicyRule{{
		APIGroups: []string{""},
		Resources: []string{"pods"},
		Verbs:     []string{"list"},
	}}
	role := &rbacv1.Role{
		ObjectMeta: metav1.ObjectMeta{Name: "configmap-reader", Namespace: "tenant-a"},
		Rules:      roleRules,
	}
	clusterRole := &rbacv1.ClusterRole{
		ObjectMeta: metav1.ObjectMeta{Name: "pod-reader"},
		Rules:      clusterRoleRules,
	}
	serviceAccountSubject := rbacv1.Subject{
		Kind:      rbacv1.ServiceAccountKind,
		Name:      serviceAccountName,
		Namespace: serviceAccountNamespace,
	}
	roleBinding := &rbacv1.RoleBinding{
		ObjectMeta: metav1.ObjectMeta{Name: "allow-edge-client", Namespace: "tenant-a"},
		Subjects:   []rbacv1.Subject{serviceAccountSubject},
		RoleRef: rbacv1.RoleRef{
			APIGroup: rbacv1.GroupName,
			Kind:     "Role",
			Name:     role.Name,
		},
	}
	roleBindingToClusterRole := &rbacv1.RoleBinding{
		ObjectMeta: metav1.ObjectMeta{Name: "allow-edge-client", Namespace: "tenant-b"},
		Subjects:   []rbacv1.Subject{serviceAccountSubject},
		RoleRef: rbacv1.RoleRef{
			APIGroup: rbacv1.GroupName,
			Kind:     "ClusterRole",
			Name:     clusterRole.Name,
		},
	}
	brokenRoleBinding := &rbacv1.RoleBinding{
		ObjectMeta: metav1.ObjectMeta{Name: "a-broken", Namespace: "tenant-c"},
		Subjects:   []rbacv1.Subject{serviceAccountSubject},
		RoleRef: rbacv1.RoleRef{
			APIGroup: rbacv1.GroupName,
			Kind:     "Role",
			Name:     "missing",
		},
	}
	nonMatchingRoleBinding := roleBinding.DeepCopy()
	nonMatchingRoleBinding.Name = "other-service-account"
	nonMatchingRoleBinding.Subjects[0].Namespace = "tenant-a"
	clusterRoleBinding := &rbacv1.ClusterRoleBinding{
		ObjectMeta: metav1.ObjectMeta{Name: "allow-edge-client-cluster-wide"},
		Subjects:   []rbacv1.Subject{serviceAccountSubject},
		RoleRef: rbacv1.RoleRef{
			APIGroup: rbacv1.GroupName,
			Kind:     "ClusterRole",
			Name:     clusterRole.Name,
		},
	}

	testScheme := runtime.NewScheme()
	if err := rbacv1.AddToScheme(testScheme); err != nil {
		t.Fatalf("failed to add rbacv1 scheme: %v", err)
	}
	fakeClient := fake.NewClientBuilder().WithScheme(testScheme).WithObjects(
		role,
		clusterRole,
		brokenRoleBinding,
		roleBinding,
		roleBindingToClusterRole,
		nonMatchingRoleBinding,
		clusterRoleBinding,
	).Build()
	controller := &Controller{Client: fakeClient, Reader: fakeClient}
	userInfo := &user.DefaultInfo{
		Name: "system:serviceaccount:" + serviceAccountNamespace + ":" + serviceAccountName,
		UID:  "edge-client-uid",
		Groups: []string{
			"system:serviceaccounts",
			"system:serviceaccounts:" + serviceAccountNamespace,
			user.AllAuthenticated,
		},
	}
	access := &policyv1alpha1.ServiceAccountAccess{}

	controller.VisitRulesFor(context.Background(), userInfo, serviceAccountNamespace, access)

	wantRoleBindings := map[string][]rbacv1.PolicyRule{
		"tenant-a/allow-edge-client": roleRules,
		"tenant-b/allow-edge-client": clusterRoleRules,
	}
	if len(access.Spec.AccessRoleBinding) != len(wantRoleBindings) {
		t.Fatalf("VisitRulesFor() collected %d rolebindings, want %d: %#v", len(access.Spec.AccessRoleBinding), len(wantRoleBindings), access.Spec.AccessRoleBinding)
	}
	for _, binding := range access.Spec.AccessRoleBinding {
		key := binding.RoleBinding.Namespace + "/" + binding.RoleBinding.Name
		wantRules, ok := wantRoleBindings[key]
		if !ok {
			t.Errorf("VisitRulesFor() collected unexpected rolebinding %s", key)
			continue
		}
		if !equality.Semantic.DeepEqual(binding.Rules, wantRules) {
			t.Errorf("VisitRulesFor() rules for %s = %#v, want %#v", key, binding.Rules, wantRules)
		}
	}
	if len(access.Spec.AccessClusterRoleBinding) != 1 {
		t.Fatalf("VisitRulesFor() collected %d clusterrolebindings, want 1", len(access.Spec.AccessClusterRoleBinding))
	}
	if !equality.Semantic.DeepEqual(access.Spec.AccessClusterRoleBinding[0].Rules, clusterRoleRules) {
		t.Errorf("VisitRulesFor() clusterrolebinding rules = %#v, want %#v", access.Spec.AccessClusterRoleBinding[0].Rules, clusterRoleRules)
	}

	if err := fakeClient.Delete(context.Background(), roleBinding); err != nil {
		t.Fatalf("failed to delete cross namespace rolebinding: %v", err)
	}
	afterDelete := &policyv1alpha1.ServiceAccountAccess{}
	if err := controller.visitRulesFor(context.Background(), userInfo, afterDelete); err != nil {
		t.Fatalf("visitRulesFor() after rolebinding deletion returned error: %v", err)
	}
	if len(afterDelete.Spec.AccessRoleBinding) != 1 {
		t.Fatalf("visitRulesFor() after deletion collected %d rolebindings, want 1: %#v", len(afterDelete.Spec.AccessRoleBinding), afterDelete.Spec.AccessRoleBinding)
	}
	remaining := afterDelete.Spec.AccessRoleBinding[0].RoleBinding
	if remaining.Namespace != "tenant-b" || remaining.Name != "allow-edge-client" {
		t.Errorf("visitRulesFor() after deletion retained %s/%s, want tenant-b/allow-edge-client", remaining.Namespace, remaining.Name)
	}
}

type roleBindingListErrorClient struct {
	client.Client
	err error
}

func (c roleBindingListErrorClient) List(ctx context.Context, list client.ObjectList, opts ...client.ListOption) error {
	if _, ok := list.(*rbacv1.RoleBindingList); ok {
		return c.err
	}
	return c.Client.List(ctx, list, opts...)
}

func TestSyncRulesRequeuesWhenRoleBindingListFails(t *testing.T) {
	testScheme := runtime.NewScheme()
	if err := v1.AddToScheme(testScheme); err != nil {
		t.Fatalf("failed to add core v1 scheme: %v", err)
	}
	if err := rbacv1.AddToScheme(testScheme); err != nil {
		t.Fatalf("failed to add rbacv1 scheme: %v", err)
	}
	serviceAccount := &v1.ServiceAccount{
		ObjectMeta: metav1.ObjectMeta{Name: "edge-client", Namespace: "kube-system", UID: "edge-client-uid"},
	}
	fakeClient := fake.NewClientBuilder().WithScheme(testScheme).WithObjects(serviceAccount).Build()
	listErr := errors.New("rolebinding list failed")
	controller := &Controller{
		Client: roleBindingListErrorClient{Client: fakeClient, err: listErr},
		Reader: fakeClient,
	}
	access := &policyv1alpha1.ServiceAccountAccess{
		ObjectMeta: metav1.ObjectMeta{Name: serviceAccount.Name, Namespace: serviceAccount.Namespace},
		Spec:       policyv1alpha1.AccessSpec{ServiceAccount: *serviceAccount.DeepCopy()},
	}

	result, err := controller.syncRules(context.Background(), access)
	if !errors.Is(err, listErr) {
		t.Fatalf("syncRules() error = %v, want %v", err, listErr)
	}
	if !result.Requeue {
		t.Fatal("syncRules() did not request a requeue after rolebinding list failure")
	}
}

func TestSyncRules(t *testing.T) {
	var pod1 v1.Pod
	err := json.Unmarshal([]byte(podStr1), &pod1)
	if err != nil {
		t.Errorf("Failed to unmarshal pod1: %v", err)
	}
	var pod2 v1.Pod
	err = json.Unmarshal([]byte(podStr2), &pod2)
	if err != nil {
		t.Errorf("Failed to unmarshal pod2: %v", err)
	}
	var podWithoutNodeName v1.Pod
	err = json.Unmarshal([]byte(podStrWithoutNodeName), &podWithoutNodeName)
	if err != nil {
		t.Errorf("Failed to unmarshal podWithoutNodeName: %v", err)
	}
	var rb2 rbacv1.RoleBinding
	err = json.Unmarshal([]byte(rbStr2), &rb2)
	if err != nil {
		t.Errorf("Failed to unmarshal rb2: %v", err)
	}
	var crb1 rbacv1.ClusterRoleBinding
	err = json.Unmarshal([]byte(crbStr1), &crb1)
	if err != nil {
		t.Errorf("Failed to unmarshal crb1: %v", err)
	}
	var sa1 v1.ServiceAccount
	err = json.Unmarshal([]byte(saStr1), &sa1)
	if err != nil {
		t.Errorf("Failed to unmarshal sa1: %v", err)
	}
	var sa2 v1.ServiceAccount
	err = json.Unmarshal([]byte(saStr2), &sa2)
	if err != nil {
		t.Errorf("Failed to unmarshal sa2: %v", err)
	}
	var cr1 rbacv1.ClusterRole
	err = json.Unmarshal([]byte(crStr1), &cr1)
	if err != nil {
		t.Errorf("Failed to unmarshal cr1: %v", err)
	}
	var rb1 rbacv1.RoleBinding
	err = json.Unmarshal([]byte(rbStr1), &rb1)
	if err != nil {
		t.Errorf("Failed to unmarshal rb1: %v", err)
	}
	var role1 rbacv1.Role
	err = json.Unmarshal([]byte(roleStr1), &role1)
	if err != nil {
		t.Errorf("Failed to unmarshal role1: %v", err)
	}
	var role2 rbacv1.Role
	err = json.Unmarshal([]byte(roleStr2), &role2)
	if err != nil {
		t.Errorf("Failed to unmarshal role1: %v", err)
	}
	nodeList := &v1.NodeList{
		Items: []v1.Node{
			{
				ObjectMeta: metav1.ObjectMeta{
					Name: "my-node",
					Labels: map[string]string{
						"node-role.kubernetes.io/edge": "",
					},
				},
			},
			{
				ObjectMeta: metav1.ObjectMeta{
					Name: "my-node-2",
					Labels: map[string]string{
						"node-role.kubernetes.io/edge": "",
					},
				},
			},
			{
				ObjectMeta: metav1.ObjectMeta{
					Name: "my-node-3",
					Labels: map[string]string{
						"node-role.kubernetes.io/edge": "",
					},
				},
			},
			{
				ObjectMeta: metav1.ObjectMeta{
					Name: "my-node-4",
				},
			},
		},
	}
	var nodeStatus1 = policyv1alpha1.AccessStatus{NodeList: []string{"my-node"}}
	var nodeStatus2 = policyv1alpha1.AccessStatus{NodeList: []string{"my-node", "my-node-2"}}
	var nodeStatus3 = policyv1alpha1.AccessStatus{NodeList: []string{"my-node-2"}}
	var nodeStatus4 = policyv1alpha1.AccessStatus{NodeList: []string{"my-node-2", "my-node-3"}}
	var saa1 = policyv1alpha1.ServiceAccountAccess{
		ObjectMeta: metav1.ObjectMeta{Name: "sa1", Namespace: "my-namespace"},
		Spec: policyv1alpha1.AccessSpec{
			ServiceAccount:           sa1,
			AccessRoleBinding:        []policyv1alpha1.AccessRoleBinding{{RoleBinding: rb1, Rules: role1.Rules}},
			AccessClusterRoleBinding: []policyv1alpha1.AccessClusterRoleBinding{{ClusterRoleBinding: crb1, Rules: cr1.Rules}},
		},
		Status: nodeStatus1,
	}
	var saa2 = policyv1alpha1.ServiceAccountAccess{
		ObjectMeta: metav1.ObjectMeta{Name: "sa1", Namespace: "my-namespace"},
		Spec: policyv1alpha1.AccessSpec{
			ServiceAccount:           sa1,
			AccessRoleBinding:        []policyv1alpha1.AccessRoleBinding{{RoleBinding: rb1, Rules: role1.Rules}},
			AccessClusterRoleBinding: []policyv1alpha1.AccessClusterRoleBinding{{ClusterRoleBinding: crb1, Rules: cr1.Rules}},
		},
		Status: nodeStatus2,
	}
	var saa3 = policyv1alpha1.ServiceAccountAccess{
		ObjectMeta: metav1.ObjectMeta{Name: "sa1", Namespace: "my-namespace"},
		Spec: policyv1alpha1.AccessSpec{
			ServiceAccount:           sa1,
			AccessRoleBinding:        []policyv1alpha1.AccessRoleBinding{{RoleBinding: rb1, Rules: role1.Rules}},
			AccessClusterRoleBinding: []policyv1alpha1.AccessClusterRoleBinding{{ClusterRoleBinding: crb1, Rules: cr1.Rules}},
		},
		Status: nodeStatus3,
	}
	var saa4 = policyv1alpha1.ServiceAccountAccess{
		ObjectMeta: metav1.ObjectMeta{Name: "sa1", Namespace: "my-namespace"},
		Spec: policyv1alpha1.AccessSpec{
			ServiceAccount:           sa1,
			AccessRoleBinding:        []policyv1alpha1.AccessRoleBinding{{RoleBinding: rb1, Rules: role1.Rules}},
			AccessClusterRoleBinding: []policyv1alpha1.AccessClusterRoleBinding{{ClusterRoleBinding: crb1, Rules: cr1.Rules}},
		},
		Status: nodeStatus4,
	}
	var saa5 = policyv1alpha1.ServiceAccountAccess{
		ObjectMeta: metav1.ObjectMeta{Name: "sa1", Namespace: "my-namespace"},
		Spec: policyv1alpha1.AccessSpec{
			ServiceAccount:           sa1,
			AccessRoleBinding:        []policyv1alpha1.AccessRoleBinding{{RoleBinding: rb1, Rules: role1.Rules}},
			AccessClusterRoleBinding: []policyv1alpha1.AccessClusterRoleBinding{{ClusterRoleBinding: crb1, Rules: cr1.Rules}},
		},
		Status: nodeStatus1,
	}
	var saaDiffName = policyv1alpha1.ServiceAccountAccess{
		ObjectMeta: metav1.ObjectMeta{Name: "sa2", Namespace: "my-namespace"},
		Spec: policyv1alpha1.AccessSpec{
			ServiceAccount: sa2,
		},
		Status: nodeStatus2,
	}
	var saaDeletion = policyv1alpha1.ServiceAccountAccess{
		ObjectMeta: metav1.ObjectMeta{Name: "sa1", Namespace: "my-namespace", DeletionTimestamp: &metav1.Time{Time: time.Now()}, Finalizers: []string{"test"}},
		Spec: policyv1alpha1.AccessSpec{
			ServiceAccount:           sa1,
			AccessRoleBinding:        []policyv1alpha1.AccessRoleBinding{{RoleBinding: rb1, Rules: role1.Rules}},
			AccessClusterRoleBinding: []policyv1alpha1.AccessClusterRoleBinding{{ClusterRoleBinding: crb1, Rules: cr1.Rules}},
		},
		Status: nodeStatus1,
	}
	var tests = []struct {
		name            string
		input           *policyv1alpha1.ServiceAccountAccess
		obj             []client.Object
		reconcileResult controllerruntime.Result
		output          *policyv1alpha1.ServiceAccountAccess
		msgOpr          []string
	}{
		{
			name:  "rolebinding updated only",
			input: saa1.DeepCopy(),
			obj: []client.Object{saa1.DeepCopy(), pod1.DeepCopy(), sa1.DeepCopy(), rb1.DeepCopy(), crb1.DeepCopy(),
				cr1.DeepCopy(), role1.DeepCopy(), rb2.DeepCopy(), role2.DeepCopy()},
			reconcileResult: controllerruntime.Result{},
			output: &policyv1alpha1.ServiceAccountAccess{ObjectMeta: metav1.ObjectMeta{Name: "sa1", Namespace: "my-namespace"},
				Spec: policyv1alpha1.AccessSpec{
					ServiceAccount:           *sa1.DeepCopy(),
					AccessRoleBinding:        []policyv1alpha1.AccessRoleBinding{{RoleBinding: *rb1.DeepCopy(), Rules: role1.Rules}, {RoleBinding: *rb2.DeepCopy(), Rules: role2.Rules}},
					AccessClusterRoleBinding: []policyv1alpha1.AccessClusterRoleBinding{{ClusterRoleBinding: *crb1.DeepCopy(), Rules: cr1.Rules}},
				},
				Status: policyv1alpha1.AccessStatus{NodeList: []string{"my-node"}},
			},
			msgOpr: []string{model.UpdateOperation},
		},
		{
			name:  "rolebinding updated and inserted new node",
			input: saa1.DeepCopy(),
			obj: []client.Object{saa1.DeepCopy(), pod1.DeepCopy(), pod2.DeepCopy(), sa1.DeepCopy(), rb1.DeepCopy(),
				crb1.DeepCopy(), cr1.DeepCopy(), role1.DeepCopy(), rb2.DeepCopy(), role2.DeepCopy()},
			reconcileResult: controllerruntime.Result{},
			output: &policyv1alpha1.ServiceAccountAccess{ObjectMeta: metav1.ObjectMeta{Name: "sa1", Namespace: "my-namespace"},
				Spec: policyv1alpha1.AccessSpec{
					ServiceAccount:           *sa1.DeepCopy(),
					AccessRoleBinding:        []policyv1alpha1.AccessRoleBinding{{RoleBinding: *rb1.DeepCopy(), Rules: role1.Rules}, {RoleBinding: *rb2.DeepCopy(), Rules: role2.Rules}},
					AccessClusterRoleBinding: []policyv1alpha1.AccessClusterRoleBinding{{ClusterRoleBinding: *crb1.DeepCopy(), Rules: cr1.Rules}},
				},
				Status: policyv1alpha1.AccessStatus{NodeList: []string{"my-node", "my-node-2"}},
			},
			msgOpr: []string{model.UpdateOperation, model.UpdateOperation},
		},
		{
			name:  "rolebinding updated and inserted/deleted new node",
			input: saa3.DeepCopy(),
			obj: []client.Object{saa3.DeepCopy(), pod1.DeepCopy(), sa1.DeepCopy(), rb1.DeepCopy(), crb1.DeepCopy(),
				cr1.DeepCopy(), role1.DeepCopy(), rb2.DeepCopy(), role2.DeepCopy()},
			reconcileResult: controllerruntime.Result{},
			output: &policyv1alpha1.ServiceAccountAccess{ObjectMeta: metav1.ObjectMeta{Name: "sa1", Namespace: "my-namespace"},
				Spec: policyv1alpha1.AccessSpec{
					ServiceAccount:           *sa1.DeepCopy(),
					AccessRoleBinding:        []policyv1alpha1.AccessRoleBinding{{RoleBinding: *rb1.DeepCopy(), Rules: role1.Rules}, {RoleBinding: *rb2.DeepCopy(), Rules: role2.Rules}},
					AccessClusterRoleBinding: []policyv1alpha1.AccessClusterRoleBinding{{ClusterRoleBinding: *crb1.DeepCopy(), Rules: cr1.Rules}},
				},
				Status: policyv1alpha1.AccessStatus{NodeList: []string{"my-node"}},
			},
			msgOpr: []string{model.DeleteOperation, model.UpdateOperation},
		},
		{
			name:  "rolebinding updated and inserted/deleted/updated new node",
			input: saa4.DeepCopy(),
			obj: []client.Object{saa4.DeepCopy(), pod1.DeepCopy(), pod2.DeepCopy(), sa1.DeepCopy(), rb1.DeepCopy(),
				crb1.DeepCopy(), cr1.DeepCopy(), role1.DeepCopy(), rb2.DeepCopy(), role2.DeepCopy()},
			reconcileResult: controllerruntime.Result{},
			output: &policyv1alpha1.ServiceAccountAccess{ObjectMeta: metav1.ObjectMeta{Name: "sa1", Namespace: "my-namespace"},
				Spec: policyv1alpha1.AccessSpec{
					ServiceAccount:           *sa1.DeepCopy(),
					AccessRoleBinding:        []policyv1alpha1.AccessRoleBinding{{RoleBinding: *rb1.DeepCopy(), Rules: role1.Rules}, {RoleBinding: *rb2.DeepCopy(), Rules: role2.Rules}},
					AccessClusterRoleBinding: []policyv1alpha1.AccessClusterRoleBinding{{ClusterRoleBinding: *crb1.DeepCopy(), Rules: cr1.Rules}},
				},
				Status: policyv1alpha1.AccessStatus{NodeList: []string{"my-node", "my-node-2"}},
			},
			msgOpr: []string{model.DeleteOperation, model.UpdateOperation},
		},
		{
			name:            "rolebinding updated and inserted new node with none old node",
			input:           saa5.DeepCopy(),
			obj:             []client.Object{saa5.DeepCopy(), &pod1, &pod2, &sa1, &rb1, &crb1, &cr1, &role1, &rb2, &role2},
			reconcileResult: controllerruntime.Result{},
			output: &policyv1alpha1.ServiceAccountAccess{ObjectMeta: metav1.ObjectMeta{Name: "sa1", Namespace: "my-namespace"},
				Spec: policyv1alpha1.AccessSpec{
					ServiceAccount:           *sa1.DeepCopy(),
					AccessRoleBinding:        []policyv1alpha1.AccessRoleBinding{{RoleBinding: *rb1.DeepCopy(), Rules: role1.Rules}, {RoleBinding: *rb2.DeepCopy(), Rules: role2.Rules}},
					AccessClusterRoleBinding: []policyv1alpha1.AccessClusterRoleBinding{{ClusterRoleBinding: *crb1.DeepCopy(), Rules: cr1.Rules}},
				},
				Status: policyv1alpha1.AccessStatus{NodeList: []string{"my-node", "my-node-2"}},
			},
			msgOpr: []string{model.UpdateOperation, model.UpdateOperation},
		},
		{
			name:  "rolebinding updated and deleted old node only",
			input: saa4.DeepCopy(),
			obj: []client.Object{saa4.DeepCopy(), podWithoutNodeName.DeepCopy(), sa1.DeepCopy(), rb1.DeepCopy(),
				crb1.DeepCopy(), cr1.DeepCopy(), role1.DeepCopy(), rb2.DeepCopy(), role2.DeepCopy()},
			reconcileResult: controllerruntime.Result{},
			output: &policyv1alpha1.ServiceAccountAccess{ObjectMeta: metav1.ObjectMeta{Name: "sa1", Namespace: "my-namespace"},
				Spec: policyv1alpha1.AccessSpec{
					ServiceAccount:           *sa1.DeepCopy(),
					AccessRoleBinding:        []policyv1alpha1.AccessRoleBinding{{RoleBinding: *rb1.DeepCopy(), Rules: role1.Rules}, {RoleBinding: *rb2.DeepCopy(), Rules: role2.Rules}},
					AccessClusterRoleBinding: []policyv1alpha1.AccessClusterRoleBinding{{ClusterRoleBinding: *crb1.DeepCopy(), Rules: cr1.Rules}},
				},
				Status: policyv1alpha1.AccessStatus{NodeList: []string{}},
			},
			msgOpr: []string{model.DeleteOperation},
		},
		{
			name:  "rolebinding updated and updated/deleted node",
			input: saa2.DeepCopy(),
			obj: []client.Object{saa2.DeepCopy(), pod1.DeepCopy(), sa1.DeepCopy(), rb1.DeepCopy(),
				crb1.DeepCopy(), cr1.DeepCopy(), role1.DeepCopy(), rb2.DeepCopy(), role2.DeepCopy()},
			reconcileResult: controllerruntime.Result{},
			output: &policyv1alpha1.ServiceAccountAccess{ObjectMeta: metav1.ObjectMeta{Name: "sa1", Namespace: "my-namespace"},
				Spec: policyv1alpha1.AccessSpec{
					ServiceAccount:           *sa1.DeepCopy(),
					AccessRoleBinding:        []policyv1alpha1.AccessRoleBinding{{RoleBinding: *rb1.DeepCopy(), Rules: role1.Rules}, {RoleBinding: *rb2.DeepCopy(), Rules: role2.Rules}},
					AccessClusterRoleBinding: []policyv1alpha1.AccessClusterRoleBinding{{ClusterRoleBinding: *crb1.DeepCopy(), Rules: cr1.Rules}},
				},
				Status: policyv1alpha1.AccessStatus{NodeList: []string{"my-node"}},
			},
			msgOpr: []string{model.DeleteOperation, model.UpdateOperation},
		},
		{
			name:  "rolebinding updated and none nodes",
			input: saa5.DeepCopy(),
			obj: []client.Object{saa5.DeepCopy(), podWithoutNodeName.DeepCopy(), sa1.DeepCopy(), rb1.DeepCopy(),
				crb1.DeepCopy(), cr1.DeepCopy(), role1.DeepCopy(), rb2.DeepCopy(), role2.DeepCopy()},
			reconcileResult: controllerruntime.Result{},
			output: &policyv1alpha1.ServiceAccountAccess{ObjectMeta: metav1.ObjectMeta{Name: "sa1", Namespace: "my-namespace"},
				Spec: policyv1alpha1.AccessSpec{
					ServiceAccount:           *sa1.DeepCopy(),
					AccessRoleBinding:        []policyv1alpha1.AccessRoleBinding{{RoleBinding: *rb1.DeepCopy(), Rules: role1.Rules}, {RoleBinding: *rb2.DeepCopy(), Rules: role2.Rules}},
					AccessClusterRoleBinding: []policyv1alpha1.AccessClusterRoleBinding{{ClusterRoleBinding: *crb1.DeepCopy(), Rules: cr1.Rules}},
				},
				Status: policyv1alpha1.AccessStatus{NodeList: []string{}},
			},
			msgOpr: []string{},
		},
		{
			name:  "service account not found",
			input: saa5.DeepCopy(),
			obj: []client.Object{saa5.DeepCopy(), podWithoutNodeName.DeepCopy(), sa2.DeepCopy(), rb1.DeepCopy(),
				crb1.DeepCopy(), cr1.DeepCopy(), role1.DeepCopy(), rb2.DeepCopy(), role2.DeepCopy()},
			reconcileResult: controllerruntime.Result{},
			output: &policyv1alpha1.ServiceAccountAccess{
				Status: policyv1alpha1.AccessStatus{NodeList: []string{"my-node"}},
			},
			msgOpr: []string{model.DeleteOperation},
		},
		{
			name:  "insert only",
			input: saa1.DeepCopy(),
			obj: []client.Object{saa1.DeepCopy(), pod1.DeepCopy(), pod2.DeepCopy(), sa1.DeepCopy(), rb1.DeepCopy(),
				crb1.DeepCopy(), cr1.DeepCopy(), role1.DeepCopy()},
			reconcileResult: controllerruntime.Result{},
			output: &policyv1alpha1.ServiceAccountAccess{ObjectMeta: metav1.ObjectMeta{Name: "sa1", Namespace: "my-namespace"},
				Spec: policyv1alpha1.AccessSpec{
					ServiceAccount:           *sa1.DeepCopy(),
					AccessRoleBinding:        []policyv1alpha1.AccessRoleBinding{{RoleBinding: *rb1.DeepCopy(), Rules: role1.Rules}},
					AccessClusterRoleBinding: []policyv1alpha1.AccessClusterRoleBinding{{ClusterRoleBinding: *crb1.DeepCopy(), Rules: cr1.Rules}},
				},
				Status: policyv1alpha1.AccessStatus{NodeList: []string{"my-node", "my-node-2"}},
			},
			msgOpr: []string{model.InsertOperation},
		},
		{
			name:  "delete only updates status",
			input: saa2.DeepCopy(),
			obj: []client.Object{saa2.DeepCopy(), pod1.DeepCopy(), sa1.DeepCopy(), rb1.DeepCopy(),
				crb1.DeepCopy(), cr1.DeepCopy(), role1.DeepCopy()},
			reconcileResult: controllerruntime.Result{},
			output: &policyv1alpha1.ServiceAccountAccess{ObjectMeta: metav1.ObjectMeta{Name: "sa1", Namespace: "my-namespace"},
				Spec: policyv1alpha1.AccessSpec{
					ServiceAccount:           *sa1.DeepCopy(),
					AccessRoleBinding:        []policyv1alpha1.AccessRoleBinding{{RoleBinding: *rb1.DeepCopy(), Rules: role1.Rules}},
					AccessClusterRoleBinding: []policyv1alpha1.AccessClusterRoleBinding{{ClusterRoleBinding: *crb1.DeepCopy(), Rules: cr1.Rules}},
				},
				Status: policyv1alpha1.AccessStatus{NodeList: []string{"my-node"}},
			},
			msgOpr: []string{model.DeleteOperation},
		},
		{
			name:  "reconcile failed cause serviceaccountaccess not found",
			input: saaDiffName.DeepCopy(),
			obj: []client.Object{saa1.DeepCopy(), pod1.DeepCopy(), pod2.DeepCopy(), sa1.DeepCopy(), rb1.DeepCopy(),
				crb1.DeepCopy(), cr1.DeepCopy(), role1.DeepCopy()},
			reconcileResult: controllerruntime.Result{},
			output:          &policyv1alpha1.ServiceAccountAccess{},
			msgOpr:          []string{},
		},
		{
			name:  "reconcile failed cause deletionTimestamp not nil",
			input: saaDeletion.DeepCopy(),
			obj: []client.Object{saaDeletion.DeepCopy(), pod1.DeepCopy(), sa1.DeepCopy(), rb1.DeepCopy(), crb1.DeepCopy(),
				cr1.DeepCopy(), role1.DeepCopy()},
			reconcileResult: controllerruntime.Result{},
			output: &policyv1alpha1.ServiceAccountAccess{ObjectMeta: metav1.ObjectMeta{Name: "sa1", Namespace: "my-namespace"},
				Spec: policyv1alpha1.AccessSpec{
					ServiceAccount:           *sa1.DeepCopy(),
					AccessRoleBinding:        []policyv1alpha1.AccessRoleBinding{{RoleBinding: *rb1.DeepCopy(), Rules: role1.Rules}},
					AccessClusterRoleBinding: []policyv1alpha1.AccessClusterRoleBinding{{ClusterRoleBinding: *crb1.DeepCopy(), Rules: cr1.Rules}},
				},
				Status: policyv1alpha1.AccessStatus{NodeList: []string{"my-node"}},
			},
			msgOpr: []string{},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cloudHub := &common.ModuleInfo{
				ModuleName: modules.CloudHubModuleName,
				ModuleType: common.MsgCtxTypeChannel,
			}
			beehiveContext.InitContext([]string{common.MsgCtxTypeChannel})
			beehiveContext.AddModule(cloudHub)
			var accessScheme = runtime.NewScheme()
			if err := policyv1alpha1.AddToScheme(accessScheme); err != nil {
				t.Errorf("Failed to add policyv1alpha1 scheme: %v", err)
			}
			if err := v1.AddToScheme(accessScheme); err != nil {
				t.Errorf("Failed to add v1 scheme: %v", err)
			}
			if err := rbacv1.AddToScheme(accessScheme); err != nil {
				t.Errorf("Failed to add rbacv1 scheme: %v", err)
			}
			pdStrategyTypeIndexer := func(obj client.Object) []string {
				serviceAccountName := "sa1"
				return []string{serviceAccountName}
			}
			fakeClient := fake.NewClientBuilder().WithScheme(accessScheme).WithObjects(tt.obj...).WithLists(nodeList).WithIndex(&v1.Pod{}, "spec.serviceAccountName", pdStrategyTypeIndexer).WithStatusSubresource(tt.input).Build()
			ctr := &Controller{
				Client:       fakeClient,
				Reader:       fakeClient,
				MessageLayer: messagelayer.PolicyControllerMessageLayer(),
			}
			var rst controllerruntime.Result
			var wg sync.WaitGroup
			wg.Add(1)
			go func() {
				defer wg.Done()
				inputObj := tt.input.DeepCopy()
				rst, err = ctr.Reconcile(context.Background(), controllerruntime.Request{NamespacedName: types.NamespacedName{Name: inputObj.Name, Namespace: inputObj.Namespace}})
				if err != nil {
					t.Errorf("TestCase %q Failed to syncRules: %v", tt.name, err)
				}
			}()
			var oprs []string
			for range tt.msgOpr {
				message, _ := beehiveContext.Receive(modules.CloudHubModuleName)
				oprs = append(oprs, message.GetOperation())
			}
			wg.Wait()
			sort.Strings(oprs)
			sort.Strings(tt.msgOpr)
			if !equality.Semantic.DeepEqual(oprs, tt.msgOpr) {
				t.Errorf("TestCase %q message operation got %v, want %v", tt.name, oprs, tt.msgOpr)
			}
			if !equality.Semantic.DeepEqual(rst, tt.reconcileResult) {
				t.Errorf("TestCase %q Expected: %v, got: %v", tt.name, tt.reconcileResult, rst)
			}
			saa := &policyv1alpha1.ServiceAccountAccess{}
			err = fakeClient.Get(context.Background(), types.NamespacedName{Name: tt.input.Name, Namespace: tt.input.Namespace}, saa)
			if err != nil && apierror.IsNotFound(err) {
				return
			} else if err != nil {
				t.Errorf("TestCase %q Failed to get saa: %v", tt.name, err)
			}
			if !equality.Semantic.DeepEqual((*saa).Spec.ServiceAccount, (*(tt.output)).Spec.ServiceAccount) {
				t.Errorf("TestCase %q Expected spec serviceaccount: %+v, got: %+v", tt.name, tt.output.Spec.ServiceAccount, saa.Spec.ServiceAccount)
			}
			if !equality.Semantic.DeepEqual((*saa).Spec.AccessClusterRoleBinding, (*(tt.output)).Spec.AccessClusterRoleBinding) {
				t.Errorf("TestCase %q Expected spec crb: %+v, got: %+v", tt.name, tt.output.Spec.AccessClusterRoleBinding, saa.Spec.AccessClusterRoleBinding)
			}
			if !equality.Semantic.DeepEqual((*saa).Spec.AccessRoleBinding, (*(tt.output)).Spec.AccessRoleBinding) {
				t.Errorf("TestCase %q Expected spec rb: %+v, got: %+v", tt.name, tt.output.Spec.AccessRoleBinding, saa.Spec.AccessRoleBinding)
			}
			sort.Strings(saa.Status.NodeList)
			sort.Strings(tt.output.Status.NodeList)
			if !equality.Semantic.DeepEqual(saa.Status.NodeList, tt.output.Status.NodeList) {
				t.Errorf("TestCase %q Expected status: %v, got: %v", tt.name, tt.output.Status.NodeList, saa.Status.NodeList)
			}
		})
	}
}

func TestVisitRulesForSkipsDanglingRoleRef(t *testing.T) {
	testUser := &user.DefaultInfo{Name: "test-user"}
	subjects := []rbacv1.Subject{{Kind: rbacv1.UserKind, Name: testUser.Name}}

	clusterRole := rbacv1.ClusterRole{
		ObjectMeta: metav1.ObjectMeta{Name: "cr1"},
		Rules:      []rbacv1.PolicyRule{{Verbs: []string{"get"}, APIGroups: []string{""}, Resources: []string{"pods"}}},
	}
	danglingCRB := rbacv1.ClusterRoleBinding{
		ObjectMeta: metav1.ObjectMeta{Name: "crb-dangling"},
		Subjects:   subjects,
		RoleRef:    rbacv1.RoleRef{APIGroup: rbacv1.GroupName, Kind: "ClusterRole", Name: "missing-cr"},
	}
	goodCRB := rbacv1.ClusterRoleBinding{
		ObjectMeta: metav1.ObjectMeta{Name: "crb-good"},
		Subjects:   subjects,
		RoleRef:    rbacv1.RoleRef{APIGroup: rbacv1.GroupName, Kind: "ClusterRole", Name: clusterRole.Name},
	}

	role := rbacv1.Role{
		ObjectMeta: metav1.ObjectMeta{Name: "r1", Namespace: "ns1"},
		Rules:      []rbacv1.PolicyRule{{Verbs: []string{"list"}, APIGroups: []string{""}, Resources: []string{"configmaps"}}},
	}
	danglingRB := rbacv1.RoleBinding{
		ObjectMeta: metav1.ObjectMeta{Name: "rb-dangling", Namespace: "ns1"},
		Subjects:   subjects,
		RoleRef:    rbacv1.RoleRef{APIGroup: rbacv1.GroupName, Kind: "Role", Name: "missing-role"},
	}
	goodRB := rbacv1.RoleBinding{
		ObjectMeta: metav1.ObjectMeta{Name: "rb-good", Namespace: "ns1"},
		Subjects:   subjects,
		RoleRef:    rbacv1.RoleRef{APIGroup: rbacv1.GroupName, Kind: "Role", Name: role.Name},
	}

	accessScheme := runtime.NewScheme()
	if err := rbacv1.AddToScheme(accessScheme); err != nil {
		t.Fatalf("Failed to add rbacv1 scheme: %v", err)
	}
	fakeClient := fake.NewClientBuilder().WithScheme(accessScheme).
		WithObjects(&clusterRole, &danglingCRB, &goodCRB, &role, &danglingRB, &goodRB).Build()
	ctr := &Controller{Client: fakeClient, Reader: fakeClient}

	acc := &policyv1alpha1.ServiceAccountAccess{}
	ctr.VisitRulesFor(context.Background(), testUser, "ns1", acc)

	if len(acc.Spec.AccessClusterRoleBinding) != 1 || acc.Spec.AccessClusterRoleBinding[0].ClusterRoleBinding.Name != goodCRB.Name {
		t.Errorf("expected only the resolvable clusterrolebinding %q to be visited, got %+v", goodCRB.Name, acc.Spec.AccessClusterRoleBinding)
	}
	if len(acc.Spec.AccessRoleBinding) != 1 || acc.Spec.AccessRoleBinding[0].RoleBinding.Name != goodRB.Name {
		t.Errorf("expected only the resolvable rolebinding %q to be visited, got %+v", goodRB.Name, acc.Spec.AccessRoleBinding)
	}
}
