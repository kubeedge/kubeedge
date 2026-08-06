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

package edge

import (
	"errors"
	"os"
	"path/filepath"
	"testing"

	"github.com/agiledragon/gomonkey/v2"
	"github.com/stretchr/testify/assert"
	"sigs.k8s.io/yaml"

	"github.com/kubeedge/api/apis/componentconfig/edgecore/v1alpha2"
	"github.com/kubeedge/kubeedge/keadm/cmd/keadm/app/cmd/util"
	"github.com/kubeedge/kubeedge/pkg/util/execs"
)

func TestNewEdgeConfigUpdate(t *testing.T) {
	cmd := NewEdgeConfigUpdate()

	assert.NotNil(t, cmd)
	assert.Equal(t, "config-update", cmd.Use)
	assert.Equal(t, "Update EdgeCore Configuration.", cmd.Short)
	assert.Equal(t, "Update EdgeCore Configuration.", cmd.Long)
	assert.NotNil(t, cmd.RunE)
	assert.True(t, len(cmd.Flags().Lookup("config").DefValue) > 0)
	assert.NotNil(t, cmd.Flags().Lookup("set"))
}

func TestNewConfigUpdateExecutor(t *testing.T) {
	executor := newConfigUpdateExecutor()
	assert.NotNil(t, executor)
}

func TestConfigUpdateExecutor_ConfigUpdate(t *testing.T) {
	t.Run("file read error", func(t *testing.T) {
		executor := &configUpdateExecutor{}
		opts := ConfigUpdateOptions{
			BaseOptions: BaseOptions{
				Config: "/nonexistent/config.yaml",
			},
		}

		err := executor.configUpdate(opts)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "failed to read configfile")
	})

	t.Run("yaml unmarshal error", func(t *testing.T) {
		tmpDir := t.TempDir()
		configFile := filepath.Join(tmpDir, "config.yaml")
		err := os.WriteFile(configFile, []byte("invalid: yaml: content: ["), 0644)
		assert.NoError(t, err)

		executor := &configUpdateExecutor{}
		opts := ConfigUpdateOptions{
			BaseOptions: BaseOptions{
				Config: configFile,
			},
		}

		err = executor.configUpdate(opts)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "failed to unmarshal configfile")
	})

	t.Run("parse sets error", func(t *testing.T) {
		patches := gomonkey.NewPatches()
		defer patches.Reset()

		tmpDir := t.TempDir()
		configFile := filepath.Join(tmpDir, "config.yaml")
		config := &v1alpha2.EdgeCoreConfig{}
		configData, err := yaml.Marshal(config)
		assert.NoError(t, err)
		err = os.WriteFile(configFile, configData, 0644)
		assert.NoError(t, err)

		patches.ApplyFunc(util.ParseSet, func(config interface{}, sets string) error {
			return errors.New("parse sets failed")
		})

		executor := &configUpdateExecutor{}
		opts := ConfigUpdateOptions{
			BaseOptions: BaseOptions{
				Config: configFile,
			},
			Sets: "invalid=set",
		}

		err = executor.configUpdate(opts)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "failed to parse sets value to config file")
	})

	t.Run("write config error", func(t *testing.T) {
		patches := gomonkey.NewPatches()
		defer patches.Reset()

		tmpDir := t.TempDir()
		configFile := filepath.Join(tmpDir, "config.yaml")
		config := &v1alpha2.EdgeCoreConfig{}
		configData, err := yaml.Marshal(config)
		assert.NoError(t, err)
		err = os.WriteFile(configFile, configData, 0644)
		assert.NoError(t, err)

		patches.ApplyFunc(util.ParseSet, func(config interface{}, sets string) error {
			return nil
		})
		patches.ApplyMethodFunc(&v1alpha2.EdgeCoreConfig{}, "WriteTo", func(path string) error {
			return errors.New("write failed")
		})

		executor := &configUpdateExecutor{}
		opts := ConfigUpdateOptions{
			BaseOptions: BaseOptions{
				Config: configFile,
			},
			Sets: "key=value",
		}

		err = executor.configUpdate(opts)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "failed to write new edgecore config")
	})

	t.Run("systemctl restart error", func(t *testing.T) {
		patches := gomonkey.NewPatches()
		defer patches.Reset()

		tmpDir := t.TempDir()
		configFile := filepath.Join(tmpDir, "config.yaml")
		config := &v1alpha2.EdgeCoreConfig{}
		configData, err := yaml.Marshal(config)
		assert.NoError(t, err)
		err = os.WriteFile(configFile, configData, 0644)
		assert.NoError(t, err)

		patches.ApplyFunc(util.ParseSet, func(config interface{}, sets string) error {
			return nil
		})
		patches.ApplyMethodFunc(&v1alpha2.EdgeCoreConfig{}, "WriteTo", func(path string) error {
			return nil
		})
		patches.ApplyFunc(execs.NewCommand, func(cmd string) *execs.Command {
			return &execs.Command{}
		})
		patches.ApplyMethodFunc(&execs.Command{}, "Exec", func() error {
			return errors.New("systemctl failed")
		})

		executor := &configUpdateExecutor{}
		opts := ConfigUpdateOptions{
			BaseOptions: BaseOptions{
				Config: configFile,
			},
			Sets: "key=value",
		}

		err = executor.configUpdate(opts)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "failed restart edgecore")
	})

	t.Run("successful config update", func(t *testing.T) {
		patches := gomonkey.NewPatches()
		defer patches.Reset()

		tmpDir := t.TempDir()
		configFile := filepath.Join(tmpDir, "config.yaml")
		config := &v1alpha2.EdgeCoreConfig{}
		configData, err := yaml.Marshal(config)
		assert.NoError(t, err)
		err = os.WriteFile(configFile, configData, 0644)
		assert.NoError(t, err)

		patches.ApplyFunc(util.ParseSet, func(config interface{}, sets string) error {
			return nil
		})
		patches.ApplyMethodFunc(&v1alpha2.EdgeCoreConfig{}, "WriteTo", func(path string) error {
			return nil
		})

		var execCalled bool
		patches.ApplyFunc(execs.NewCommand, func(cmd string) *execs.Command {
			assert.Equal(t, "sudo systemctl restart edgecore.service", cmd)
			return &execs.Command{}
		})
		patches.ApplyMethodFunc(&execs.Command{}, "Exec", func() error {
			execCalled = true
			return nil
		})

		executor := &configUpdateExecutor{}
		opts := ConfigUpdateOptions{
			BaseOptions: BaseOptions{
				Config: configFile,
			},
			Sets: "modules.edged.containerRuntime=docker",
		}

		err = executor.configUpdate(opts)
		assert.NoError(t, err)
		assert.True(t, execCalled)
	})
}
