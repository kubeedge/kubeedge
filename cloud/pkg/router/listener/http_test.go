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
	rh := &RestHandler{}
	// sync.Map.Range visits keys in a different order for each map, so use
	// many namespaces to make the result not depend on the iteration order.
	for i := 0; i < 100; i++ {
		rh.handlers.Store(fmt.Sprintf("/ns%d/a", i), nil)
		rh.handlers.Store(fmt.Sprintf("/ns%d/a/b", i), nil)
	}

	for i := 0; i < 100; i++ {
		uri := fmt.Sprintf("/node1/ns%d/a/b", i)
		got, ok := rh.matchedPath(uri)
		if !ok {
			t.Fatalf("no rule matched %s", uri)
		}
		if want := fmt.Sprintf("/ns%d/a/b", i); got != want {
			t.Errorf("%s matched %s, want %s", uri, got, want)
		}
	}
}
