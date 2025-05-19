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

package commons

import "time"

// Constants for name-related values used by the logger.
const (
	LoggerFieldInstanceName = "name"

	LoggerNameNodeUpgradeJob   = "node-upgrade-job"
	LoggerNameImagePrePullJob  = "image-prepull-job"
	LoggerNameConfigeUpdateJob = "config-update-job"
	LoggerFieldNodeJobType     = "jobtype"
)

// Constants for the default values.
const (
	DefaultRequeueTime = 10 * time.Second

	DefaultNodeJobTimeout = 300 * time.Second
)
