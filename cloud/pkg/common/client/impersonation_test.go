package client

import (
	"context"
	"fmt"
	"net/http"
	"net/url"
	"testing"

	"github.com/stretchr/testify/assert"
	authenticationv1 "k8s.io/api/authentication/v1"

	ctxutl "github.com/kubeedge/kubeedge/cloud/pkg/common/context"
)

type fakeNextRoundTripper struct {
	enable bool
}

func (f *fakeNextRoundTripper) RoundTrip(req *http.Request) (*http.Response, error) {
	if f.enable {
		if vals := req.Header[authenticationv1.ImpersonateUserHeader]; len(vals) == 0 || vals[0] == "" {
			return nil, fmt.Errorf("invalid request header %s", authenticationv1.ImpersonateUserHeader)
		}
		if vals := req.Header[authenticationv1.ImpersonateGroupHeader]; len(vals) == 0 || vals[0] == "" {
			return nil, fmt.Errorf("invalid request header %s", authenticationv1.ImpersonateGroupHeader)
		}
	} else {
		if vals := req.Header[authenticationv1.ImpersonateUserHeader]; len(vals) > 0 {
			return nil, fmt.Errorf("invalid request header %s", authenticationv1.ImpersonateUserHeader)
		}
		if vals := req.Header[authenticationv1.ImpersonateGroupHeader]; len(vals) > 0 {
			return nil, fmt.Errorf("invalid request header %s", authenticationv1.ImpersonateGroupHeader)
		}
	}
	return nil, nil
}

func TestRoundTrip(t *testing.T) {
	cases := []struct {
		name   string
		enable bool
	}{
		{name: "enable impersonation", enable: true},
		{name: "disable impersonation", enable: false},
	}

	url, err := url.Parse("http://localhost:6443/apis")
	assert.NoError(t, err)
	ctx := ctxutl.WithEdgeNode(context.TODO(), "test-node")

	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			req := &http.Request{
				Method: http.MethodGet,
				URL:    url,
				Header: make(http.Header),
			}
			r := &impersonationRoundTripper{
				enable: c.enable,
				rt:     &fakeNextRoundTripper{enable: c.enable},
			}
			_, err := r.RoundTrip(req.WithContext(ctx))
			assert.NoError(t, err)
		})
	}
}
