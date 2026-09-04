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
	"net"
	"time"

	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/util/wait"
	"k8s.io/klog/v2"
)

// IsTransientError determines whether a Kubernetes API or network error is transient.
func IsTransientError(err error) bool {
	if err == nil {
		return false
	}

	// Context cancellation should terminate immediately.
	// These are not transient retryable errors.
	if errors.Is(err, context.Canceled) || errors.Is(err, context.DeadlineExceeded) {
		return false
	}

	// 2. Standard library network errors (TCP timeouts, refused connections, resets, DNS drops)
	var netErr net.Error
	if errors.As(err, &netErr) {
		return true
	}

	// 3. Kubernetes API specific transient errors (timeouts, rate limits, temporary unavailability)
	if apierrors.IsTimeout(err) ||
		apierrors.IsServerTimeout(err) ||
		apierrors.IsTooManyRequests(err) ||
		apierrors.IsServiceUnavailable(err) ||
		apierrors.IsInternalError(err) ||
		apierrors.IsUnexpectedServerError(err) {
		return true
	}

	return false
}

// PreflightBackoff is the overridable backoff config (1µs base during testing, 1s base in production).
var PreflightBackoff = wait.Backoff{
	Steps:    3,
	Duration: 1 * time.Second,
	Factor:   2.0,
	Jitter:   0.0,
}

// DefaultPreflightBackoff returns the standard exponential backoff configuration.
func DefaultPreflightBackoff() wait.Backoff {
	return PreflightBackoff
}

// RetryWithBackoff executes f, retrying transient errors using wait.ExponentialBackoff while respecting context cancellation.
func RetryWithBackoff(ctx context.Context, backoff wait.Backoff, f func() error) error {
	var lastErr error
	attempt := 0
	currentDelay := backoff.Duration

	err := wait.ExponentialBackoff(backoff, func() (bool, error) {
		// 1. Verify context is still active before performing the request
		if ctxErr := ctx.Err(); ctxErr != nil {
			return true, ctxErr
		}

		attempt++
		err := f()
		if err == nil {
			if attempt > 1 {
				klog.Infof("Preflight check recovered successfully after retry")
			}
			return true, nil
		}

		lastErr = err

		// 2. Do not retry permanent errors (e.g. Forbidden, Unauthorized, Bad Request)
		if !IsTransientError(err) {
			return false, err
		}

		// 3. Log warning and continue retrying if steps remain
		if attempt < backoff.Steps {
			klog.Warningf("Preflight check attempt %d/%d failed: %v. Retrying in %v...", attempt, backoff.Steps, err, currentDelay)
			currentDelay = time.Duration(float64(currentDelay) * backoff.Factor)
			return false, nil
		}

		return false, nil
	})

	if err == wait.ErrWaitTimeout {
		return lastErr
	}
	return err
}
