/*
Copyright 2022 The KubeEdge Authors.

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

package controller

import (
	"fmt"
	"strings"
	"time"

	"github.com/distribution/distribution/v3/reference"

	"github.com/kubeedge/kubeedge/pkg/apis/operations/v1alpha1"
)

const (
	NodeUpgradeJobStatusKey   = "nodeupgradejob.operations.kubeedge.io/status"
	NodeUpgradeJobStatusValue = ""
	NodeUpgradeHistoryKey     = "nodeupgradejob.operations.kubeedge.io/history"
)

const (
	NodeUpgrade = "upgrade"

	ISO8601UTC = "2006-01-02T15:04:05Z"
)

// filterVersion returns true only if the edge node version already on the upgrade req
// version is like: v1.22.6-kubeedge-v1.10.0-beta.0.185+95378fb019912a, expected is like v1.10.0
func filterVersion(version string, expected string) bool {
	// if not correct version format, also return true
	index := strings.Index(version, "-kubeedge-")
	if index == -1 {
		return false
	}

	length := len("-kubeedge-")

	// filter nodes that already in the required version
	return version[index+length:] == expected
}

// isCompleted returns true only if some/all edge upgrade is upgrading or completed
func isCompleted(upgrade *v1alpha1.NodeUpgradeJob) bool {
	// all edge node upgrade is upgrading or completed
	if upgrade.Status.State != v1alpha1.InitialValue {
		return true
	}

	// partial edge node upgrade is upgrading or completed
	for _, status := range upgrade.Status.Status {
		if status.State != v1alpha1.InitialValue {
			return true
		}
	}

	return false
}

// UpdateNodeUpgradeJobStatus updates the status
// return the updated result
func UpdateNodeUpgradeJobStatus(old *v1alpha1.NodeUpgradeJob, status *v1alpha1.UpgradeStatus) *v1alpha1.NodeUpgradeJob {
	// return value upgrade cannot populate the input parameter old
	upgrade := old.DeepCopy()

	for index := range upgrade.Status.Status {
		// If Node's Upgrade info exist, just overwrite
		if upgrade.Status.Status[index].NodeName == status.NodeName {
			// The input status no upgradeTime, we need set it with old value
			status.History.UpgradeTime = upgrade.Status.Status[index].History.UpgradeTime
			upgrade.Status.Status[index] = *status
			return upgrade
		}
	}

	// if Node's Upgrade info not exist, just append
	if status.History.UpgradeTime == "" {
		// If upgrade time is blank, set to the current time
		status.History.UpgradeTime = time.Now().Format(ISO8601UTC)
	}
	upgrade.Status.Status = append(upgrade.Status.Status, *status)

	return upgrade
}

// mergeAnnotationUpgradeHistory constructs the new history based on the origin history
// and we'll only keep 3 records
func mergeAnnotationUpgradeHistory(origin, fromVersion, toVersion string) string {
	newHistory := fmt.Sprintf("%s->%s", fromVersion, toVersion)
	if origin == "" {
		return newHistory
	}

	sets := strings.Split(origin, ";")
	if len(sets) > 2 {
		sets = sets[1:]
	}

	sets = append(sets, newHistory)
	return strings.Join(sets, ";")
}

// GetImageRepo gets repo from a container image
func GetImageRepo(image string) (string, error) {
	named, err := reference.ParseNormalizedNamed(image)
	if err != nil {
		return "", fmt.Errorf("failed to parse image name: %v", err)
	}

	return named.Name(), nil
}
