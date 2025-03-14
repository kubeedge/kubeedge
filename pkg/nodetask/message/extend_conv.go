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

package message

import (
	"encoding/json"
	"fmt"
	"strings"

	operationsv1alpha2 "github.com/kubeedge/api/apis/operations/v1alpha2"
)

// FormatNodeUpgradeJobExtend formats the node upgrade job extend.
func FormatNodeUpgradeJobExtend(fromVer, toVer string) string {
	return strings.Join([]string{fromVer, toVer}, ",")
}

// ParseNodeUpgradeJobExtend parses the node upgrade job extend.
func ParseNodeUpgradeJobExtend(extend string) (fromVer, toVer string, err error) {
	parts := strings.Split(extend, ",")
	if len(parts) != 2 {
		return "", "", fmt.Errorf("invalid node upgrade job extend: %s", extend)
	}
	return parts[0], parts[1], nil
}

func FormatImagePrePullJobExtend(statusItems []operationsv1alpha2.ImageStatus) (string, error) {
	bff, err := json.Marshal(statusItems)
	if err != nil {
		return "", err
	}
	return string(bff), nil
}

func ParseImagePrePullJobExtend(extend string) ([]operationsv1alpha2.ImageStatus, error) {
	var statusItems []operationsv1alpha2.ImageStatus
	if err := json.Unmarshal([]byte(extend), &statusItems); err != nil {
		return nil, err
	}
	return statusItems, nil
}
