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

package config

import (
	"sync"
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/kubeedge/api/apis/componentconfig/edgecore/v1alpha2"
)

func TestInitConfigure(t *testing.T) {
	// Save old package-level state
	oldConfig := Config
	oldOnce := once

	// Reset state for testing
	once = sync.Once{}
	Config = Configure{}

	// Restore state in deferred cleanup
	defer func() {
		Config = oldConfig
		once = oldOnce
	}()

	s := &v1alpha2.ServiceBus{
		Enable:  true,
		Server:  "127.0.0.1",
		Port:    9090,
		Timeout: 30,
	}

	InitConfigure(s)

	// Compare the whole ServiceBus struct to cover future fields automatically
	assert.Equal(t, *s, Config.ServiceBus)
}
