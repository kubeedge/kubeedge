/*
Copyright 2022 The KubeEdge Authors.

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

package flushwriter

import (
	"bytes"
	"testing"
)

func TestFlushWriter_Write(t *testing.T) {
	data := "Hello World"

	var buf bytes.Buffer
	fw := Wrap(&buf)
	_, err := fw.Write([]byte(data))
	if err != nil {
		t.Fatal(err)
	}
	if got := buf.String(); data != got {
		t.Errorf("Writer_Write() got = %v, want = %v", got, data)
	}
}
