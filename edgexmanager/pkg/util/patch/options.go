/*
Copyright 2020 The Kubernetes Authors.

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

package patch

import "github.com/kubeedge/kubeedge/cloud/pkg/apis/devices/v1alpha3"

// Option is some configuration that modifies options for a patch request.
type Option interface {
	// ApplyToHelper applies this configuration to the given Helper options.
	ApplyToHelper(*HelperOptions)
}

// HelperOptions contains options for patch options.
type HelperOptions struct {
	// IncludeStatusObservedGeneration sets the status.observedGeneration field
	// on the incoming object to match metadata.generation, only if there is a change.
	IncludeStatusObservedGeneration bool

	// ForceOverwriteConditions allows the patch helper to overwrite conditions in case of conflicts.
	// This option should only ever be set in controller managing the object being patched.
	ForceOverwriteConditions bool

	// OwnedConditions defines condition types owned by the controller.
	// In case of conflicts for the owned conditions, the patch helper will always use the value provided by the controller.
	OwnedConditions []v1alpha3.ConditionType
}

// WithForceOverwriteConditions allows the patch helper to overwrite conditions in case of conflicts.
// This option should only ever be set in controller managing the object being patched.
type WithForceOverwriteConditions struct{}

// ApplyToHelper applies this configuration to the given HelperOptions.
func (w WithForceOverwriteConditions) ApplyToHelper(in *HelperOptions) {
	in.ForceOverwriteConditions = true
}

// WithStatusObservedGeneration sets the status.observedGeneration field
// on the incoming object to match metadata.generation, only if there is a change.
type WithStatusObservedGeneration struct{}

// ApplyToHelper applies this configuration to the given HelperOptions.
func (w WithStatusObservedGeneration) ApplyToHelper(in *HelperOptions) {
	in.IncludeStatusObservedGeneration = true
}

// WithOwnedConditions allows to define condition types owned by the controller.
// In case of conflicts for the owned conditions, the patch helper will always use the value provided by the controller.
type WithOwnedConditions struct {
	Conditions []v1alpha3.ConditionType
}

// ApplyToHelper applies this configuration to the given HelperOptions.
func (w WithOwnedConditions) ApplyToHelper(in *HelperOptions) {
	in.OwnedConditions = w.Conditions
}
