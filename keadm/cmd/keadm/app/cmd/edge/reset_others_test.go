//go:build !windows

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

package edge

import (
	"errors"
	"testing"

	"github.com/agiledragon/gomonkey/v2"
	"github.com/stretchr/testify/assert"

	"github.com/kubeedge/kubeedge/keadm/cmd/keadm/app/cmd/util"
	"github.com/kubeedge/kubeedge/keadm/cmd/keadm/app/cmd/util/extsystem"
)

type noServiceExtSystem struct {
	extsystem.ExtSystem
}

func (noServiceExtSystem) ServiceExists(string) bool {
	return false
}

func TestTearDownEdgeCore(t *testing.T) {
	var (
		extSystem extsystem.ExtSystem
		extErr    error
		running   bool
		killed    bool
	)
	patches := gomonkey.NewPatches()
	defer patches.Reset()
	patches.ApplyFunc(extsystem.GetExtSystem, func() (extsystem.ExtSystem, error) {
		return extSystem, extErr
	})
	patches.ApplyFunc(util.IsKubeEdgeProcessRunning, func(string) (bool, error) {
		return running, nil
	})
	patches.ApplyFunc(util.KillKubeEdgeBinary, func(string) error {
		killed = true
		return nil
	})

	noInitSystem := errors.New("no supported init system detected")
	cases := []struct {
		name      string
		extSystem extsystem.ExtSystem
		extErr    error
		running   bool
	}{
		{name: "no init system, edgecore running", extErr: noInitSystem, running: true},
		{name: "no init system, edgecore not running", extErr: noInitSystem},
		{name: "no edgecore service, edgecore running", extSystem: noServiceExtSystem{}, running: true},
		{name: "no edgecore service, edgecore not running", extSystem: noServiceExtSystem{}},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			extSystem, extErr, running, killed = c.extSystem, c.extErr, c.running, false
			assert.NoError(t, TearDownEdgeCore())
			assert.Equal(t, c.running, killed)
		})
	}
}
