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

package mux

type HandlerFunc func(*MessageContainer, ResponseWriter)

type MessageMuxEntry struct {
	pattern    *MessagePattern
	handleFunc HandlerFunc
}

func NewEntry(pattern *MessagePattern, handle func(*MessageContainer, ResponseWriter)) *MessageMuxEntry {
	return &MessageMuxEntry{
		pattern:    pattern,
		handleFunc: handle,
	}
}

func (entry *MessageMuxEntry) Pattern(pattern *MessagePattern) *MessageMuxEntry {
	entry.pattern = pattern
	return entry
}

func (entry *MessageMuxEntry) Handle(handle func(*MessageContainer, ResponseWriter)) *MessageMuxEntry {
	entry.handleFunc = handle
	return entry
}
