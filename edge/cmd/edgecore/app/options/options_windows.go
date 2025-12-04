//go:build windows

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

package options

import (
	"path/filepath"

	"github.com/spf13/pflag"

	"github.com/kubeedge/api/apis/common/constants"
)

type OSExclusive struct {
	// LogFilePath is the path to the log file
	LogFilePath string
}

func osExclusiveFlags(fs *pflag.FlagSet, opts *EdgeCoreOptions) {
	fs.StringVar(&opts.LogFilePath, "log-file", filepath.Join(constants.KubeEdgeLogPath, "edgecore.log"),
		"Use this key to set the log file path for edgecore")
}
