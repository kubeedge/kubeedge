/*
Copyright 2024 The KubeEdge Authors.

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

	"github.com/stretchr/testify/assert"
)

func TestGet(t *testing.T) {
	info := Get()

	// Assert that all fields in the returned Info struct match 
	// the package-level variables and runtime environment.
	assert.Equal(t, gitMajor, info.Major, "Major version mismatch")
	assert.Equal(t, gitMinor, info.Minor, "Minor version mismatch")
	assert.Equal(t, gitVersion, info.GitVersion, "GitVersion mismatch")
	assert.Equal(t, gitCommit, info.GitCommit, "GitCommit mismatch")
	assert.Equal(t, gitTreeState, info.GitTreeState, "GitTreeState mismatch")
	assert.Equal(t, buildDate, info.BuildDate, "BuildDate mismatch")
	
	assert.Equal(t, runtime.Version(), info.GoVersion, "GoVersion mismatch")
	assert.Equal(t, runtime.Compiler, info.Compiler, "Compiler mismatch")
	
	expectedPlatform := fmt.Sprintf("%s/%s", runtime.GOOS, runtime.GOARCH)
	assert.Equal(t, expectedPlatform, info.Platform, "Platform mismatch")
}