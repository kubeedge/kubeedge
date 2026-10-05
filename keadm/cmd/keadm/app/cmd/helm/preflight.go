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
	"context"
	"fmt"

	authv1 "k8s.io/api/authorization/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"

	"github.com/kubeedge/kubeedge/keadm/cmd/keadm/app/cmd/util"
)

func (c *CloudCoreHelmTool) runPreflightChecks(kubeConfig string, skip bool) error {
	if skip {
		fmt.Println("Skipping pre-flight checks.")
		return nil
	}

	fmt.Println("Running pre-flight checks...")

	cli, err := util.KubeClient(kubeConfig)
	if err != nil {
		return fmt.Errorf("pre-flight check failed: cannot create kube client, err: %v", err)
	}

	if err := checkNodeReadiness(cli); err != nil {
		return err
	}
	if err := checkCoreDNS(cli); err != nil {
		return err
	}
	if err := checkCloudCorePermissions(cli); err != nil {
		return err
	}

	fmt.Println("Pre-flight checks passed.")
	return nil
}

// checkNodeReadiness verifies at least one node in the cluster is Ready.
// cloudcore depends on the node registry being functional before it can schedule edge workloads.
func checkNodeReadiness(cli kubernetes.Interface) error {
	nodes, err := cli.CoreV1().Nodes().List(context.Background(), metav1.ListOptions{})
	if err != nil {
		return fmt.Errorf("pre-flight check failed: cannot list nodes, err: %v", err)
	}
	if len(nodes.Items) == 0 {
		return fmt.Errorf("pre-flight check failed: no nodes found in the cluster")
	}
	for _, node := range nodes.Items {
		for _, cond := range node.Status.Conditions {
			if cond.Type == corev1.NodeReady && cond.Status == corev1.ConditionTrue {
				return nil
			}
		}
	}
	return fmt.Errorf("pre-flight check failed: no node is Ready, the Kubernetes cluster may be unhealthy; " +
		"run 'kubectl get nodes' to investigate before retrying")
}

// checkCoreDNS verifies that at least one CoreDNS (or kube-dns) pod is Ready in kube-system.
// A missing DNS component causes cloudcore webhook and certificate operations to fail silently.
func checkCoreDNS(cli kubernetes.Interface) error {
	selectors := []string{"app=coredns", "k8s-app=kube-dns"}
	for _, sel := range selectors {
		pods, err := cli.CoreV1().Pods("kube-system").List(context.Background(), metav1.ListOptions{
			LabelSelector: sel,
		})
		if err != nil {
			return fmt.Errorf("pre-flight check failed: cannot list pods in kube-system, err: %v", err)
		}
		for _, pod := range pods.Items {
			if isPodReady(&pod) {
				return nil
			}
		}
	}
	return fmt.Errorf("pre-flight check failed: no Ready CoreDNS/kube-dns pod found in kube-system; " +
		"DNS must be healthy before installing cloudcore")
}

func isPodReady(pod *corev1.Pod) bool {
	for _, cond := range pod.Status.Conditions {
		if cond.Type == corev1.PodReady && cond.Status == corev1.ConditionTrue {
			return true
		}
	}
	return false
}

// checkCloudCorePermissions uses SelfSubjectAccessReview to verify the kubeconfig has
// sufficient privileges for the operations cloudcore requires at install time.
func checkCloudCorePermissions(cli kubernetes.Interface) error {
	required := []authv1.ResourceAttributes{
		{Verb: "list", Resource: "nodes"},
		{Verb: "list", Resource: "pods"},
		{Verb: "create", Resource: "namespaces"},
		{Verb: "create", Resource: "clusterroles", Group: "rbac.authorization.k8s.io"},
		{Verb: "create", Resource: "clusterrolebindings", Group: "rbac.authorization.k8s.io"},
	}
	for _, attr := range required {
		attr := attr
		sar := &authv1.SelfSubjectAccessReview{
			Spec: authv1.SelfSubjectAccessReviewSpec{
				ResourceAttributes: &attr,
			},
		}
		result, err := cli.AuthorizationV1().SelfSubjectAccessReviews().Create(
			context.Background(), sar, metav1.CreateOptions{},
		)
		if err != nil {
			return fmt.Errorf("pre-flight check failed: cannot verify permissions for %s %s, err: %v",
				attr.Verb, attr.Resource, err)
		}
		if !result.Status.Allowed {
			return fmt.Errorf("pre-flight check failed: insufficient permissions to %s %s; "+
				"ensure the kubeconfig has cluster-admin privileges before running keadm init",
				attr.Verb, attr.Resource)
		}
	}
	return nil
}
