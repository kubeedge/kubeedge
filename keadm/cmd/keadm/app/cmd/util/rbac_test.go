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
	"errors"
	"strings"
	"testing"

	authv1 "k8s.io/api/authorization/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/kubernetes/fake"
	clienttesting "k8s.io/client-go/testing"

	"github.com/kubeedge/kubeedge/common/constants"
	types "github.com/kubeedge/kubeedge/keadm/cmd/keadm/app/cmd/common"
)

func TestRequiredPermissions(t *testing.T) {
	tests := []struct {
		name          string
		opts          *types.InitOptions
		wantResources map[string][]string // resource name -> expected verbs
	}{
		{
			name: "default options (all permissions checked)",
			opts: &types.InitOptions{
				SkipCRDs:            false,
				SkipPreflightChecks: false,
			},
			wantResources: map[string][]string{
				"namespaces":                {"get", "create"},
				"clusterroles":              {"get", "create"},
				"clusterrolebindings":       {"get", "create"},
				"customresourcedefinitions": {"get", "create"},
				"serviceaccounts":           {"get", "create"},
				"configmaps":                {"get", "create"},
				"secrets":                   {"get", "create"},
				"services":                  {"get", "create"},
				"deployments":               {"get", "create"},
				"pods":                      {"list", "watch"},
			},
		},
		{
			name: "skip CRDs enabled",
			opts: &types.InitOptions{
				SkipCRDs: true,
			},
			wantResources: map[string][]string{
				"namespaces":          {"get", "create"},
				"clusterroles":        {"get", "create"},
				"clusterrolebindings": {"get", "create"},
				"serviceaccounts":     {"get", "create"},
				"configmaps":          {"get", "create"},
				"secrets":             {"get", "create"},
				"services":            {"get", "create"},
				"deployments":         {"get", "create"},
				"pods":                {"list", "watch"},
			},
		},
		{
			name: "force install / wait behavior disabled",
			opts: &types.InitOptions{
				CloudInitUpdateBase: types.CloudInitUpdateBase{
					Force: true,
				},
			},
			wantResources: map[string][]string{
				"namespaces":                {"get", "create"},
				"clusterroles":              {"get", "create"},
				"clusterrolebindings":       {"get", "create"},
				"customresourcedefinitions": {"get", "create"},
				"serviceaccounts":           {"get", "create"},
				"configmaps":                {"get", "create"},
				"secrets":                   {"get", "create"},
				"services":                  {"get", "create"},
				"deployments":               {"get", "create"},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			checks := RequiredPermissions(tt.opts)
			gotResources := make(map[string][]string)
			for _, check := range checks {
				gotResources[check.Resource] = append(gotResources[check.Resource], check.Verb)
				// Ensure correct namespace-scoped resource checks
				if check.Resource == "pods" || check.Resource == "deployments" || check.Resource == "secrets" {
					if check.Namespace != constants.SystemNamespace {
						t.Errorf("expected namespace %q, got %q for resource %s", constants.SystemNamespace, check.Namespace, check.Resource)
					}
				}
			}

			if len(gotResources) != len(tt.wantResources) {
				t.Errorf("expected %d resources checked, got %d", len(tt.wantResources), len(gotResources))
			}

			for res, expectedVerbs := range tt.wantResources {
				verbs, ok := gotResources[res]
				if !ok {
					t.Fatalf("missing expected resource check: %s", res)
				}
				// verify verbs match
				for _, ev := range expectedVerbs {
					found := false
					for _, v := range verbs {
						if v == ev {
							found = true
							break
						}
					}
					if !found {
						t.Errorf("resource %s: expected verb %s not checked", res, ev)
					}
				}
			}
		})
	}
}

func TestCheckKubernetesPermissions(t *testing.T) {
	tests := []struct {
		name       string
		allowRules func(verb, resource, namespace, group string) bool
		reactErr   error
		wantErr    bool
		errSubstrs []string
	}{
		{
			name: "all permissions allowed",
			allowRules: func(verb, resource, namespace, group string) bool {
				return true
			},
			wantErr: false,
		},
		{
			name: "single permission denied",
			allowRules: func(verb, resource, namespace, group string) bool {
				return !(resource == "secrets" && verb == "create")
			},
			wantErr: true,
			errSubstrs: []string{
				"RBAC preflight check failed.",
				"secrets (namespace: kubeedge)",
				"  - create",
			},
		},
		{
			name: "multiple permissions denied",
			allowRules: func(verb, resource, namespace, group string) bool {
				if resource == "secrets" && verb == "create" {
					return false
				}
				if resource == "clusterroles" && verb == "create" {
					return false
				}
				return true
			},
			wantErr: true,
			errSubstrs: []string{
				"RBAC preflight check failed.",
				"secrets (namespace: kubeedge)",
				"  - create",
				"clusterroles",
				"  - create",
			},
		},
		{
			name: "API call failure",
			allowRules: func(verb, resource, namespace, group string) bool {
				return true
			},
			reactErr:   errors.New("connection timeout to API server"),
			wantErr:    true,
			errSubstrs: []string{"rbac preflight check API call failed", "connection timeout to API server"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cli := fake.NewSimpleClientset()
			cli.PrependReactor("create", "selfsubjectaccessreviews",
				func(action clienttesting.Action) (bool, runtime.Object, error) {
					if tt.reactErr != nil {
						return true, nil, tt.reactErr
					}
					createAction := action.(clienttesting.CreateAction)
					sar := createAction.GetObject().(*authv1.SelfSubjectAccessReview)
					allowed := tt.allowRules(
						sar.Spec.ResourceAttributes.Verb,
						sar.Spec.ResourceAttributes.Resource,
						sar.Spec.ResourceAttributes.Namespace,
						sar.Spec.ResourceAttributes.Group,
					)
					return true, &authv1.SelfSubjectAccessReview{
						Status: authv1.SubjectAccessReviewStatus{Allowed: allowed},
					}, nil
				})

			opts := &types.InitOptions{}
			checks := RequiredPermissions(opts)
			err := CheckKubernetesPermissions(cli, checks)

			if (err != nil) != tt.wantErr {
				t.Fatalf("CheckKubernetesPermissions() error = %v, wantErr %v", err, tt.wantErr)
			}

			if tt.wantErr {
				for _, sub := range tt.errSubstrs {
					if !strings.Contains(err.Error(), sub) {
						t.Errorf("expected error to contain %q, got: %v", sub, err)
					}
				}
			}
		})
	}
}
