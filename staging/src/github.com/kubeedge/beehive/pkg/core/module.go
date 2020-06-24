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

package core

import (
	"k8s.io/klog"
)

const (
	tryReadKeyTimes = 5
)

// Module interface
type Module interface {
	Name() string
	Group() string
	Start()
	Enable() bool
}

var (
	// Modules map
	modules         map[string]Module
	disabledModules map[string]Module
)

func init() {
	modules = make(map[string]Module)
	disabledModules = make(map[string]Module)
}

// Register register module
func Register(m Module) {
	if m.Enable() {
		modules[m.Name()] = m
		klog.Infof("Module %v registered successfully", m.Name())
	} else {
		disabledModules[m.Name()] = m
		klog.Warningf("Module %v is disabled, do not register", m.Name())
	}
}

// GetModules gets modules map
func GetModules() map[string]Module {
	return modules
}
