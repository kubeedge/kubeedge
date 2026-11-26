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

package dmiserver

import (
	"testing"
	"time"
)

func TestGetTimestamp(t *testing.T) {
	before := time.Now().UnixNano() / 1e6

	ts := getTimestamp()

	after := time.Now().UnixNano() / 1e6

	if ts < before || ts > after {
		t.Fatalf("timestamp %d not between %d and %d", ts, before, after)
	}
}
