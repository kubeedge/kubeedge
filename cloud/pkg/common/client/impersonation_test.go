package client

import (
	"context"
	"fmt"
	"net/http"
	"net/url"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	authenticationv1 "k8s.io/api/authentication/v1"
	"k8s.io/client-go/rest"

	ctxutl "github.com/kubeedge/kubeedge/cloud/pkg/common/context"
)

const (
	testHost       = "http://localhost:6443"
	testAPIPath    = "/apis"
	testUser       = "test-user"
	testGroup      = "test-group"
	testNode       = "test-node"
	defaultTimeout = 30 * time.Second
)

type contextKey string

const (
	userContextKey  contextKey = "impersonate-user"
	groupContextKey contextKey = "impersonate-group"
)

func WithUser(ctx context.Context, user string) context.Context {
	return context.WithValue(ctx, userContextKey, user)
}

func WithGroup(ctx context.Context, group string) context.Context {
	return context.WithValue(ctx, groupContextKey, group)
}

func GetUser(ctx context.Context) (string, bool) {
	user, ok := ctx.Value(userContextKey).(string)
	return user, ok
}

func GetGroup(ctx context.Context) (string, bool) {
	group, ok := ctx.Value(groupContextKey).(string)
	return group, ok
}

type testCase[T any] struct {
	name          string
	input         T
	enable        bool
	expectedError bool
}

type fakeNextRoundTripper struct {
	enable bool
	err    error
}

func (f *fakeNextRoundTripper) RoundTrip(req *http.Request) (*http.Response, error) {
	if f.err != nil {
		return nil, f.err
	}

	if err := f.validateHeaders(req); err != nil {
		return nil, err
	}

	return &http.Response{
		StatusCode: http.StatusOK,
		Body:       http.NoBody,
	}, nil
}

func (f *fakeNextRoundTripper) validateHeaders(req *http.Request) error {
	if !f.enable {
		return nil
	}

	if err := validateHeader(req.Header, authenticationv1.ImpersonateUserHeader); err != nil {
		return err
	}
	return validateHeader(req.Header, authenticationv1.ImpersonateGroupHeader)
}

func validateHeader(headers http.Header, headerName string) error {
	vals := headers[headerName]
	if len(vals) > 0 && len(vals[0]) == 0 {
		return fmt.Errorf("empty %s header", headerName)
	}
	return nil
}

func setupTestRequest(t *testing.T) *http.Request {
	testURL, err := url.Parse(testHost + testAPIPath)
	require.NoError(t, err, "Failed to parse test URL")

	return &http.Request{
		Method: http.MethodGet,
		URL:    testURL,
		Header: make(http.Header),
	}
}

func setupTestContext(user, group string) context.Context {
	ctx := context.TODO()
	if user != "" {
		ctx = WithUser(ctx, user)
	}
	if group != "" {
		ctx = WithGroup(ctx, group)
	}
	return ctxutl.WithEdgeNode(ctx, testNode)
}

func TestRoundTrip(t *testing.T) {
	cases := []testCase[struct {
		user         string
		group        string
		roundTripErr error
	}]{
		{
			name: "enable impersonation with valid user and group",
			input: struct {
				user         string
				group        string
				roundTripErr error
			}{
				user:  testUser,
				group: testGroup,
			},
			enable: true,
		},
		{
			name:   "disable impersonation",
			enable: false,
		},
		{
			name: "roundtrip error",
			input: struct {
				user         string
				group        string
				roundTripErr error
			}{
				user:         testUser,
				group:        testGroup,
				roundTripErr: fmt.Errorf("roundtrip error"),
			},
			enable:        true,
			expectedError: true,
		},
	}

	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			req := setupTestRequest(t)
			ctx := setupTestContext(c.input.user, c.input.group)

			r := &impersonationRoundTripper{
				enable: c.enable,
				rt:     &fakeNextRoundTripper{enable: c.enable, err: c.input.roundTripErr},
			}

			resp, err := r.RoundTrip(req.WithContext(ctx))
			assertResponse(t, resp, err, c.expectedError)
		})
	}
}

func assertResponse(t *testing.T, resp *http.Response, err error, expectedError bool) {
	if expectedError {
		assert.Error(t, err)
		assert.Nil(t, resp)
		return
	}
	assert.NoError(t, err)
	assert.NotNil(t, resp)
	assert.Equal(t, http.StatusOK, resp.StatusCode)
}

func TestClientCreation(t *testing.T) {
	validConfig := &rest.Config{
		Host:    "https://localhost:6443",
		Timeout: defaultTimeout,
	}

	t.Run("valid configurations", func(t *testing.T) {
		testValidConfigurations(t, validConfig)
	})

	invalidConfig := &rest.Config{
		Host: "://invalid-url",
	}

	t.Run("invalid configurations", func(t *testing.T) {
		testInvalidConfigurations(t, invalidConfig)
	})
}

func testValidConfigurations(t *testing.T, config *rest.Config) {
	tests := []struct {
		name     string
		createFn func(*rest.Config, bool) interface{}
	}{
		{
			name:     "newForK8sConfigOrDie",
			createFn: func(c *rest.Config, b bool) interface{} { return newForK8sConfigOrDie(c, b) },
		},
		{
			name:     "newForDynamicConfigOrDie",
			createFn: func(c *rest.Config, b bool) interface{} { return newForDynamicConfigOrDie(c, b) },
		},
		{
			name:     "newForCrdConfigOrDie",
			createFn: func(c *rest.Config, b bool) interface{} { return newForCrdConfigOrDie(c, b) },
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.NotPanics(t, func() {
				client := tt.createFn(config, false)
				assert.NotNil(t, client)
			})
		})
	}
}

func testInvalidConfigurations(t *testing.T, config *rest.Config) {
	tests := []struct {
		name     string
		createFn func(*rest.Config, bool)
	}{
		{
			name:     "newForK8sConfigOrDie",
			createFn: func(c *rest.Config, b bool) { _ = newForK8sConfigOrDie(c, b) },
		},
		{
			name:     "newForDynamicConfigOrDie",
			createFn: func(c *rest.Config, b bool) { _ = newForDynamicConfigOrDie(c, b) },
		},
		{
			name:     "newForCrdConfigOrDie",
			createFn: func(c *rest.Config, b bool) { _ = newForCrdConfigOrDie(c, b) },
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Panics(t, func() {
				tt.createFn(config, false)
			})
		})
	}
}
