/*
Copyright 2024 The KubeEdge Authors.

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

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"k8s.io/apimachinery/pkg/watch"
)

func TestEvents(t *testing.T) {
	assert := assert.New(t)

	events := make(chan watch.Event, 10)
	taskCache := &TaskCache{
		events: events,
	}

	returnedEvents := taskCache.Events()
	assert.Equal(events, returnedEvents)
}
