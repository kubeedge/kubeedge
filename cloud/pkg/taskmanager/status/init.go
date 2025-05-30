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

package status

import (
	"context"
)

var (
	imagePrePullJobStatusUpdater *StatusUpdater

	nodeUpgradeJobStatusUpdater *StatusUpdater

	configUpdateJobStatusUpdater *StatusUpdater
)

func Init(ctx context.Context) {
	imagePrePullJobStatusUpdater = NewStatusUpdater(ctx, tryUpdateImagePrePullJobStatus)
	go imagePrePullJobStatusUpdater.WatchUpdateChannel()

	nodeUpgradeJobStatusUpdater = NewStatusUpdater(ctx, tryUpdateNodeUpgradeJobStatus)
	go nodeUpgradeJobStatusUpdater.WatchUpdateChannel()

	configUpdateJobStatusUpdater = NewStatusUpdater(ctx, tryUpdateConfigUpdateJobStatus)
	go configUpdateJobStatusUpdater.WatchUpdateChannel()
}

func GetImagePrePullJobStatusUpdater() *StatusUpdater {
	return imagePrePullJobStatusUpdater
}

func GetNodeUpgradeJobStatusUpdater() *StatusUpdater {
	return nodeUpgradeJobStatusUpdater
}

func GetConfigeUpdateJobStatusUpdater() *StatusUpdater {
	return configUpdateJobStatusUpdater
}
