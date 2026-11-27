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

package image

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	corev1 "k8s.io/api/core/v1"
)

func dockerConfigJSONSecret(config string) *corev1.Secret {
	return &corev1.Secret{
		Type: corev1.SecretTypeDockerConfigJson,
		Data: map[string][]byte{corev1.DockerConfigJsonKey: []byte(config)},
	}
}

func TestPullSecretAuthForImage(t *testing.T) {
	// The format written by kubectl create secret docker-registry.
	secret := dockerConfigJSONSecret(`{"auths":{
		"registry.example.com:5000":{"username":"alice","password":"secret"},
		"https://index.docker.io/v1/":{"username":"bob","password":"hub-secret"}}}`)
	auth, err := NewPullSecretAuth(secret)
	require.NoError(t, err)

	cases := []struct {
		image    string
		username string
		password string
	}{
		{image: "registry.example.com:5000/team/app:1.0", username: "alice", password: "secret"},
		{image: "registry.example.com:5000/team/app@sha256:" +
			"0123456789abcdef0123456789abcdef0123456789abcdef0123456789abcdef", username: "alice", password: "secret"},
		{image: "nginx:1.27", username: "bob", password: "hub-secret"},
		{image: "docker.io/library/busybox", username: "bob", password: "hub-secret"},
	}
	for _, c := range cases {
		t.Run(c.image, func(t *testing.T) {
			cfg := auth.ForImage(c.image)
			require.NotNil(t, cfg)
			assert.Equal(t, c.username, cfg.Username)
			assert.Equal(t, c.password, cfg.Password)
		})
	}

	assert.Nil(t, auth.ForImage("quay.io/team/app:1.0"), "no credentials for another registry")
}

func TestPullSecretAuthWithoutCredentials(t *testing.T) {
	opaque := &corev1.Secret{
		Type: corev1.SecretTypeOpaque,
		Data: map[string][]byte{corev1.DockerConfigJsonKey: []byte(`{"auths":{"registry.example.com":{"username":"alice"}}}`)},
	}
	auth, err := NewPullSecretAuth(opaque)
	require.NoError(t, err)
	assert.Nil(t, auth.ForImage("registry.example.com/app:1.0"))

	var nilAuth *PullSecretAuth
	assert.Nil(t, nilAuth.ForImage("registry.example.com/app:1.0"))
}

func TestNewPullSecretAuthInvalidConfig(t *testing.T) {
	_, err := NewPullSecretAuth(dockerConfigJSONSecret("not json"))
	assert.Error(t, err)
}
