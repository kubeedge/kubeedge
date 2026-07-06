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

package taskexecutor

import (
	"testing"

	"github.com/stretchr/testify/require"

	commontypes "github.com/kubeedge/kubeedge/common/types"
)

func TestPrepareKeadmRequiresImageDigest(t *testing.T) {
	err := prepareKeadm(&commontypes.NodeUpgradeJobRequest{
		Version: "v1.21.0",
		Image:   "kubeedge/installation-package:v1.21.0",
	})

	require.EqualError(t, err, "imageDigest is required for node upgrade jobs")
}
