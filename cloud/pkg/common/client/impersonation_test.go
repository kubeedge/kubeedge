package client

import (
	"context"
	"fmt"
	"net/http"
	"net/url"
	"testing"

	"github.com/stretchr/testify/assert"
	authenticationv1 "k8s.io/api/authentication/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/rest"

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

func TestHttpClientFor(t *testing.T) {
	cases := []struct {
		name                string
		enableImpersonation bool
	}{
		{name: "impersonation enabled", enableImpersonation: true},
		{name: "impersonation disabled", enableImpersonation: false},
	}

	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			cfg := &rest.Config{
				Host: "http://localhost:6443",
			}
			client, err := httpClientFor(cfg, c.enableImpersonation)
			assert.NoError(t, err)
			assert.NotNil(t, client)

			rt, ok := client.Transport.(*impersonationRoundTripper)
			assert.True(t, ok)
			assert.Equal(t, c.enableImpersonation, rt.enable)
		})
	}
}

func invalidTLSConfig() *rest.Config {
	return &rest.Config{
		Host: "http://localhost:6443",
		TLSClientConfig: rest.TLSClientConfig{
			CertData: []byte("invalid-cert"),
			KeyData:  []byte("invalid-key"),
		},
	}
}

func TestHttpClientFor_TransportError(t *testing.T) {
	cfg := invalidTLSConfig()
	client, err := httpClientFor(cfg, false)
	assert.Error(t, err)
	assert.Nil(t, client)
}

func TestNewForK8sConfigOrDie(t *testing.T) {
	cases := []struct {
		name                string
		enableImpersonation bool
	}{
		{name: "impersonation enabled", enableImpersonation: true},
		{name: "impersonation disabled", enableImpersonation: false},
	}

	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			cfg := &rest.Config{
				Host: "http://localhost:6443",
			}
			assert.NotPanics(t, func() {
				cs := newForK8sConfigOrDie(cfg, c.enableImpersonation)
				assert.NotNil(t, cs)
			})
		})
	}
}

func TestNewForK8sConfigOrDie_WithUserAgent(t *testing.T) {
	cfg := &rest.Config{
		Host:      "http://localhost:6443",
		UserAgent: "custom-agent/1.0",
	}
	assert.NotPanics(t, func() {
		cs := newForK8sConfigOrDie(cfg, false)
		assert.NotNil(t, cs)
	})
}

func TestNewForK8sConfigOrDie_PanicOnTransportError(t *testing.T) {
	cfg := invalidTLSConfig()
	assert.Panics(t, func() {
		newForK8sConfigOrDie(cfg, false)
	})
}

func TestNewForDynamicConfigOrDie(t *testing.T) {
	cases := []struct {
		name                string
		enableImpersonation bool
	}{
		{name: "impersonation enabled", enableImpersonation: true},
		{name: "impersonation disabled", enableImpersonation: false},
	}

	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			cfg := &rest.Config{
				Host: "http://localhost:6443",
			}
			assert.NotPanics(t, func() {
				cs := newForDynamicConfigOrDie(cfg, c.enableImpersonation)
				assert.NotNil(t, cs)
			})
		})
	}
}

func TestNewForDynamicConfigOrDie_PanicOnTransportError(t *testing.T) {
	cfg := invalidTLSConfig()
	assert.Panics(t, func() {
		newForDynamicConfigOrDie(cfg, false)
	})
}

func TestNewForCrdConfigOrDie(t *testing.T) {
	cases := []struct {
		name                string
		enableImpersonation bool
	}{
		{name: "impersonation enabled", enableImpersonation: true},
		{name: "impersonation disabled", enableImpersonation: false},
	}

	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			cfg := &rest.Config{
				Host: "http://localhost:6443",
			}
			assert.NotPanics(t, func() {
				cs := newForCrdConfigOrDie(cfg, c.enableImpersonation)
				assert.NotNil(t, cs)
			})
		})
	}
}

func TestNewForCrdConfigOrDie_WithUserAgent(t *testing.T) {
	cfg := &rest.Config{
		Host:      "http://localhost:6443",
		UserAgent: "custom-agent/1.0",
	}
	assert.NotPanics(t, func() {
		cs := newForCrdConfigOrDie(cfg, false)
		assert.NotNil(t, cs)
	})
}

func TestNewForCrdConfigOrDie_PanicOnTransportError(t *testing.T) {
	cfg := invalidTLSConfig()
	assert.Panics(t, func() {
		newForCrdConfigOrDie(cfg, false)
	})
}

func TestNewForK8sConfigOrDie_PanicOnClientError(t *testing.T) {
	assert.Panics(t, func() {
		cfg := &rest.Config{
			Host: "http://localhost:6443",
			ContentConfig: rest.ContentConfig{
				NegotiatedSerializer: runtime.NewSimpleNegotiatedSerializer(runtime.SerializerInfo{}),
			},
		}
		newForK8sConfigOrDie(cfg, false)
	})
}

func TestNewForDynamicConfigOrDie_PanicOnClientError(t *testing.T) {
	assert.Panics(t, func() {
		cfg := &rest.Config{
			Host: "http://localhost:6443",
			ContentConfig: rest.ContentConfig{
				NegotiatedSerializer: runtime.NewSimpleNegotiatedSerializer(runtime.SerializerInfo{}),
			},
		}
		newForDynamicConfigOrDie(cfg, false)
	})
}

func TestNewForCrdConfigOrDie_PanicOnClientError(t *testing.T) {
	assert.Panics(t, func() {
		cfg := &rest.Config{
			Host: "http://localhost:6443",
			ContentConfig: rest.ContentConfig{
				NegotiatedSerializer: runtime.NewSimpleNegotiatedSerializer(runtime.SerializerInfo{}),
			},
		}
		newForCrdConfigOrDie(cfg, false)
	})
}
