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

package metaclient

import (
	"errors"
	"strings"
	"testing"
	"time"

	"github.com/agiledragon/gomonkey/v2"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"k8s.io/client-go/kubernetes"
	restclient "k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"

	cfgv1alpha2 "github.com/kubeedge/api/apis/componentconfig/edgecore/v1alpha2"
	"github.com/kubeedge/api/client/clientset/versioned"
	keadutil "github.com/kubeedge/kubeedge/keadm/cmd/keadm/app/cmd/util"
)

func newBaseConfig() *cfgv1alpha2.EdgeCoreConfig {
	config := cfgv1alpha2.NewDefaultEdgeCoreConfig()
	config.Modules.MetaManager.MetaServer.Enable = true
	config.Modules.MetaManager.MetaServer.Server = "127.0.0.1:12345"
	return config
}

func TestGetKubeConfigWithConfig_HTTP(t *testing.T) {
	patches := gomonkey.NewPatches()
	defer patches.Reset()

	config := newBaseConfig()
	config.FeatureGates = map[string]bool{"requireAuthorization": false}

	patches.ApplyFunc(clientcmd.BuildConfigFromFlags, func(masterUrl, kubeconfigPath string) (*restclient.Config, error) {
		assert.True(t, strings.HasPrefix(masterUrl, "http://"))
		return &restclient.Config{}, nil
	})

	cfg, err := GetKubeConfigWithConfig(config)
	require.NoError(t, err)
	assert.NotNil(t, cfg)
	assert.Equal(t, 1*time.Minute, cfg.Timeout)
}

func TestGetKubeConfigWithConfig_HTTPS(t *testing.T) {
	patches := gomonkey.NewPatches()
	defer patches.Reset()

	config := newBaseConfig()
	config.FeatureGates = map[string]bool{"requireAuthorization": true}
	config.Modules.MetaManager.MetaServer.TLSCertFile = "/path/to/cert"
	config.Modules.MetaManager.MetaServer.TLSPrivateKeyFile = "/path/to/key"
	config.Modules.MetaManager.MetaServer.TLSCaFile = "/path/to/ca"

	patches.ApplyFunc(clientcmd.BuildConfigFromFlags, func(masterUrl, kubeconfigPath string) (*restclient.Config, error) {
		assert.True(t, strings.HasPrefix(masterUrl, "https://"))
		return &restclient.Config{}, nil
	})

	cfg, err := GetKubeConfigWithConfig(config)
	require.NoError(t, err)
	assert.Equal(t, "/path/to/ca", cfg.TLSClientConfig.CAFile)
	assert.Equal(t, "/path/to/cert", cfg.TLSClientConfig.CertFile)
	assert.Equal(t, "/path/to/key", cfg.TLSClientConfig.KeyFile)
}

func TestGetKubeConfigWithConfig_Disabled(t *testing.T) {
	config := newBaseConfig()
	config.Modules.MetaManager.MetaServer.Enable = false

	_, err := GetKubeConfigWithConfig(config)
	require.ErrorContains(t, err, "metaserver don't open")
}

func TestKubeClient_Success(t *testing.T) {
	patches := gomonkey.NewPatches()
	defer patches.Reset()

	cfg := newBaseConfig()
	patches.ApplyFuncReturn(keadutil.ParseEdgecoreConfig, cfg, nil)
	patches.ApplyFuncReturn(clientcmd.BuildConfigFromFlags, &restclient.Config{}, nil)
	patches.ApplyFunc(kubernetes.NewForConfig, func(c *restclient.Config) (kubernetes.Interface, error) {
		return &kubernetes.Clientset{}, nil
	})

	cli, err := KubeClient()
	require.NoError(t, err)
	assert.NotNil(t, cli)
}

func TestKubeClient_ParseError(t *testing.T) {
	patches := gomonkey.NewPatches()
	defer patches.Reset()

	patches.ApplyFuncReturn(keadutil.ParseEdgecoreConfig, (*cfgv1alpha2.EdgeCoreConfig)(nil), errors.New("parse error"))

	_, err := KubeClient()
	require.ErrorContains(t, err, "parse error")
}

func TestKubeClientWithConfig_NewForConfigError(t *testing.T) {
	patches := gomonkey.NewPatches()
	defer patches.Reset()

	cfg := newBaseConfig()
	patches.ApplyFuncReturn(clientcmd.BuildConfigFromFlags, &restclient.Config{}, nil)
	patches.ApplyFunc(kubernetes.NewForConfig, func(c *restclient.Config) (kubernetes.Interface, error) {
		return nil, errors.New("new for config error")
	})

	_, err := KubeClientWithConfig(cfg)
	require.ErrorContains(t, err, "new for config error")
}

func TestVersionedKubeClient_Success(t *testing.T) {
	patches := gomonkey.NewPatches()
	defer patches.Reset()

	cfg := newBaseConfig()
	patches.ApplyFuncReturn(keadutil.ParseEdgecoreConfig, cfg, nil)
	patches.ApplyFuncReturn(clientcmd.BuildConfigFromFlags, &restclient.Config{}, nil)
	patches.ApplyFunc(versioned.NewForConfig, func(c *restclient.Config) (versioned.Interface, error) {
		return &versioned.Clientset{}, nil
	})

	cli, err := VersionedKubeClient()
	require.NoError(t, err)
	assert.NotNil(t, cli)
}

func TestVersionedKubeClient_ParseError(t *testing.T) {
	patches := gomonkey.NewPatches()
	defer patches.Reset()

	patches.ApplyFuncReturn(keadutil.ParseEdgecoreConfig, (*cfgv1alpha2.EdgeCoreConfig)(nil), errors.New("parse error"))

	_, err := VersionedKubeClient()
	require.ErrorContains(t, err, "parse error")
}

func TestVersionedKubeClientWithConfig_NewForConfigError(t *testing.T) {
	patches := gomonkey.NewPatches()
	defer patches.Reset()

	cfg := newBaseConfig()
	patches.ApplyFuncReturn(clientcmd.BuildConfigFromFlags, &restclient.Config{}, nil)
	patches.ApplyFunc(versioned.NewForConfig, func(c *restclient.Config) (versioned.Interface, error) {
		return nil, errors.New("versioned new for config error")
	})

	_, err := VersionedKubeClientWithConfig(cfg)
	require.ErrorContains(t, err, "versioned new for config error")
}
