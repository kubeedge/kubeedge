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
	"testing"

	authenticationv1 "k8s.io/api/authentication/v1"

	"github.com/kubeedge/beehive/pkg/core/model"
	"github.com/kubeedge/kubeedge/edge/pkg/metamanager/client"
)

// TestParseResourceServiceAccountTokenResID pins the resID returned for a
// service account token.
//
// processQuery selects its lookup on resID: an empty resID makes it query the
// cached tokens by type, which returns every token on the node, and the caller
// accepts a single row only. A node running more than one token-mounting pod
// then cannot read any token back while it is offline, even when every cached
// token is fresh. resID has to stay non-empty so the token is looked up by the
// key it was stored under.
func TestParseResourceServiceAccountTokenResID(t *testing.T) {
	expirationSeconds := int64(3607)
	tr := &authenticationv1.TokenRequest{
		Spec: authenticationv1.TokenRequestSpec{
			ExpirationSeconds: &expirationSeconds,
			BoundObjectRef: &authenticationv1.BoundObjectReference{
				Kind: "Pod",
				Name: "nginx",
				UID:  "a3452707-5b1b-45e4-bc12-f5c3a7169127",
			},
		},
	}

	msg := model.NewMessage("")
	msg.SetResourceOperation("default/serviceaccounttoken/default", model.QueryOperation)
	msg.FillBody(tr)

	resKey, resType, resID := parseResource(msg)

	if resType != model.ResourceTypeServiceAccountToken {
		t.Fatalf("resType = %q, want %q", resType, model.ResourceTypeServiceAccountToken)
	}
	if want := client.KeyFunc("default", "default", tr); resKey != want {
		t.Errorf("resKey = %q, want %q", resKey, want)
	}
	if resID == "" {
		t.Error("resID is empty, so processQuery would query cached tokens by type instead of by key")
	}
}

// TestParseResourceResID covers the resource shapes parseResource accepts, so
// that the service account token case stays consistent with the rest.
func TestParseResourceResID(t *testing.T) {
	cases := []struct {
		name     string
		resource string
		wantType string
		wantID   string
	}{
		{
			name:     "namespaced resource with id",
			resource: "default/pod/nginx",
			wantType: model.ResourceTypePod,
			wantID:   "nginx",
		},
		{
			name:     "namespaced resource without id",
			resource: "default/pod",
			wantType: model.ResourceTypePod,
			wantID:   "",
		},
	}

	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			msg := model.NewMessage("")
			msg.SetResourceOperation(c.resource, model.QueryOperation)

			_, resType, resID := parseResource(msg)
			if resType != c.wantType {
				t.Errorf("resType = %q, want %q", resType, c.wantType)
			}
			if resID != c.wantID {
				t.Errorf("resID = %q, want %q", resID, c.wantID)
			}
		})
	}
}
