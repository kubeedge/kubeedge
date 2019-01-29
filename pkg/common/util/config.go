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
	"io/ioutil"
	"os"
	"path"
	"path/filepath"
	"strings"

	"github.com/kubeedge/kubeedge/beehive/pkg/common/config"

	"github.com/ServiceComb/go-archaius/sources/file-source"
	"gopkg.in/yaml.v2"
)

// LoadConfig is function to Load Configarations from a specified location. If no location is specified it loads the config from the default location
func LoadConfig(confLocation ...string) error {
	err := config.CONFIG.DeInit()
	if err != nil {
		return err
	}
	fSource := filesource.NewYamlConfigurationSource()
	if len(confLocation) == 0 {
		confLocation = []string{os.Getenv("GOPATH") + "/src/github.com/kubeedge/kubeedge/conf"}
	}
	err = filepath.Walk(confLocation[0], func(location string, f os.FileInfo, err error) error {
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

//GenerateTestYaml is a function is used to create a temporary file to be used for testing
//It accepts 3 arguments:"test" is the interface used to generate the YAML,
// "path" is the directory path at which the directory is to be created,
// "filename" is the name of the file to be creatd without the ".yaml" extension
func GenerateTestYaml(test interface{}, path, filename string) error {
	data, err := yaml.Marshal(test)
	if err != nil {
		return err
	}
	err = os.MkdirAll(path, 0777)
	if err != nil {
		return err
	}
	err = ioutil.WriteFile(path+"/"+filename+".yaml", data, 0777)
	if err != nil {
		return err
	}
	return nil
}
