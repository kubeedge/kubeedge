/*
Copyright 2018 The KubeEdge Authors.

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
	"os"
	"path"
	"path/filepath"
	"strings"

	"github.com/ServiceComb/go-archaius/sources/file-source"

	"github.com/kubeedge/kubeedge/beehive/pkg/common/config"
)

// LoadConfig is function to Load Configarations
func LoadConfig() error {
	fSource := filesource.NewYamlConfigurationSource()
	confLocation := os.Getenv("GOPATH") + "/src/github.com/kubeedge/kubeedge/conf"
	err := filepath.Walk(confLocation, func(location string, f os.FileInfo, err error) error {
		if f == nil {
			return err
		}
		if f.IsDir() {
			return nil
		}
		ext := strings.ToLower(path.Ext(location))
		if ext == ".yml" || ext == ".yaml" {
			fSource.AddFileSource(location, 0)
		}
		return nil
	})
	if err != nil {
		return err
	}
	config.CONFIG.AddSource(fSource)
	return nil
}
