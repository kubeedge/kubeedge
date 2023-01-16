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

package manager

// Filter filter the event before enqueuing the key
type EventFilter interface {
	// Create returns true if the Create event should be processed
	Create(obj interface{}) bool

	// Delete returns true if the Delete event should be processed
	Delete(obj interface{}) bool

	// Update returns true if the Update event should be processed
	Update(oldObj, newObj interface{}) bool
}
