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

package version

import (
	"errors"
	"os"
	"path/filepath"

	"github.com/kubeedge/api/apis/common/constants"
)

const edgecoreVersionFile = "edgecore_version"

// WriteEdgeCoreVersion writes the edgecore version to the path relative to the edgecore.yaml.
func WriteEdgeCoreVersion(configPath string) error {
	return os.WriteFile(getEdgeCoreVersionFile(configPath), []byte(Get().String()), os.ModePerm)
}

// ReadEdgeCoreVersion reads the edgecore version from the path relative to the edgecore.yaml.
func ReadEdgeCoreVersion(configPath string) (string, error) {
	bff, err := os.ReadFile(getEdgeCoreVersionFile(configPath))
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return "", nil
		}
		return "", err
	}
	return string(bff), nil
}

// RemoveEdgeCoreVersion removes the edgecore version from the path relative to the edgecore.yaml.
func RemoveEdgeCoreVersion(configPath string) error {
	if err := os.Remove(getEdgeCoreVersionFile(configPath)); err != nil && !os.IsNotExist(err) {
		return err
	}
	return nil
}

// getEdgeCoreVersionFile returns the path of the edgecore version file.
// If the configPath is the default path, the edgecore version file is in the /etc/kubeedge.
// Otherwise, the edgecore version file is in the same directory as the configPath.
func getEdgeCoreVersionFile(configPath string) string {
	var versionFile string
	if configPath == constants.EdgecoreConfigPath {
		versionFile = filepath.Join(constants.KubeEdgePath, edgecoreVersionFile)
	} else {
		versionFile = filepath.Join(filepath.Dir(configPath), edgecoreVersionFile)
	}
	return versionFile
}
