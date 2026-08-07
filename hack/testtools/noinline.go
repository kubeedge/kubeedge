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

// Package testtools provides small test-only helpers shared across
// KubeEdge's cloud and edge test suites.
package testtools

import (
	"os"
	"testing"
)

// EnvNoInline is the environment variable set by `make test` (via
// hack/lib/golang.sh) to signal that the test binary is being built and
// run with inlining disabled (`-gcflags "all=-N -l"`).
const EnvNoInline = "KUBEEDGE_TEST_NOINLINE"

// RequireNoInline skips the current test unless it detects that inlining
// optimizations are disabled for this test run.
//
// Many tests in cloud/pkg and edge/pkg use gomonkey to patch package-level
// functions. gomonkey works by overwriting the target function's compiled
// machine code, which requires the target to not be inlined at its call
// sites. When such tests are run without `-gcflags=all=-l` (e.g. via a
// plain `go test ./...`, `gotestsum`, or an IDE's built-in test runner),
// the compiler may inline the patched function, and gomonkey's patch can
// corrupt unrelated code, crashing the whole test process with a SIGSEGV
// instead of failing the individual test.
//
// Call RequireNoInline at the start of any gomonkey-based test, before
// applying any patches, so unsupported test runs fail cleanly with an
// actionable skip message instead of crashing.
func RequireNoInline(t *testing.T) {
	t.Helper()
	if os.Getenv(EnvNoInline) != "1" {
		t.Skip("skipping gomonkey-based test: inlining is not confirmed disabled for this run; " +
			"run with 'make test' or 'go test -gcflags=all=-l ...' (gomonkey requires non-inlined targets, see hack/testtools/noinline.go)")
	}
}
