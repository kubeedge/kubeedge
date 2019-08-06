// +build !linux

/*
Copyright 2018 The Kubernetes Authors.

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

package quota

import (
	"errors"
	"k8s.io/apimachinery/pkg/api/resource"
	"k8s.io/kubernetes/pkg/util/mount"
)

// Dummy quota implementation for systems that do not implement support
// for volume quotas

var errNotImplemented = errors.New("not implemented")

// SupportsQuotas -- dummy implementation
func SupportsQuotas(_ mount.Interface, _ string) (bool, error) {
	return false, errNotImplemented
}

// AssignQuota -- dummy implementation
func AssignQuota(_ mount.Interface, _ string, _ string, _ *resource.Quantity) error {
	return errNotImplemented
}

// GetConsumption -- dummy implementation
func GetConsumption(_ string) (*resource.Quantity, error) {
	return nil, errNotImplemented
}

// GetInodes -- dummy implementation
func GetInodes(_ string) (*resource.Quantity, error) {
	return nil, errNotImplemented
}

// ClearQuota -- dummy implementation
func ClearQuota(_ mount.Interface, _ string) error {
	return errNotImplemented
}
