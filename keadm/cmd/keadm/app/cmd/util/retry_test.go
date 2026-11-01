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
	"net"
	"strings"
	"testing"
	"time"

	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/util/wait"
)

type mockNetError struct {
	timeout bool
}

func (m mockNetError) Error() string   { return "mock net error" }
func (m mockNetError) Timeout() bool   { return m.timeout }
func (m mockNetError) Temporary() bool { return true }

func TestIsTransientError(t *testing.T) {
	tests := []struct {
		name string
		err  error
		want bool
	}{
		{
			name: "nil error is not transient",
			err:  nil,
			want: false,
		},
		{
			name: "context deadline exceeded is not transient",
			err:  context.DeadlineExceeded,
			want: false,
		},
		{
			name: "context canceled is not transient",
			err:  context.Canceled,
			want: false,
		},
		{
			name: "net timeout error is transient",
			err:  mockNetError{timeout: true},
			want: true,
		},
		{
			name: "net non-timeout error is transient (all net.Error are transient)",
			err:  mockNetError{timeout: false},
			want: true,
		},
		{
			name: "real net OpError is transient",
			err:  &net.OpError{Op: "dial", Net: "tcp", Err: errors.New("connection refused")},
			want: true,
		},
		{
			name: "real net DNSError is transient",
			err:  &net.DNSError{Err: "no such host", Name: "kubernetes.default"},
			want: true,
		},
		{
			name: "k8s too many requests is transient",
			err:  apierrors.NewTooManyRequests("too many requests", 10),
			want: true,
		},
		{
			name: "k8s service unavailable is transient",
			err:  apierrors.NewServiceUnavailable("service unavailable"),
			want: true,
		},
		{
			name: "k8s internal error is transient",
			err:  apierrors.NewInternalError(errors.New("internal error")),
			want: true,
		},
		{
			name: "k8s server timeout is transient",
			err:  apierrors.NewServerTimeout(schema.GroupResource{}, "list", 10),
			want: true,
		},
		{
			name: "k8s timeout is transient",
			err:  apierrors.NewTimeoutError("timeout", 10),
			want: true,
		},
		{
			name: "k8s unexpected server error is transient",
			err: &apierrors.StatusError{
				ErrStatus: metav1.Status{
					Status:  metav1.StatusFailure,
					Code:    500,
					Message: "unexpected server error",
					Details: &metav1.StatusDetails{
						Causes: []metav1.StatusCause{
							{
								Type:    metav1.CauseTypeUnexpectedServerResponse,
								Message: "unexpected server error",
							},
						},
					},
				},
			},
			want: true,
		},
		{
			name: "k8s forbidden is permanent",
			err:  apierrors.NewForbidden(schema.GroupResource{}, "secret", errors.New("denied")),
			want: false,
		},
		{
			name: "k8s unauthorized is permanent",
			err:  apierrors.NewUnauthorized("unauthorized"),
			want: false,
		},
		{
			name: "k8s bad request is permanent",
			err:  apierrors.NewBadRequest("bad request"),
			want: false,
		},
		{
			name: "k8s not found is permanent",
			err:  apierrors.NewNotFound(schema.GroupResource{}, "crd"),
			want: false,
		},
		{
			name: "generic random error is not transient",
			err:  errors.New("some random failure"),
			want: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := IsTransientError(tt.err); got != tt.want {
				t.Errorf("IsTransientError() = %v, want %v for err: %v", got, tt.want, tt.err)
			}
		})
	}
}

func TestRetryWithBackoff_SuccessFirstAttempt(t *testing.T) {
	calls := 0
	f := func() error {
		calls++
		return nil
	}

	backoff := wait.Backoff{
		Steps:    3,
		Duration: 1 * time.Microsecond,
		Factor:   2.0,
	}

	err := RetryWithBackoff(context.Background(), backoff, f)
	if err != nil {
		t.Fatalf("expected nil error, got: %v", err)
	}
	if calls != 1 {
		t.Fatalf("expected exactly 1 call, got: %d", calls)
	}
}

func TestRetryWithBackoff_SuccessAfterRetries(t *testing.T) {
	calls := 0
	f := func() error {
		calls++
		if calls < 3 {
			return mockNetError{timeout: true} // transient net error
		}
		return nil
	}

	backoff := wait.Backoff{
		Steps:    3,
		Duration: 1 * time.Microsecond,
		Factor:   2.0,
	}

	err := RetryWithBackoff(context.Background(), backoff, f)
	if err != nil {
		t.Fatalf("expected nil error, got: %v", err)
	}
	if calls != 3 {
		t.Fatalf("expected exactly 3 calls, got: %d", calls)
	}
}

func TestRetryWithBackoff_MaxRetriesExceeded(t *testing.T) {
	calls := 0
	transientErr := mockNetError{timeout: true}
	f := func() error {
		calls++
		return fmt.Errorf("error number %d: %w", calls, transientErr)
	}

	backoff := wait.Backoff{
		Steps:    3,
		Duration: 1 * time.Microsecond,
		Factor:   2.0,
	}

	err := RetryWithBackoff(context.Background(), backoff, f)
	if err == nil {
		t.Fatal("expected non-nil error")
	}
	if !errors.Is(err, transientErr) {
		t.Fatalf("expected error to wrap transientErr, got: %v", err)
	}
	if !strings.Contains(err.Error(), "error number 3") {
		t.Fatalf("expected error from the 3rd attempt, got: %v", err)
	}
	if calls != 3 {
		t.Fatalf("expected exactly 3 calls, got: %d", calls)
	}
}

func TestRetryWithBackoff_PermanentErrorNotRetried(t *testing.T) {
	calls := 0
	permErr := apierrors.NewForbidden(schema.GroupResource{Resource: "pods"}, "pod-1", errors.New("denied"))
	f := func() error {
		calls++
		return permErr
	}

	backoff := wait.Backoff{
		Steps:    3,
		Duration: 1 * time.Microsecond,
		Factor:   2.0,
	}

	err := RetryWithBackoff(context.Background(), backoff, f)
	if err == nil {
		t.Fatal("expected non-nil error")
	}
	var statusErr *apierrors.StatusError
	if !errors.As(err, &statusErr) || statusErr.ErrStatus.Code != 403 {
		t.Fatalf("expected 403 Forbidden error, got: %v", err)
	}
	if calls != 1 {
		t.Fatalf("expected exactly 1 call (no retries), got: %d", calls)
	}
}

func TestRetryWithBackoff_ContextCanceled(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	cancel() // Cancel immediately

	calls := 0
	f := func() error {
		calls++
		return mockNetError{timeout: true}
	}

	backoff := wait.Backoff{
		Steps:    3,
		Duration: 1 * time.Microsecond,
		Factor:   2.0,
	}

	err := RetryWithBackoff(ctx, backoff, f)
	if err == nil {
		t.Fatal("expected context canceled error, got nil")
	}
	if !errors.Is(err, context.Canceled) {
		t.Fatalf("expected context.Canceled, got: %v", err)
	}
	if calls != 0 {
		t.Fatalf("expected 0 calls due to immediate cancel, got: %d", calls)
	}
}

func TestDefaultPreflightBackoff(t *testing.T) {
	backoff := DefaultPreflightBackoff()
	if backoff.Steps != 3 {
		t.Errorf("expected 3 steps, got: %d", backoff.Steps)
	}
	if backoff.Duration != 1*time.Second {
		t.Errorf("expected 1s duration, got: %v", backoff.Duration)
	}
	if backoff.Factor != 2.0 {
		t.Errorf("expected 2.0 factor, got: %v", backoff.Factor)
	}
	if backoff.Jitter != 0.0 {
		t.Errorf("expected 0.0 jitter, got: %v", backoff.Jitter)
	}
}
