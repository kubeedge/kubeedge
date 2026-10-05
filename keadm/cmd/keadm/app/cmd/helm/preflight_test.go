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
package helm

import (
	"errors"
	"strings"
	"testing"

	authv1 "k8s.io/api/authorization/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/kubernetes/fake"
	clienttesting "k8s.io/client-go/testing"
)

func newNode(name string, ready corev1.ConditionStatus) *corev1.Node {
	return &corev1.Node{
		ObjectMeta: metav1.ObjectMeta{Name: name},
		Status: corev1.NodeStatus{
			Conditions: []corev1.NodeCondition{
				{Type: corev1.NodeReady, Status: ready},
			},
		},
	}
}

func newPod(name, namespace string, labels map[string]string, ready corev1.ConditionStatus) *corev1.Pod {
	return &corev1.Pod{
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: namespace,
			Labels:    labels,
		},
		Status: corev1.PodStatus{
			Phase: corev1.PodRunning,
			Conditions: []corev1.PodCondition{
				{Type: corev1.PodReady, Status: ready},
			},
		},
	}
}

func TestCheckNodeReadiness(t *testing.T) {
	tests := []struct {
		name      string
		nodes     []runtime.Object
		wantErr   bool
		errSubstr string
	}{
		{
			name:      "no nodes in cluster",
			nodes:     nil,
			wantErr:   true,
			errSubstr: "no nodes found",
		},
		{
			name: "all nodes NotReady",
			nodes: []runtime.Object{
				newNode("n1", corev1.ConditionFalse),
				newNode("n2", corev1.ConditionUnknown),
			},
			wantErr:   true,
			errSubstr: "no node is Ready",
		},
		{
			name: "at least one Ready node",
			nodes: []runtime.Object{
				newNode("n1", corev1.ConditionFalse),
				newNode("n2", corev1.ConditionTrue),
			},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cli := fake.NewSimpleClientset(tt.nodes...)
			err := checkNodeReadiness(cli)
			if (err != nil) != tt.wantErr {
				t.Fatalf("checkNodeReadiness() error = %v, wantErr %v", err, tt.wantErr)
			}
			if tt.wantErr && !strings.Contains(err.Error(), tt.errSubstr) {
				t.Fatalf("expected error containing %q, got %q", tt.errSubstr, err.Error())
			}
		})
	}
}

func TestCheckNodeReadinessListFails(t *testing.T) {
	cli := fake.NewSimpleClientset()
	cli.PrependReactor("list", "nodes", func(action clienttesting.Action) (bool, runtime.Object, error) {
		return true, nil, errors.New("api server unreachable")
	})
	err := checkNodeReadiness(cli)
	if err == nil || !strings.Contains(err.Error(), "cannot list nodes") {
		t.Fatalf("expected list-nodes error, got %v", err)
	}
}

func TestCheckCoreDNS(t *testing.T) {
	corednsLabels := map[string]string{"app": "coredns"}
	kubeDNSLabels := map[string]string{"k8s-app": "kube-dns"}

	tests := []struct {
		name      string
		pods      []runtime.Object
		wantErr   bool
		errSubstr string
	}{
		{
			name:      "no DNS pods at all",
			pods:      nil,
			wantErr:   true,
			errSubstr: "no Ready CoreDNS/kube-dns pod",
		},
		{
			name: "coredns pod exists but not Ready",
			pods: []runtime.Object{
				newPod("coredns-1", "kube-system", corednsLabels, corev1.ConditionFalse),
			},
			wantErr:   true,
			errSubstr: "no Ready CoreDNS/kube-dns pod",
		},
		{
			name: "Ready coredns pod",
			pods: []runtime.Object{
				newPod("coredns-1", "kube-system", corednsLabels, corev1.ConditionTrue),
			},
			wantErr: false,
		},
		{
			name: "Ready kube-dns pod (legacy label)",
			pods: []runtime.Object{
				newPod("kube-dns-1", "kube-system", kubeDNSLabels, corev1.ConditionTrue),
			},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cli := fake.NewSimpleClientset(tt.pods...)
			err := checkCoreDNS(cli)
			if (err != nil) != tt.wantErr {
				t.Fatalf("checkCoreDNS() error = %v, wantErr %v", err, tt.wantErr)
			}
			if tt.wantErr && !strings.Contains(err.Error(), tt.errSubstr) {
				t.Fatalf("expected error containing %q, got %q", tt.errSubstr, err.Error())
			}
		})
	}
}

func TestCheckCoreDNSListFails(t *testing.T) {
	cli := fake.NewSimpleClientset()
	cli.PrependReactor("list", "pods", func(action clienttesting.Action) (bool, runtime.Object, error) {
		return true, nil, errors.New("forbidden")
	})
	err := checkCoreDNS(cli)
	if err == nil || !strings.Contains(err.Error(), "cannot list pods in kube-system") {
		t.Fatalf("expected list-pods error, got %v", err)
	}
}

func TestIsPodReady(t *testing.T) {
	tests := []struct {
		name string
		pod  *corev1.Pod
		want bool
	}{
		{
			name: "PodReady=True",
			pod: &corev1.Pod{Status: corev1.PodStatus{Conditions: []corev1.PodCondition{
				{Type: corev1.PodReady, Status: corev1.ConditionTrue},
			}}},
			want: true,
		},
		{
			name: "PodReady=False",
			pod: &corev1.Pod{Status: corev1.PodStatus{Conditions: []corev1.PodCondition{
				{Type: corev1.PodReady, Status: corev1.ConditionFalse},
			}}},
			want: false,
		},
		{
			name: "no PodReady condition present",
			pod: &corev1.Pod{Status: corev1.PodStatus{Conditions: []corev1.PodCondition{
				{Type: corev1.PodInitialized, Status: corev1.ConditionTrue},
			}}},
			want: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := isPodReady(tt.pod); got != tt.want {
				t.Fatalf("isPodReady() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestCheckCloudCorePermissions(t *testing.T) {
	tests := []struct {
		name      string
		allow     bool
		reactErr  error
		wantErr   bool
		errSubstr string
	}{
		{
			name:    "all permissions granted",
			allow:   true,
			wantErr: false,
		},
		{
			name:      "permission denied",
			allow:     false,
			wantErr:   true,
			errSubstr: "insufficient permissions",
		},
		{
			name:      "SAR API call fails",
			reactErr:  errors.New("api server timeout"),
			wantErr:   true,
			errSubstr: "cannot verify permissions",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cli := fake.NewSimpleClientset()
			cli.PrependReactor("create", "selfsubjectaccessreviews",
				func(action clienttesting.Action) (bool, runtime.Object, error) {
					return true, &authv1.SelfSubjectAccessReview{
						Status: authv1.SubjectAccessReviewStatus{Allowed: tt.allow},
					}, tt.reactErr
				})

			err := checkCloudCorePermissions(cli)
			if (err != nil) != tt.wantErr {
				t.Fatalf("checkCloudCorePermissions() error = %v, wantErr %v", err, tt.wantErr)
			}
			if tt.wantErr && !strings.Contains(err.Error(), tt.errSubstr) {
				t.Fatalf("expected error containing %q, got %q", tt.errSubstr, err.Error())
			}
		})
	}
}
