/*
Copyright 2024 The KubeEdge Authors.

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

package authorization

import (
	"context"
	"testing"

	"k8s.io/apiserver/pkg/authentication/user"
	"k8s.io/apiserver/pkg/authorization/authorizer"
)

func TestAuthorize(t *testing.T) {
	var authz kubeedgeResourceAuthorizer

	tests := []struct {
		name     string
		attrs    authorizer.Attributes
		decision authorizer.Decision
		wantErr  bool
	}{
		{
			name: "kubeedge message",
			attrs: &authorizer.AttributesRecord{
				User: &user.DefaultInfo{Extra: map[string][]string{kubeedgeResourceKey: nil}},
			},
			decision: authorizer.DecisionAllow,
		},
		{
			name:     "nonkubeedge message",
			attrs:    &authorizer.AttributesRecord{},
			decision: authorizer.DecisionNoOpinion,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			decision, _, err := authz.Authorize(context.Background(), tt.attrs)
			if err != nil {
				if !tt.wantErr {
					t.Errorf("Authorize(): unexpect error: %v", err)
				}
				return
			}

			if decision != tt.decision {
				t.Errorf("Authorize() got = %v, want %v", decision, tt.decision)
			}
		})
	}
}

func TestRulesFor(t *testing.T) {
	var authz kubeedgeResourceAuthorizer

	_, _, _, err := authz.RulesFor(&user.DefaultInfo{}, "")
	if err == nil {
		t.Error("RulesFor() should not support user rule resolution")
	}
}
