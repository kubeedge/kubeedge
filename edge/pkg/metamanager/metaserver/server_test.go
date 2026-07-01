package metaserver

import (
	"errors"
	"net/http"
	"net/url"
	"testing"

	"k8s.io/apiserver/pkg/authentication/authenticator"
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
	delegate := &mockAuthenticator{
		resp: nil,
		ok:   false,
		err:  authErr,
	}

	safeAuth := NewDiscoverySafeAuthenticator(delegate)

	tests := []struct {
		name       string
		method     string
		path       string
		wantErr    bool
	}{
		{
			name:       "a token validation error on GET /readyz falling back to anonymous authentication (error ignored)",
			method:     http.MethodGet,
			path:       "/readyz",
			wantErr:    false,
		},
		{
			name:       "the same error on a protected resource path still being returned",
			method:     http.MethodGet,
			path:       "/api/v1/pods",
			wantErr:    true,
		},
		{
			name:       "a non-GET request to a discovery path not bypassing the authentication error",
			method:     http.MethodPost,
			path:       "/readyz",
			wantErr:    true,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			req := &http.Request{
				Method: tc.method,
				URL: &url.URL{
					Path: tc.path,
				},
			}

			_, _, err := safeAuth.AuthenticateRequest(req)

			if tc.wantErr && err == nil {
				t.Errorf("expected error %v, got nil", authErr)
			}
			if !tc.wantErr && err != nil {
				t.Errorf("expected nil error, got %v", err)
			}
		})
	}
}
