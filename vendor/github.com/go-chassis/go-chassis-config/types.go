/*
 * Copyright 2017 Huawei Technologies Co., Ltd
 *
 * Licensed under the Apache License, Version 2.0 (the "License");
 * you may not use this file except in compliance with the License.
 * You may obtain a copy of the License at
 *
 *    http://www.apache.org/licenses/LICENSE-2.0
 *
 * Unless required by applicable law or agreed to in writing, software
 * distributed under the License is distributed on an "AS IS" BASIS,
 * WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 * See the License for the specific language governing permissions and
 * limitations under the License.
 */

// Package client created on 2017/6/22.
package config

// error response constants
const (
	LoggerInitFailed            = "logger initialization failed"
	PackageInitError            = "package not initialize successfully"
	ConfigServerMemRefreshError = "error in poulating config server member"
	EmptyConfigServerMembers    = "empty config server member"
	EmptyConfigServerConfig     = "empty config sevrer passed"
	RefreshModeError            = "refreshMode must be 0 or 1."
)
