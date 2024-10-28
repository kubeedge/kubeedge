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

package options

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestNewControllerManagerOptions(t *testing.T) {
	assert := assert.New(t)

	opt := NewControllerManagerOptions()
	assert.NotNil(opt)
	assert.False(opt.UseServerSideApply, "UseServerSideApply should be false by default")
}

func TestFlags(t *testing.T) {
	assert := assert.New(t)

	opt := NewControllerManagerOptions()
	fss := opt.Flags()
	fs := fss.FlagSet("ControllerManager")
	assert.NotNil(fs)

	flag := fs.Lookup("use-server-side-apply")
	assert.NotNil(flag)
	assert.Equal("use-server-side-apply", flag.Name)
	assert.Equal("false", flag.DefValue)
	assert.Equal("If use server-side apply when updating templates", flag.Usage)
}
