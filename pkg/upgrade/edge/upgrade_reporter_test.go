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

package edge

import (
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestJSONFileReporter(t *testing.T) {
	srcJSONFile := upgradeReportJSONFile
	upgradeReportJSONFile = "test.json"
	defer func() {
		upgradeReportJSONFile = srcJSONFile
	}()

	t.Run("report upgrade successful", func(t *testing.T) {
		var err error
		err = NewJSONFileReporter(EventTypeBackup, "v1.20.0", "").
			Report(nil)
		assert.NoError(t, err)
		defer func() {
			err = RemoveJSONReporterInfo()
			assert.NoError(t, err)
		}()
		info, err := ParseJSONReporterInfo()
		assert.NoError(t, err)
		assert.Equal(t, EventTypeBackup, info.EventType)
		assert.True(t, info.Success)
		assert.Empty(t, info.ErrorMessage)
		assert.Equal(t, "v1.20.0", info.FromVersion)
	})

	t.Run("report upgrade failed", func(t *testing.T) {
		var err error
		err = NewJSONFileReporter(EventTypeUpgrade, "v1.20.0", "v1.21.0").
			Report(errors.New("test error"))
		assert.NoError(t, err)
		defer func() {
			err = RemoveJSONReporterInfo()
			assert.NoError(t, err)
		}()
		info, err := ParseJSONReporterInfo()
		assert.NoError(t, err)
		assert.Equal(t, EventTypeUpgrade, info.EventType)
		assert.False(t, info.Success)
		assert.Equal(t, "test error", info.ErrorMessage)
		assert.Equal(t, "v1.20.0", info.FromVersion)
		assert.Equal(t, "v1.21.0", info.ToVersion)
	})
}
