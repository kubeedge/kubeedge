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

package csidriver

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestNewUnixDomainSocket(t *testing.T) {
	assert := assert.New(t)

	// Using default buffer size
	us := NewUnixDomainSocket("/tmp/test.sock")
	assert.NotNil(us)
	assert.Equal("/tmp/test.sock", us.filename)
	assert.Equal(DefaultBufferSize, us.buffersize)

	// Using custom buffer size
	us = NewUnixDomainSocket("/tmp/test.sock", 2048)
	assert.NotNil(us)
	assert.Equal("/tmp/test.sock", us.filename)
	assert.Equal(2048, us.buffersize)
}
