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

package edgecore

import (
	"context"
	"errors"
	"testing"

	"github.com/agiledragon/gomonkey/v2"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/kubernetes/fake"

	"github.com/kubeedge/api/apis/componentconfig/edgecore/v1alpha2"
	"github.com/kubeedge/kubeedge/keadm/cmd/keadm/app/cmd/util/metaclient"
)

func TestGetVersion(t *testing.T) {
	ctx := context.TODO()
	cfg := &v1alpha2.EdgeCoreConfig{
		Modules: &v1alpha2.Modules{
			Edged: &v1alpha2.Edged{
				TailoredKubeletFlag: v1alpha2.TailoredKubeletFlag{
					HostnameOverride: "test-node",
				},
			},
		},
		EdgeCoreVersion: "0.0.0",
	}

	t.Run("get kube client failed", func(t *testing.T) {
		patches := gomonkey.NewPatches()
		defer patches.Reset()

		patches.ApplyFunc(metaclient.KubeClientWithConfig,
			func(_config *v1alpha2.EdgeCoreConfig) (kubernetes.Interface, error) {
				return nil, errors.New("test error")
			})

		ver := GetVersion(ctx, cfg)
		require.Equal(t, cfg.EdgeCoreVersion, ver)
	})

	t.Run("not found node", func(t *testing.T) {
		cli := fake.NewSimpleClientset()

		patches := gomonkey.NewPatches()
		defer patches.Reset()

		patches.ApplyFunc(metaclient.KubeClientWithConfig,
			func(_config *v1alpha2.EdgeCoreConfig) (kubernetes.Interface, error) {
				return cli, nil
			})

		ver := GetVersion(ctx, cfg)
		require.Equal(t, cfg.EdgeCoreVersion, ver)
	})

	t.Run("get version from the node info", func(t *testing.T) {
		cli := fake.NewSimpleClientset(&corev1.Node{
			ObjectMeta: metav1.ObjectMeta{
				Name: "test-node",
			},
			Status: corev1.NodeStatus{
				NodeInfo: corev1.NodeSystemInfo{
					KubeletVersion: "v1.30.0-kubeedge-v1.20.0-beta.0.71+3ec13c91a30adb-dirty",
				},
			},
		})

		patches := gomonkey.NewPatches()
		defer patches.Reset()

		patches.ApplyFunc(metaclient.KubeClientWithConfig,
			func(_config *v1alpha2.EdgeCoreConfig) (kubernetes.Interface, error) {
				return cli, nil
			})

		ver := GetVersion(ctx, cfg)
		assert.Equal(t, "v1.20.0-beta.0.71+3ec13c91a30adb-dirty", ver)
	})
}
