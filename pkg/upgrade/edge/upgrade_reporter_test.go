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
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestJSONFileReporter(t *testing.T) {
	srcJSONFile := upgradeReportJSONFile
	upgradeReportJSONFile = filepath.Join(t.TempDir(), "test.json")
	t.Cleanup(func() {
		upgradeReportJSONFile = srcJSONFile
	})

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

// TestJSONReporterInfoExists covers the JSONReporterInfoExists() function
func TestJSONReporterInfoExists(t *testing.T) {
	srcJSONFile := upgradeReportJSONFile
	upgradeReportJSONFile = filepath.Join(t.TempDir(), "test_exists.json")
	t.Cleanup(func() {
		upgradeReportJSONFile = srcJSONFile
	})

	// File does not exist yet
	assert.False(t, JSONReporterInfoExists())

	// Create the file
	err := NewJSONFileReporter(EventTypeRollback, "v1.21.0", "v1.20.0").Report(nil)
	assert.NoError(t, err)

	// File now exists
	assert.True(t, JSONReporterInfoExists())

	// Cleanup
	err = RemoveJSONReporterInfo()
	assert.NoError(t, err)
}

// TestParseJSONReporterInfo_FileNotFound covers the os.ReadFile error branch
func TestParseJSONReporterInfo_FileNotFound(t *testing.T) {
	srcJSONFile := upgradeReportJSONFile
	upgradeReportJSONFile = "/nonexistent/path/upgrade_report.json"
	t.Cleanup(func() {
		upgradeReportJSONFile = srcJSONFile
	})

	info, err := ParseJSONReporterInfo()
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "failed to read upgrade result from file")
	assert.Empty(t, info.EventType)
}

// TestParseJSONReporterInfo_InvalidJSON covers the json.Unmarshal error branch
func TestParseJSONReporterInfo_InvalidJSON(t *testing.T) {
	srcJSONFile := upgradeReportJSONFile
	upgradeReportJSONFile = filepath.Join(t.TempDir(), "test_invalid.json")
	t.Cleanup(func() {
		os.Remove(upgradeReportJSONFile)
		upgradeReportJSONFile = srcJSONFile
	})

	// Write invalid JSON to the file
	err := os.WriteFile(upgradeReportJSONFile, []byte("not valid json {{{"), os.ModePerm)
	assert.NoError(t, err)

	info, err := ParseJSONReporterInfo()
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "failed to unmarshal upgrade result")
	assert.Empty(t, info.EventType)
}

// TestRemoveJSONReporterInfo_FileNotFound covers the os.Remove error branch
func TestRemoveJSONReporterInfo_FileNotFound(t *testing.T) {
	srcJSONFile := upgradeReportJSONFile
	upgradeReportJSONFile = "nonexistent_file.json"
	t.Cleanup(func() {
		upgradeReportJSONFile = srcJSONFile
	})

	err := RemoveJSONReporterInfo()
	assert.Error(t, err)
}

// TestReport_WriteFileError covers the os.WriteFile error branch in Report()
func TestReport_WriteFileError(t *testing.T) {
	srcJSONFile := upgradeReportJSONFile
	// Use a path with a non-existent directory to force WriteFile to fail
	upgradeReportJSONFile = "/nonexistent/directory/upgrade_report.json"
	t.Cleanup(func() {
		upgradeReportJSONFile = srcJSONFile
	})

	err := NewJSONFileReporter(EventTypeConfigUpdate, "v1.20.0", "v1.21.0").Report(nil)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "failed to write upgrade result to file")
}

func TestReport_MarshalError(t *testing.T) {
	origMarshal := jsonMarshal
	jsonMarshal = func(v any) ([]byte, error) {
		return nil, errors.New("forced marshal error")
	}
	t.Cleanup(func() { jsonMarshal = origMarshal })

	err := NewJSONFileReporter(EventTypeUpgrade, "v1.20.0", "v1.21.0").Report(nil)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "failed to marshal upgrade result")
}
