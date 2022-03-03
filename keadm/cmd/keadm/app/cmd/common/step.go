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

package common

import (
	"fmt"
	"strconv"

	"k8s.io/klog/v2"
)

func NewStep() *Step {
	return &Step{}
}

type Step struct {
	n int
}

func (s *Step) Printf(format string, args ...interface{}) {
	s.n++
	format = strconv.Itoa(s.n) + ". " + format
	klog.InfoDepth(2, fmt.Sprintf(format, args...))
}
