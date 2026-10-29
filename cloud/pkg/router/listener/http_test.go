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

package listener

import (
	"fmt"
	"testing"
)

func TestMatchedPathPrefersMoreSpecificRule(t *testing.T) {
	for i := 0; i < 100; i++ {
		rh := &RestHandler{}
		broad := fmt.Sprintf("/ns%d/a", i)
		specific := fmt.Sprintf("/ns%d/a/b", i)
		rh.handlers.Store(broad, nil)
		rh.handlers.Store(specific, nil)

		got, ok := rh.matchedPath(fmt.Sprintf("/node1/ns%d/a/b", i))
		if !ok {
			t.Fatalf("expected a matched path for namespace ns%d", i)
		}
		if got != specific {
			t.Errorf("expected %q, got %q", specific, got)
		}
	}
}
