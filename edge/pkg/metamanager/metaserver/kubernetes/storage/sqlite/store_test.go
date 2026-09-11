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

package sqlite

import (
	"errors"
	"testing"
)

// TestCountReturnsErrorAndDoesNotPanic verifies that Count:
//  1. does not panic (regression guard against the original panic("implement me"))
//  2. returns exactly 0 as the count
//  3. returns a non-nil error
//  4. returns ErrCountUnsupported so callers can use errors.Is
func TestCountReturnsErrorAndDoesNotPanic(t *testing.T) {
	s := &store{}

	// Ensure Count does not panic.
	defer func() {
		if r := recover(); r != nil {
			t.Errorf("Count panicked: %v", r)
		}
	}()

	count, err := s.Count("any/key")

	if count != 0 {
		t.Errorf("Count() count = %d, want 0", count)
	}

	if err == nil {
		t.Error("Count() error = nil, want non-nil error")
	}

	if !errors.Is(err, ErrCountUnsupported) {
		t.Errorf("Count() error = %v, want errors.Is(err, ErrCountUnsupported) == true", err)
	}
}
