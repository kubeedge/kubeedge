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

package rule

//Key string type for rules exporting
type Key string

//constants for different rules key
const (
	EventType     Key = "event_type"
	MessageFilter Key = "message_filter"
	FunctionUrn   Key = "function_urn"
	TargetAddress Key = "target_address"
)

//Rule defines map of rules
type Rule struct {
	Name string         `json:"name,omitempty"`
	Data map[Key]string `json:"data,omitempty"`
}
