/*
Copyright 2023 The KubeEdge Authors.

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

package util

import (
	"fmt"
	"strings"

	"github.com/distribution/distribution/v3/reference"
	metav1 "k8s.io/api/core/v1"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	versionutil "k8s.io/apimachinery/pkg/util/version"
	"k8s.io/klog/v2"

	"github.com/kubeedge/api/apis/operations/v1alpha1"
	"github.com/kubeedge/kubeedge/common/constants"
)

const (
	NodeUpgradeJobStatusKey   = "nodeupgradejob.operations.kubeedge.io/status"
	NodeUpgradeJobStatusValue = ""
	NodeUpgradeHistoryKey     = "nodeupgradejob.operations.kubeedge.io/history"
)

const (
	TaskUpgrade  = "upgrade"
	TaskRollback = "rollback"
	TaskBackup   = "backup"
	TaskPrePull  = "prepull"
)

type TaskMessage struct {
	Type            string
	Name            string
	TimeOutSeconds  *uint32
	ShutDown        bool
	CheckItem       []string
	Concurrency     int32
	FailureTolerate float64
	NodeNames       []string
	LabelSelector   *v1.LabelSelector
	Status          v1alpha1.TaskStatus
	Msg             interface{}
}

// FilterVersion returns true only if the edge node version already on the upgrade req
// version is like: v1.22.6-kubeedge-v1.10.0-beta.0.185+95378fb019912a, expected is like v1.10.0
func FilterVersion(version string, expected string) bool {
	// if not correct version format, also return true
	strs := strings.Split(version, "-")
	if len(strs) < 3 {
		klog.Warningf("version format should be {k8s version}-kubeedge-{edgecore version}, but got : %s", version)
		return true
	}

	// filter nodes that already in the required version
	less, err := VersionLess(strs[2], expected)
	if err != nil {
		klog.Warningf("version filter failed: %s", err.Error())
		less = false
	}
	return !less
}

// IsEdgeNode checks whether a node is an Edge Node
// only if label {"node-role.kubernetes.io/edge": ""} exists, it is an edge node
func IsEdgeNode(node *metav1.Node) bool {
	if node.Labels == nil {
		return false
	}
	if _, ok := node.Labels[constants.EdgeNodeRoleKey]; !ok {
		return false
	}

	if node.Labels[constants.EdgeNodeRoleKey] != constants.EdgeNodeRoleValue {
		return false
	}

	return true
}

// RemoveDuplicateElement deduplicate
func RemoveDuplicateElement(s []string) []string {
	result := make([]string, 0, len(s))
	temp := make(map[string]struct{}, len(s))

	for _, item := range s {
		if _, ok := temp[item]; !ok {
			temp[item] = struct{}{}
			result = append(result, item)
		}
	}

	return result
}

// MergeAnnotationUpgradeHistory constructs the new history based on the origin history
// and we'll only keep 3 records
func MergeAnnotationUpgradeHistory(origin, fromVersion, toVersion string) string {
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

func GetNodeName(resource string) string {
	// task/${TaskID}/node/${NodeID}
	s := strings.Split(resource, "/")
	return s[3]
}
func GetTaskID(resource string) string {
	// task/${TaskID}/node/${NodeID}
	s := strings.Split(resource, "/")
	return s[1]
}

func VersionLess(version1, version2 string) (bool, error) {
	less := false
	ver1, err := versionutil.ParseGeneric(version1)
	if err != nil {
		return less, fmt.Errorf("version1 error: %v", err)
	}
	ver2, err := versionutil.ParseGeneric(version2)
	if err != nil {
		return less, fmt.Errorf("version2 error: %v", err)
	}
	// If the remote Major version is bigger or if the Major versions are the same,
	// but the remote Minor is bigger use the client version release. This handles Major bumps too.
	if ver1.Major() < ver2.Major() ||
		(ver1.Major() == ver2.Major()) && ver1.Minor() < ver2.Minor() ||
		(ver1.Major() == ver2.Major() && ver1.Minor() == ver2.Minor()) && ver1.Patch() < ver2.Patch() {
		less = true
	}
	return less, nil
}

func NodeUpdated(old, new v1alpha1.TaskStatus) bool {
	if old.NodeName != new.NodeName {
		klog.V(4).Infof("old node %s and new node %s is not same", old.NodeName, new.NodeName)
		return false
	}
	if old.State == new.State || new.State == "" {
		klog.V(4).Infof("node %s state is not change", old.NodeName)
		return false
	}
	return true
}
