/*
Copyright 2024 The KubeEdge Authors.

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

package cloud

import (
	"testing"

	"github.com/spf13/cobra"
	"github.com/stretchr/testify/assert"

	"github.com/kubeedge/kubeedge/keadm/cmd/keadm/app/cmd/common"
)

func TestNewGetToken(t *testing.T) {
	assert := assert.New(t)

	cmd := NewGettoken()

	assert.NotNil(cmd)
	assert.Equal(cmd.Use, "gettoken")
	assert.Equal(cmd.Short, "To get the token for edge nodes to join the cluster")
	assert.Equal(cmd.Long, gettokenLongDescription)
	assert.Equal(cmd.Example, gettokenExample)

	assert.NotNil(cmd.RunE)

	flag := cmd.Flags().Lookup(common.FlagNameKubeConfig)
	assert.NotNil(flag)
	assert.Equal(common.DefaultKubeConfig, flag.DefValue)
	assert.Equal(common.FlagNameKubeConfig, flag.Name)
}

func TestAddGettokenFlags(t *testing.T) {
	assert := assert.New(t)

	cmd := &cobra.Command{}
	gettokenOptions := newGettokenOptions()

	addGettokenFlags(cmd, gettokenOptions)

	flag := cmd.Flags().Lookup(common.FlagNameKubeConfig)
	assert.NotNil(flag)
	assert.Equal(common.DefaultKubeConfig, flag.DefValue)
	assert.Equal(common.FlagNameKubeConfig, flag.Name)
}

func TestNewGettokenOptions(t *testing.T) {
	assert := assert.New(t)

	opts := newGettokenOptions()

	assert.NotNil(opts)
	assert.Equal(common.DefaultKubeConfig, opts.Kubeconfig)
}

func TestShowToken(t *testing.T) {
	cases := []struct {
		data    []byte
		wantErr bool
	}{
		{
			data:    []byte("valid token"),
			wantErr: false,
		},
		{
			data:    []byte(""),
			wantErr: false,
		},
	}

	assert := assert.New(t)

	for _, test := range cases {
		t.Run("Testing showToken()", func(t *testing.T) {
			err := showToken(test.data)
			if !test.wantErr {
				assert.NoError(err)
			}
		})
	}
}
