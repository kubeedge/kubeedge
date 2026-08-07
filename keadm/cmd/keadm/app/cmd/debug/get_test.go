/*
Copyright 2026 The KubeEdge Authors.

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

package debug

import (
	"testing"

	"github.com/agiledragon/gomonkey/v2"
	"github.com/stretchr/testify/assert"

	edgecoreCfg "github.com/kubeedge/api/apis/componentconfig/edgecore/v1alpha2"
)

func TestValidateUsesDefaultDataPathWhenNotSpecified(t *testing.T) {
	getOptions := NewGetOptions()
	getOptions.DataPath = ""

	patches := gomonkey.NewPatches()
	defer patches.Reset()

	patches.ApplyFunc(isFileExist, func(path string) bool {
		return path == edgecoreCfg.DataBaseDataSource
	})
	patches.ApplyFunc(InitDB, func(driverName, dbName, dataSource string) error {
		assert.Equal(t, edgecoreCfg.DataBaseDataSource, dataSource)
		return nil
	})

	err := getOptions.Validate([]string{"pod"})

	assert.NoError(t, err)
	assert.Equal(t, edgecoreCfg.DataBaseDataSource, getOptions.DataPath)
}
