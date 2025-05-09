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

package jsonpatch

import (
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestJosnPath(t *testing.T) {
	type beer struct {
		Number int    `json:"numb"`
		Str    string `json:"str"`
	}
	cases := []struct {
		op    Operation
		path  string
		value any
		want  string
	}{
		{
			op:    OpAdd,
			path:  "/a/b/c",
			value: "123",
			want:  `[{"op":"add","path":"/a/b/c","value":"123"}]`,
		},
		{
			op:   OpRemove,
			path: "/a/b/c",
			want: `[{"op":"remove","path":"/a/b/c"}]`,
		},
		{
			op:    OpReplace,
			path:  "/a/b/c",
			value: int32(1),
			want:  `[{"op":"replace","path":"/a/b/c","value":"1"}]`,
		},
		{
			op:    OpReplace,
			path:  "/a/b/c",
			value: true,
			want:  `[{"op":"replace","path":"/a/b/c","value":"true"}]`,
		},
		{
			op:    OpReplace,
			path:  "/a/b/c",
			value: 3.14,
			want:  `[{"op":"replace","path":"/a/b/c","value":"3.14"}]`,
		},
		{
			op:    OpAdd,
			path:  "/a/b/c",
			value: beer{Number: 10, Str: "Hello"},
			want:  `[{"op":"add","path":"/a/b/c","value":"{\"numb\":10,\"str\":\"Hello\"}"}]`,
		},
	}
	for i, c := range cases {
		t.Run(fmt.Sprintf("case_%d", i), func(t *testing.T) {
			bff, err := New().Add(c.op, c.path, c.value).ToJSON()
			assert.NoError(t, err)
			assert.Equal(t, c.want, string(bff))
		})
	}
}
