/*
Copyright 2019 The KubeEdge Authors.

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
	"k8s.io/kubernetes/pkg/kubelet/config"
)

type SourcesReadyFn func() bool

type sourcesReady struct {
	// sourcesReady is a function that evaluates if the sources are ready.
	sourcesReadyFn SourcesReadyFn
}

func (s *sourcesReady) AddSource(source string) {}

func (s *sourcesReady) AllReady() bool {
	return s.sourcesReadyFn()
}

//NewSourcesReady returns a new sourceready object
func NewSourcesReady(sourcesReadyFn SourcesReadyFn) config.SourcesReady {
	return &sourcesReady{
		sourcesReadyFn: sourcesReadyFn,
	}
}
