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

package util

import (
	"context"
	"errors"
	"fmt"
	"strings"

	authv1 "k8s.io/api/authorization/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"

	"github.com/kubeedge/kubeedge/common/constants"
	types "github.com/kubeedge/kubeedge/keadm/cmd/keadm/app/cmd/common"
)

type rbacCheck struct {
	Verb      string
	Resource  string
	Namespace string
	Group     string
}

// RequiredPermissions determines installer-level permissions dynamically based on InitOptions
func RequiredPermissions(opts *types.InitOptions) []rbacCheck {
	var checks []rbacCheck

	// Cluster-scoped permissions
	clusterResources := []string{"namespaces", "clusterroles", "clusterrolebindings"}
	for _, res := range clusterResources {
		group := ""
		if res == "clusterroles" || res == "clusterrolebindings" {
			group = "rbac.authorization.k8s.io"
		}
		checks = append(checks, rbacCheck{Verb: "get", Resource: res, Group: group})
		checks = append(checks, rbacCheck{Verb: "create", Resource: res, Group: group})
	}

	if opts == nil || !opts.SkipCRDs {
		checks = append(checks, rbacCheck{Verb: "get", Resource: "customresourcedefinitions", Group: "apiextensions.k8s.io"})
		checks = append(checks, rbacCheck{Verb: "create", Resource: "customresourcedefinitions", Group: "apiextensions.k8s.io"})
	}

	// Namespace-scoped permissions (defaults to constants.SystemNamespace / "kubeedge")
	ns := constants.SystemNamespace
	nsResources := []string{"serviceaccounts", "configmaps", "secrets", "services", "deployments"}
	for _, res := range nsResources {
		group := ""
		if res == "deployments" {
			group = "apps"
		}
		checks = append(checks, rbacCheck{Verb: "get", Resource: res, Namespace: ns, Group: group})
		checks = append(checks, rbacCheck{Verb: "create", Resource: res, Namespace: ns, Group: group})
	}

	// Conditional namespace-scoped permissions (only if Helm wait behavior is enabled)
	if opts == nil || !opts.Force {
		checks = append(checks, rbacCheck{Verb: "list", Resource: "pods", Namespace: ns})
		checks = append(checks, rbacCheck{Verb: "watch", Resource: "pods", Namespace: ns})
	}

	return checks
}

// CheckKubernetesPermissions executes SelfSubjectAccessReviews for all requested permissions
func CheckKubernetesPermissions(client kubernetes.Interface, checks []rbacCheck) error {
	var denied []rbacCheck

	for _, check := range checks {
		sar := &authv1.SelfSubjectAccessReview{
			Spec: authv1.SelfSubjectAccessReviewSpec{
				ResourceAttributes: &authv1.ResourceAttributes{
					Namespace: check.Namespace,
					Verb:      check.Verb,
					Group:     check.Group,
					Resource:  check.Resource,
				},
			},
		}

		result, err := client.AuthorizationV1().SelfSubjectAccessReviews().Create(
			context.Background(),
			sar,
			metav1.CreateOptions{},
		)
		if err != nil {
			return fmt.Errorf("rbac preflight check API call failed: %v", err)
		}

		if !result.Status.Allowed {
			denied = append(denied, check)
		}
	}

	if len(denied) > 0 {
		return formatGroupedRBACError(denied)
	}

	return nil
}

type resourceKey struct {
	Resource  string
	Namespace string
}

func formatGroupedRBACError(denied []rbacCheck) error {
	var orderedKeys []resourceKey
	groupedMap := make(map[resourceKey][]string)

	for _, check := range denied {
		key := resourceKey{Resource: check.Resource, Namespace: check.Namespace}
		if _, exists := groupedMap[key]; !exists {
			orderedKeys = append(orderedKeys, key)
		}
		groupedMap[key] = append(groupedMap[key], check.Verb)
	}

	var sb strings.Builder
	sb.WriteString("RBAC preflight check failed.\n\nMissing Kubernetes permissions:\n")

	for _, key := range orderedKeys {
		sb.WriteString("\n")
		if key.Namespace != "" {
			sb.WriteString(fmt.Sprintf("%s (namespace: %s)\n", key.Resource, key.Namespace))
		} else {
			sb.WriteString(fmt.Sprintf("%s\n", key.Resource))
		}
		for _, verb := range groupedMap[key] {
			sb.WriteString(fmt.Sprintf("  - %s\n", verb))
		}
	}

	sb.WriteString("\nPlease grant required permissions and retry.")
	return errors.New(sb.String())
}
