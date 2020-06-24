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
	"errors"
	"fmt"
	"strings"
	"testing"
)

func TestSpliceErrors(t *testing.T) {
	err1 := errors.New("this is error 1")
	err2 := errors.New("this is error 2")
	err3 := errors.New("this is error 3")

	const head = "[\n"
	var line1 = fmt.Sprintf("  %s\n", err1)
	var line2 = fmt.Sprintf("  %s\n", err2)
	var line3 = fmt.Sprintf("  %s\n", err3)
	const tail = "]\n"

	sliceOutput := SpliceErrors([]error{err1, err2, err3})
	if strings.Index(sliceOutput, head) != 0 ||
		strings.Index(sliceOutput, line1) != len(head) ||
		strings.Index(sliceOutput, line2) != len(head+line1) ||
		strings.Index(sliceOutput, line3) != len(head+line1+line2) ||
		strings.Index(sliceOutput, tail) != len(head+line1+line2+line3) {
		t.Error("the func format the multiple elements error slice unexpected")
		return
	}

	if SpliceErrors([]error{}) != "" || SpliceErrors(nil) != "" {
		t.Error("the func format the zero-length error slice unexpected")
		return
	}
}
