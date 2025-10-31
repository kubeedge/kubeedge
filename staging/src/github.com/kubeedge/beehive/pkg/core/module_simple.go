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

package core

import klog "k8s.io/klog/v2"

// SimpleModule is the basic structure for rapid development of a module.
// In most cases, developers only focus on implementing the Start() function,
// and writing the implementation of other functions is obviously redundant work.
type SimpleModule struct {
	// name indicates the module name.
	name string
	// group indicates the module group.
	group string
	// enable indicates whether the module is enabled, default is true.
	enable bool
	// restartPolicy indicates the module restart policy.
	restartPolicy *ModuleRestartPolicy
	// StartFunc indicates the module start function.
	StartFunc func()
	// StartEFunc indicates the module start function that can return an error.
	// The module will panic the error if the function returns an error.
	StartEFunc func() error
}

var _ Module = (*SimpleModule)(nil)

func NewSimpleModule(name, group string, opts ...SimpleModuleOption) *SimpleModule {
	// new default
	m := &SimpleModule{name: name, group: group, enable: true}
	// Set options
	for _, opt := range opts {
		opt(m)
	}
	return m
}

func (m SimpleModule) Name() string {
	return m.name
}

func (m SimpleModule) Group() string {
	return m.group
}

func (m SimpleModule) Enable() bool {
	return m.enable
}

func (m SimpleModule) Start() {
	switch {
	case m.StartFunc != nil:
		m.StartFunc()
	case m.StartEFunc != nil:
		if err := m.StartEFunc(); err != nil {
			panic(err)
		}
	default:
		klog.Warningf("SimpleModule %s - %s has no start function", m.group, m.name)
	}
}

func (m SimpleModule) RestartPolicy() *ModuleRestartPolicy {
	return m.restartPolicy
}

type SimpleModuleOption func(*SimpleModule)

type SimpleModuleOptions struct {
	enable        bool
	restartPolicy *ModuleRestartPolicy
}

func WithEnable(enable bool) SimpleModuleOption {
	return func(m *SimpleModule) {
		m.enable = enable
	}
}

func WithRestartPolicy(restartPolicy *ModuleRestartPolicy) SimpleModuleOption {
	return func(m *SimpleModule) {
		m.restartPolicy = restartPolicy
	}
}
