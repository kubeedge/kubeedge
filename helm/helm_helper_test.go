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
package helm

import "testing"

func TestMergeProfileValues(t *testing.T) {
	cases := []struct {
		name string
		file string
	}{
		{
			name: "load cloudcore values.yaml",
			file: "charts/cloudcore/values.yaml",
		},
		{
			name: "load profile values",
			file: "profiles/version.yaml",
		},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			vals, err := MergeProfileValues(c.file, []string{})
			if err != nil {
				t.Fatalf("faield to load helm values, err: %v", err)
			}
			if len(vals) == 0 {
				t.Fatal("the value returned is empty")
			}
		})
	}
}
