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
	"testing"

	"github.com/stretchr/testify/assert"
	authenticationv1 "k8s.io/api/authentication/v1"
	"k8s.io/apimachinery/pkg/types"
)

// TestKeyFuncRequestResponseRoundTrip verifies that the key computed for the
// original (edged-issued) TokenRequest, which typically leaves
// Spec.Audiences unset, matches the key computed for the apiserver's
// response, which always populates Spec.Audiences. edged persists the
// response under the key returned by KeyFunc and later looks it up again
// using the original request, so the two keys must be equal or the local
// cache can never be hit.
func TestKeyFuncRequestResponseRoundTrip(t *testing.T) {
	exp := int64(3607)
	boundObjectRef := authenticationv1.BoundObjectReference{
		Kind:       "Pod",
		APIVersion: "v1",
		Name:       "edge-eclipse-mosquitto-bfprp",
		UID:        types.UID("21cadc2e-f849-4c62-a561-b713966ec418"),
	}

	request := &authenticationv1.TokenRequest{
		Spec: authenticationv1.TokenRequestSpec{
			// edged leaves Audiences unset when the pod doesn't request a
			// specific audience.
			Audiences:         nil,
			ExpirationSeconds: &exp,
			BoundObjectRef:    &boundObjectRef,
		},
	}
	response := &authenticationv1.TokenRequest{
		Spec: authenticationv1.TokenRequestSpec{
			// the apiserver always resolves and returns a concrete audience.
			Audiences:         []string{"https://kubernetes.default.svc.cluster.local"},
			ExpirationSeconds: &exp,
			BoundObjectRef:    &boundObjectRef,
		},
	}

	requestKey := KeyFunc("default", "kubeedge", request)
	responseKey := KeyFunc("default", "kubeedge", response)

	assert.Equal(t, requestKey, responseKey, "key derived from the request must match the key derived from the response, otherwise the persisted token is never found by a later local lookup")
}
