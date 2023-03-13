/*
Copyright 2023 The KubeEdge Authors.

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

package config

import (
	"reflect"
	"testing"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	kubeletoptions "k8s.io/kubernetes/cmd/kubelet/app/options"
	kubeletconfig "k8s.io/kubernetes/pkg/kubelet/apis/config"

	"github.com/kubeedge/kubeedge/common/constants"
	"github.com/kubeedge/kubeedge/pkg/apis/componentconfig/edgecore/v1alpha2"
)

// TestInitConfigure tests configuration initialized only once
func TestInitConfigure(t *testing.T) {
	tests := []struct {
		name  string
		edged *v1alpha2.Edged
		want  v1alpha2.Edged
	}{
		{
			name: "initial edged config",
			edged: &v1alpha2.Edged{
				Enable: true,
			},
			want: v1alpha2.Edged{
				Enable: true,
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			testEdged := &v1alpha2.Edged{
				Enable: false,
			}
			InitConfigure(tt.edged)
			InitConfigure(testEdged)
			if !reflect.DeepEqual(tt.want, Config.Edged) {
				t.Errorf("TestInitConfigure %v, want %v", Config.Edged, tt.want)
			}
		})
	}
}

func TestConvertEdgedKubeletConfigurationToConfigKubeletConfiguration(t *testing.T) {
	defaultTailedKubeletConfig := v1alpha2.TailoredKubeletConfiguration{}
	v1alpha2.SetDefaultsKubeletConfiguration(&defaultTailedKubeletConfig)
	var kubeletConfig kubeletconfig.KubeletConfiguration

	t.Run("Convert Edged Kubelet Config", func(t *testing.T) {
		err := ConvertEdgedKubeletConfigurationToConfigKubeletConfiguration(&defaultTailedKubeletConfig, &kubeletConfig, nil)
		if err != nil {
			t.Errorf("Convert Edged Kubelet Configuration failed, %v", err)
		}
	})
}

func TestConvertConfigEdgedFlagToConfigKubeletFlag(t *testing.T) {
	tailorKubeletFlag := v1alpha2.TailoredKubeletFlag{
		HostnameOverride: "testnode",
		ContainerRuntimeOptions: v1alpha2.ContainerRuntimeOptions{
			ContainerRuntime:          constants.DefaultRuntimeType,
			PodSandboxImage:           constants.DefaultPodSandboxImage,
			ImagePullProgressDeadline: metav1.Duration{Duration: constants.DefaultImagePullProgressDeadline},
			CNIConfDir:                constants.DefaultCNIConfDir,
			CNIBinDir:                 constants.DefaultCNIBinDir,
			CNICacheDir:               constants.DefaultCNICacheDir,
			NetworkPluginMTU:          constants.DefaultNetworkPluginMTU,
		},
		RootDirectory:           constants.DefaultRootDir,
		MasterServiceNamespace:  metav1.NamespaceDefault,
		RemoteRuntimeEndpoint:   constants.DefaultRemoteRuntimeEndpoint,
		RemoteImageEndpoint:     constants.DefaultRemoteImageEndpoint,
		MaxContainerCount:       -1,
		MaxPerPodContainerCount: 1,
		MinimumGCAge:            metav1.Duration{Duration: 0},
		NonMasqueradeCIDR:       "10.0.0.0/8",
		NodeLabels:              make(map[string]string),
		RegisterNode:            true,
		RegisterSchedulable:     true,
	}

	var kubeletFlags kubeletoptions.KubeletFlags
	t.Run("Convert Edged Kubelet Config", func(t *testing.T) {
		ConvertConfigEdgedFlagToConfigKubeletFlag(&tailorKubeletFlag, &kubeletFlags)
		if kubeletFlags.HostnameOverride != tailorKubeletFlag.HostnameOverride {
			t.Errorf("Convert Edged Kubelet Flag failed")
		}
	})
}
