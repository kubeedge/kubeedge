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

package taskexecutor

import (
	"errors"
	"reflect"
	"testing"

	"github.com/agiledragon/gomonkey/v2"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	corev1 "k8s.io/api/core/v1"

	metaclient "github.com/kubeedge/kubeedge/edge/pkg/metamanager/client"
)

func TestMakePullSecretAuth(t *testing.T) {
	patches := gomonkey.NewPatches()
	defer patches.Reset()

	var getErr error
	patches.ApplyMethodFunc(reflect.TypeOf(metaclient.New().Secrets("default")), "Get",
		func(name string) (*corev1.Secret, error) {
			if getErr != nil {
				return nil, getErr
			}
			return &corev1.Secret{
				Type: corev1.SecretTypeDockerConfigJson,
				Data: map[string][]byte{corev1.DockerConfigJsonKey: []byte(
					`{"auths":{"registry.example.com":{"username":"user","password":"pass"}}}`)},
			}, nil
		})

	t.Run("no pull secret", func(t *testing.T) {
		auth, err := makePullSecretAuth("")
		require.NoError(t, err)
		assert.Nil(t, auth.ForImage("registry.example.com/team/app:v1"))
	})

	t.Run("pull secret format is not correct", func(t *testing.T) {
		_, err := makePullSecretAuth("no-namespace")
		require.EqualError(t, err, "pull secret format is not correct")
	})

	t.Run("get secret failed", func(t *testing.T) {
		getErr = errors.New("test error")
		defer func() { getErr = nil }()
		_, err := makePullSecretAuth("default/regcred")
		require.ErrorContains(t, err, "get secret regcred failed")
	})

	t.Run("credentials are used only for the matching registry", func(t *testing.T) {
		auth, err := makePullSecretAuth("default/regcred")
		require.NoError(t, err)
		creds := auth.ForImage("registry.example.com/team/app:v1")
		require.NotNil(t, creds)
		assert.Equal(t, "user", creds.Username)
		assert.Equal(t, "pass", creds.Password)
		assert.Nil(t, auth.ForImage("nginx:latest"))
	})
}
