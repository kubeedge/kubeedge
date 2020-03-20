/*
Copyright 2020 The KubeEdge Authors.

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
	"k8s.io/apimachinery/pkg/version"
	"testing"
)

func TestManagedKubernetesVersion(t *testing.T) {
	vers := version.Info{Minor: "17"}
	t.Run("test with minor version of 17", func(t *testing.T) {
		err := checkKubernetesVersion(&vers)
		if err != nil {
			t.Fatalf("checked errored with: %s\n", err)
		}
	})

	vers.Minor = "17+"
	t.Run("test with minor version of 17+", func(t *testing.T) {
		err := checkKubernetesVersion(&vers)
		if err != nil {
			t.Fatalf("checked errored with: %s\n", err)
		}
	})

	vers.Minor = "100"
	t.Run("test with minor version of 100", func(t *testing.T) {
		err := checkKubernetesVersion(&vers)
		if err != nil {
			t.Fatalf("checked errored with: %s\n", err)
		}
	})

	vers.Minor = "100+"
	t.Run("test with minor version of 100+", func(t *testing.T) {
		err := checkKubernetesVersion(&vers)
		if err != nil {
			t.Fatalf("checked errored with: %s\n", err)
		}
	})

	vers.Minor = "3"
	t.Run("test with minor version of 3", func(t *testing.T) {
		err := checkKubernetesVersion(&vers)
		if err == nil {
			t.Fatalf("check should return an error and didn't")
		}
	})

	vers.Minor = "3+"
	t.Run("test with minor version of 3+", func(t *testing.T) {
		err := checkKubernetesVersion(&vers)
		if err == nil {
			t.Fatalf("check should return an error and didn't")
		}
	})
}
