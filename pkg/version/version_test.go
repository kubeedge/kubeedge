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

package version

import (
	"fmt"
	"runtime"
	"testing"
)

func TestGet(t *testing.T) {
	info := Get()

	if info.Major != gitMajor {
		t.Errorf("Get() Major = %v, want %v", info.Major, gitMajor)
	}
	if info.Minor != gitMinor {
		t.Errorf("Get() Minor = %v, want %v", info.Minor, gitMinor)
	}
	if info.GitVersion != gitVersion {
		t.Errorf("Get() GitVersion = %v, want %v", info.GitVersion, gitVersion)
	}
	if info.GitCommit != gitCommit {
		t.Errorf("Get() GitCommit = %v, want %v", info.GitCommit, gitCommit)
	}
	if info.GitTreeState != gitTreeState {
		t.Errorf("Get() GitTreeState = %v, want %v", info.GitTreeState, gitTreeState)
	}
	if info.BuildDate != buildDate {
		t.Errorf("Get() BuildDate = %v, want %v", info.BuildDate, buildDate)
	}
	if info.GoVersion != runtime.Version() {
		t.Errorf("Get() GoVersion = %v, want %v", info.GoVersion, runtime.Version())
	}
	if info.Compiler != runtime.Compiler {
		t.Errorf("Get() Compiler = %v, want %v", info.Compiler, runtime.Compiler)
	}
	expectedPlatform := fmt.Sprintf("%s/%s", runtime.GOOS, runtime.GOARCH)
	if info.Platform != expectedPlatform {
		t.Errorf("Get() Platform = %v, want %v", info.Platform, expectedPlatform)
	}
}
