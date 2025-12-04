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

package app

import (
	"flag"
	"fmt"

	klog "k8s.io/klog/v2"

	"github.com/kubeedge/kubeedge/edge/cmd/edgecore/app/options"
)

func initForOS(opts *options.EdgeCoreOptions) error {
	flagset := flag.NewFlagSet("log", flag.ExitOnError)
	klog.InitFlags(flagset)
	args := map[string]string{
		"log_file":    opts.LogFilePath,
		"logtostderr": "false",
	}
	for k, v := range args {
		if err := flagset.Set(k, v); err != nil {
			return fmt.Errorf("set flag %s failed: %w", k, err)
		}
	}
	return nil
}
