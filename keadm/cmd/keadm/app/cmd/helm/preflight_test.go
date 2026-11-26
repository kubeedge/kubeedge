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
	"net"
	"strings"
	"testing"
	"time"

	authv1 "k8s.io/api/authorization/v1"
	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/util/wait"
	"k8s.io/client-go/kubernetes/fake"
	clienttesting "k8s.io/client-go/testing"

	"github.com/kubeedge/kubeedge/keadm/cmd/keadm/app/cmd/util"
)

func init() {
	util.PreflightBackoff = wait.Backoff{
		Steps:    3,
		Duration: 1 * time.Microsecond,
		Factor:   2.0,
	}
}

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

func TestCheckNodeReadinessRetry(t *testing.T) {
	t.Run("success after transient failure", func(t *testing.T) {
		cli := fake.NewSimpleClientset()
		calls := 0
		cli.PrependReactor("list", "nodes", func(action clienttesting.Action) (bool, runtime.Object, error) {
			calls++
			if calls == 1 {
				return true, nil, &net.OpError{Op: "dial", Net: "tcp", Err: errors.New("connection timeout")} // transient
			}
			// return ready node on second call
			nodes := &corev1.NodeList{
				Items: []corev1.Node{
					*newNode("node-1", corev1.ConditionTrue),
				},
			}
			return true, nodes, nil
		})

		err := checkNodeReadiness(cli)
		if err != nil {
			t.Fatalf("expected nil error, got: %v", err)
		}
		if calls != 2 {
			t.Fatalf("expected exactly 2 calls, got: %d", calls)
		}
	})

	t.Run("max retries exceeded with transient failures", func(t *testing.T) {
		cli := fake.NewSimpleClientset()
		calls := 0
		cli.PrependReactor("list", "nodes", func(action clienttesting.Action) (bool, runtime.Object, error) {
			calls++
			return true, nil, &net.OpError{Op: "dial", Net: "tcp", Err: errors.New("connection timeout")} // transient
		})

		err := checkNodeReadiness(cli)
		if err == nil {
			t.Fatal("expected non-nil error")
		}
		if !strings.Contains(err.Error(), "connection timeout") {
			t.Fatalf("expected connection timeout error, got: %v", err)
		}
		if calls != 3 {
			t.Fatalf("expected exactly 3 calls, got: %d", calls)
		}
	})

	t.Run("immediate fail on permanent failure", func(t *testing.T) {
		cli := fake.NewSimpleClientset()
		calls := 0
		cli.PrependReactor("list", "nodes", func(action clienttesting.Action) (bool, runtime.Object, error) {
			calls++
			// forbidden is a permanent error
			return true, nil, apierrors.NewForbidden(schema.GroupResource{Resource: "nodes"}, "", errors.New("denied"))
		})

		err := checkNodeReadiness(cli)
		if err == nil {
			t.Fatal("expected non-nil error")
		}
		if !strings.Contains(err.Error(), "forbidden") {
			t.Fatalf("expected forbidden error, got: %v", err)
		}
		if calls != 1 {
			t.Fatalf("expected exactly 1 call (no retry), got: %d", calls)
		}
	})
}
