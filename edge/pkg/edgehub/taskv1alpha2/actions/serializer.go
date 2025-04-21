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

package actions

import "fmt"

type SpecSerializer interface {
	GetSpec() any
}

func NewSpecSerializer(data []byte, serializeFn func(d []byte) (any, error),
) (SpecSerializer, error) {
	spec, err := serializeFn(data)
	if err != nil {
		return nil, fmt.Errorf("failed to serialize the specm, err: %v", err)
	}
	return &cachedSpecSerializer{
		spec: spec,
	}, nil
}

type cachedSpecSerializer struct {
	spec any
}

func (s cachedSpecSerializer) GetSpec() any {
	return s.spec
}
