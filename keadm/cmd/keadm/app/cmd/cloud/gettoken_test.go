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
	"errors"
	"testing"

	"github.com/agiledragon/gomonkey/v2"
	"github.com/spf13/cobra"
	"github.com/stretchr/testify/assert"
	"k8s.io/client-go/kubernetes"

	"github.com/kubeedge/kubeedge/common/constants"
	"github.com/kubeedge/kubeedge/keadm/cmd/keadm/app/cmd/common"
	"github.com/kubeedge/kubeedge/keadm/cmd/keadm/app/cmd/util"
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

func TestQueryToken(t *testing.T) {
	assert := assert.New(t)

	patches := gomonkey.NewPatches()
	defer patches.Reset()

	patches.ApplyFunc(util.KubeClient, func(kubeConfigPath string) (*kubernetes.Clientset, error) {
		return nil, errors.New("kube client error")
	})

	token, err := queryToken("namespace", "name", "kubeconfig")
	assert.Error(err)
	assert.Nil(token)
	assert.Contains(err.Error(), "kube client error")

	patches.Reset()

	mockClient := &kubernetes.Clientset{}
	patches.ApplyFunc(util.KubeClient, func(kubeConfigPath string) (*kubernetes.Clientset, error) {
		return mockClient, nil
	})

	patches.ApplyFunc(queryToken, func(namespace, name, kubeConfigPath string) ([]byte, error) {
		if name == "nonexistent-secret" {
			return nil, errors.New("secret not found")
		}
		return []byte("test-token"), nil
	})

	token, err = queryToken("namespace", "nonexistent-secret", "kubeconfig")
	assert.Error(err)
	assert.Contains(err.Error(), "secret not found")

	token, err = queryToken("test-namespace", "test-secret", "kubeconfig")
	assert.NoError(err)
	assert.Equal([]byte("test-token"), token)
}

func TestGettokenRunE(t *testing.T) {
	assert := assert.New(t)

	patches := gomonkey.NewPatches()
	defer patches.Reset()

	patches.ApplyFunc(queryToken, func(namespace, name, kubeConfigPath string) ([]byte, error) {
		assert.Equal(constants.SystemNamespace, namespace)
		assert.Equal(common.TokenSecretName, name)
		assert.Equal(common.DefaultKubeConfig, kubeConfigPath)
		return nil, errors.New("query token error")
	})

	cmd := NewGettoken()
	err := cmd.RunE(cmd, []string{})
	assert.Error(err)
	assert.Contains(err.Error(), "query token error")

	patches.Reset()

	tokenData := []byte("mock token")
	patches.ApplyFunc(queryToken, func(namespace, name, kubeConfigPath string) ([]byte, error) {
		assert.Equal(constants.SystemNamespace, namespace)
		assert.Equal(common.TokenSecretName, name)
		assert.Equal(common.DefaultKubeConfig, kubeConfigPath)
		return tokenData, nil
	})

	showTokenCalled := false
	patches.ApplyFunc(showToken, func(data []byte) error {
		showTokenCalled = true
		assert.Equal(tokenData, data)
		return nil
	})

	cmd = NewGettoken()
	err = cmd.RunE(cmd, []string{})
	assert.NoError(err)
	assert.True(showTokenCalled)

	patches.Reset()

	patches.ApplyFunc(queryToken, func(namespace, name, kubeConfigPath string) ([]byte, error) {
		assert.Equal(constants.SystemNamespace, namespace)
		assert.Equal(common.TokenSecretName, name)
		assert.Equal("/custom/config", kubeConfigPath)
		return tokenData, nil
	})

	patches.ApplyFunc(showToken, func(data []byte) error {
		return nil
	})

	cmd = NewGettoken()
	err = cmd.Flags().Set(common.FlagNameKubeConfig, "/custom/config")
	assert.NoError(err)
	err = cmd.RunE(cmd, []string{})
	assert.NoError(err)
}
