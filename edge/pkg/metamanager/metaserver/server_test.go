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

package metaserver

import (
	"errors"
	"net/http"
	"net/url"
	"testing"

	"k8s.io/apiserver/pkg/authentication/authenticator"
	"k8s.io/apiserver/pkg/authentication/user"
)

// mockAuthenticator always returns the configured response and error.
type mockAuthenticator struct {
	resp *authenticator.Response
	ok   bool
	err  error
}

func (m *mockAuthenticator) AuthenticateRequest(req *http.Request) (*authenticator.Response, bool, error) {
	return m.resp, m.ok, m.err
}

func TestDiscoverySafeAuthenticator(t *testing.T) {
	authErr := errors.New("token validation error")

	successResp := &authenticator.Response{
		User: &user.DefaultInfo{Name: "test-user"},
	}

	tests := []struct {
		name       string
		method     string
		path       string
		delegateOk bool
		delegateErr error
		wantResp   *authenticator.Response
		wantOk     bool
		wantErr    error
	}{
		{
			name:        "a token validation error on GET /readyz falling back to anonymous authentication (error ignored)",
			method:      http.MethodGet,
			path:        "/readyz",
			delegateOk:  false,
			delegateErr: authErr,
			wantResp:    nil,
			wantOk:      false,
			wantErr:     nil, // Error is bypassed
		},
		{
			name:        "the same error on a protected resource path still being returned",
			method:      http.MethodGet,
			path:        "/api/v1/pods",
			delegateOk:  false,
			delegateErr: authErr,
			wantResp:    nil,
			wantOk:      false,
			wantErr:     authErr,
		},
		{
			name:        "a non-GET request to a discovery path not bypassing the authentication error",
			method:      http.MethodPost,
			path:        "/readyz",
			delegateOk:  false,
			delegateErr: authErr,
			wantResp:    nil,
			wantOk:      false,
			wantErr:     authErr,
		},
		{
			name:        "delegate success case confirms that resp, ok, and err are returned unchanged",
			method:      http.MethodGet,
			path:        "/readyz",
			delegateOk:  true,
			delegateErr: nil,
			wantResp:    successResp,
			wantOk:      true,
			wantErr:     nil,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			delegate := &mockAuthenticator{
				resp: tc.wantResp, // Only populated in the success case
				ok:   tc.delegateOk,
				err:  tc.delegateErr,
			}
			safeAuth := newDiscoverySafeAuthenticator(delegate)

			req := &http.Request{
				Method: tc.method,
				URL: &url.URL{
					Path: tc.path,
				},
			}

			resp, ok, err := safeAuth.AuthenticateRequest(req)

			if resp != tc.wantResp {
				t.Errorf("expected resp %v, got %v", tc.wantResp, resp)
			}
			if ok != tc.wantOk {
				t.Errorf("expected ok %v, got %v", tc.wantOk, ok)
			}

			if tc.wantErr == nil {
				if err != nil {
					t.Errorf("expected nil error, got %v", err)
				}
			} else {
				if !errors.Is(err, tc.wantErr) {
					t.Errorf("expected error %v, got %v", tc.wantErr, err)
				}
			}
		})
	}
}
