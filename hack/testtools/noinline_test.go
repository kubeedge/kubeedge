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

package testtools

import (
	"os"
	"testing"
)

func TestRequireNoInline_SkipsWhenEnvUnset(t *testing.T) {
	old, had := os.LookupEnv(EnvNoInline)
	os.Unsetenv(EnvNoInline)
	defer func() {
		if had {
			os.Setenv(EnvNoInline, old)
		}
	}()

	var sub *testing.T
	t.Run("subtest", func(st *testing.T) {
		sub = st
		RequireNoInline(st)
	})

	if !sub.Skipped() {
		t.Fatal("expected RequireNoInline to skip the test when KUBEEDGE_TEST_NOINLINE is unset")
	}
}

func TestRequireNoInline_PassesWhenEnvSet(t *testing.T) {
	old, had := os.LookupEnv(EnvNoInline)
	os.Setenv(EnvNoInline, "1")
	defer func() {
		if had {
			os.Setenv(EnvNoInline, old)
		} else {
			os.Unsetenv(EnvNoInline)
		}
	}()

	RequireNoInline(t)

	if t.Skipped() {
		t.Fatal("expected RequireNoInline not to skip the test when KUBEEDGE_TEST_NOINLINE=1")
	}
}
