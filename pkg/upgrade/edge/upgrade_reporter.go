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
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"

	"github.com/kubeedge/api/apis/common/constants"
)

type Reporter interface {
	// Report reports the result of the upgrade related commands.
	// If the error is nil, it means the command was executed successfully.
	Report(err error) error
}

const (
	EventTypeBackup       = "Backup"
	EventTypeUpgrade      = "Upgrade"
	EventTypeRollback     = "Rollback"
	EventTypeConfigUpdate = "ConfigUpdate"
)

var upgradeReportJSONFile = filepath.Join(constants.KubeEdgePath, "upgrade_report.json")

// JSONReporterInfo defines the information to be reported.
type JSONReporterInfo struct {
	// EventType indicates a event of upgrade(Backup, Upgrade, or Rollback).
	EventType string `json:"event"`
	// Success indicates whether the upgrade event was successful.
	Success bool `json:"success"`
	// FromVersion indicates the version of the edgecore before the upgrade/rollback/backup.
	FromVersion string `json:"fromVersion"`
	// ToVersion indicates the version of the edgecore after the upgrade/rollback.
	ToVersion string `json:"toVersion"`
	// ErrorMessage uses to save error messages when an upgrade event fails.
	ErrorMessage string `json:"errorMessage,omitempty"`
}

// JSONFileReporter reports the result of the upgrade related commands to a JSON file.
type JSONFileReporter struct {
	EventType   string
	FromVersion string
	ToVersion   string
}

// NewJSONFileReporter creates a new JSONFileReporter.
func NewJSONFileReporter(eventType, fromVer, toVer string) Reporter {
	return &JSONFileReporter{
		EventType:   eventType,
		FromVersion: fromVer,
		ToVersion:   toVer,
	}
}

func (r JSONFileReporter) Report(inerr error) error {
	info := JSONReporterInfo{
		EventType:   r.EventType,
		Success:     inerr == nil,
		ToVersion:   r.ToVersion,
		FromVersion: r.FromVersion,
	}
	if inerr != nil {
		info.ErrorMessage = inerr.Error()
	}
	bff, err := json.Marshal(&info)
	if err != nil {
		return fmt.Errorf("failed to marshal upgrade result, err: %v", err)
	}
	if err := os.WriteFile(upgradeReportJSONFile, bff, os.ModePerm); err != nil {
		return fmt.Errorf("failed to write upgrade result to file %s, err: %v",
			upgradeReportJSONFile, err)
	}
	return nil
}

// ParseJSONReporterInfo parses the result of the upgrade related commands from a JSON file.
func ParseJSONReporterInfo() (JSONReporterInfo, error) {
	var info JSONReporterInfo
	bff, err := os.ReadFile(upgradeReportJSONFile)
	if err != nil {
		return info, fmt.Errorf("failed to read upgrade result from file %s, err: %v",
			upgradeReportJSONFile, err)
	}

	if err := json.Unmarshal(bff, &info); err != nil {
		return info, fmt.Errorf("failed to unmarshal upgrade result, err: %v", err)
	}
	return info, nil
}

func RemoveJSONReporterInfo() error {
	return os.Remove(upgradeReportJSONFile)
}
