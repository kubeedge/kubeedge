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

package config

import (
	"maps"
	"os"
	"path/filepath"
	"reflect"
	"testing"
	"time"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	utilfeature "k8s.io/apiserver/pkg/util/feature"
	kubeletconfigv1beta1 "k8s.io/kubelet/config/v1beta1"
	"k8s.io/kubernetes/pkg/features"
	kubeletconfig "k8s.io/kubernetes/pkg/kubelet/apis/config"
	configv1beta1 "k8s.io/kubernetes/pkg/kubelet/apis/config/v1beta1"
	"k8s.io/kubernetes/pkg/kubelet/eviction"
	"k8s.io/utils/ptr"

	"github.com/kubeedge/api/apis/componentconfig/edgecore/v1alpha2"
)

func TestKubeletConfigurationParseAndConvert(t *testing.T) {
	configFile := filepath.Join(t.TempDir(), "edgecore.yaml")
	err := os.WriteFile(configFile, []byte(`
modules:
  edged:
    tailoredKubeletConfig:
      featureGates:
        KubeletEnsureSecretPulledImages: true
        KubeletCrashLoopBackOffMax: true
      imagePullCredentialsVerificationPolicy: NeverVerifyAllowlistedImages
      preloadedImagesVerificationAllowlist:
        - registry.example.com/preloaded/*
      mergeDefaultEvictionSettings: true
      userNamespaces:
        idsPerPod: 131072
      crashLoopBackOff:
        maxContainerRestartPeriod: 30s
`), 0600)
	if err != nil {
		t.Fatal(err)
	}
	edgeConfig := v1alpha2.NewDefaultEdgeCoreConfig()
	if err := edgeConfig.Parse(configFile); err != nil {
		t.Fatal(err)
	}
	var out kubeletconfig.KubeletConfiguration
	if err := ConvertEdgedKubeletConfigurationToConfigKubeletConfiguration(edgeConfig.Modules.Edged.TailoredKubeletConfig, &out, nil); err != nil {
		t.Fatal(err)
	}
	if out.ImagePullCredentialsVerificationPolicy != string(kubeletconfigv1beta1.NeverVerifyAllowlistedImages) {
		t.Errorf("unexpected image pull policy: %q", out.ImagePullCredentialsVerificationPolicy)
	}
	if !reflect.DeepEqual(out.PreloadedImagesVerificationAllowlist, []string{"registry.example.com/preloaded/*"}) {
		t.Errorf("unexpected image allowlist: %v", out.PreloadedImagesVerificationAllowlist)
	}
	if !out.MergeDefaultEvictionSettings {
		t.Error("mergeDefaultEvictionSettings was not converted")
	}
	if out.UserNamespaces == nil || out.UserNamespaces.IDsPerPod == nil || *out.UserNamespaces.IDsPerPod != 131072 {
		t.Errorf("unexpected user namespaces config: %+v", out.UserNamespaces)
	}
	if out.CrashLoopBackOff.MaxContainerRestartPeriod == nil || out.CrashLoopBackOff.MaxContainerRestartPeriod.Duration != 30*time.Second {
		t.Errorf("unexpected crash loop backoff: %+v", out.CrashLoopBackOff)
	}
}

func TestKubeletConfigurationGatedDefaults(t *testing.T) {
	for _, tc := range []struct {
		name   string
		gates  map[string]bool
		policy kubeletconfigv1beta1.ImagePullCredentialsVerificationPolicy
		want   string
	}{
		{name: "default gates"},
		{name: "disabled", gates: map[string]bool{string(features.KubeletEnsureSecretPulledImages): false}},
		{name: "enabled default", gates: map[string]bool{string(features.KubeletEnsureSecretPulledImages): true}, want: string(kubeletconfigv1beta1.NeverVerifyPreloadedImages)},
		{name: "never verify", gates: map[string]bool{string(features.KubeletEnsureSecretPulledImages): true}, policy: kubeletconfigv1beta1.NeverVerify, want: string(kubeletconfigv1beta1.NeverVerify)},
		{name: "allowlist", gates: map[string]bool{string(features.KubeletEnsureSecretPulledImages): true}, policy: kubeletconfigv1beta1.NeverVerifyAllowlistedImages, want: string(kubeletconfigv1beta1.NeverVerifyAllowlistedImages)},
		{name: "always verify", gates: map[string]bool{string(features.KubeletEnsureSecretPulledImages): true}, policy: kubeletconfigv1beta1.AlwaysVerify, want: string(kubeletconfigv1beta1.AlwaysVerify)},
	} {
		t.Run(tc.name, func(t *testing.T) {
			in := v1alpha2.TailoredKubeletConfiguration{}
			v1alpha2.SetDefaultsKubeletConfiguration(&in)
			if in.MergeDefaultEvictionSettings == nil || *in.MergeDefaultEvictionSettings {
				t.Fatal("mergeDefaultEvictionSettings must default to false")
			}
			in.FeatureGates = tc.gates
			in.ImagePullCredentialsVerificationPolicy = tc.policy
			globalGate := utilfeature.DefaultFeatureGate.Enabled(features.KubeletEnsureSecretPulledImages)
			var out kubeletconfig.KubeletConfiguration
			if err := ConvertEdgedKubeletConfigurationToConfigKubeletConfiguration(&in, &out, nil); err != nil {
				t.Fatal(err)
			}
			if out.ImagePullCredentialsVerificationPolicy != tc.want {
				t.Errorf("policy = %q, want %q", out.ImagePullCredentialsVerificationPolicy, tc.want)
			}
			if in.ImagePullCredentialsVerificationPolicy != tc.policy {
				t.Error("conversion mutated the input policy")
			}
			if utilfeature.DefaultFeatureGate.Enabled(features.KubeletEnsureSecretPulledImages) != globalGate {
				t.Error("conversion mutated the global feature gate")
			}
		})
	}
	t.Run("invalid gate", func(t *testing.T) {
		in := v1alpha2.TailoredKubeletConfiguration{FeatureGates: map[string]bool{"UnknownKubeletFeature": true}}
		if err := ConvertEdgedKubeletConfigurationToConfigKubeletConfiguration(&in, &kubeletconfig.KubeletConfiguration{}, nil); err == nil {
			t.Error("expected an error for an unknown feature gate")
		}
	})
}

func TestMergeDefaultEvictionSettings(t *testing.T) {
	for _, tc := range []struct {
		name  string
		merge *bool
		hard  map[string]string
	}{
		{name: "unspecified", hard: map[string]string{"memory.available": "300Mi"}},
		{name: "disabled", merge: ptr.To(false), hard: map[string]string{"memory.available": "300Mi"}},
		{name: "enabled", merge: ptr.To(true), hard: map[string]string{"memory.available": "300Mi", "nodefs.available": "0%"}},
		{name: "enabled empty", merge: ptr.To(true), hard: map[string]string{}},
		{name: "enabled nil", merge: ptr.To(true)},
		{name: "unspecified nil"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			original := maps.Clone(tc.hard)
			defaults := maps.Clone(eviction.DefaultEvictionHard)
			in := v1alpha2.TailoredKubeletConfiguration{MergeDefaultEvictionSettings: tc.merge, EvictionHard: tc.hard}
			var out kubeletconfig.KubeletConfiguration
			if err := ConvertEdgedKubeletConfigurationToConfigKubeletConfiguration(&in, &out, nil); err != nil {
				t.Fatal(err)
			}
			want := maps.Clone(tc.hard)
			if tc.merge != nil && *tc.merge {
				want = maps.Clone(defaults)
				maps.Copy(want, tc.hard)
			}
			if !reflect.DeepEqual(out.EvictionHard, want) {
				t.Errorf("evictionHard = %v, want %v", out.EvictionHard, want)
			}
			if !reflect.DeepEqual(in.EvictionHard, original) || !reflect.DeepEqual(eviction.DefaultEvictionHard, defaults) {
				t.Error("merging eviction defaults mutated the input or global defaults")
			}
		})
	}
}

func TestUserNamespacesConversion(t *testing.T) {
	for _, userns := range []*kubeletconfigv1beta1.UserNamespaces{
		nil,
		{},
		{IDsPerPod: ptr.To[int64](65536)},
		{IDsPerPod: ptr.To[int64](131072)},
	} {
		in := v1alpha2.TailoredKubeletConfiguration{UserNamespaces: userns}
		out := kubeletconfig.KubeletConfiguration{UserNamespaces: &kubeletconfig.UserNamespaces{IDsPerPod: ptr.To[int64](262144)}}
		if err := ConvertEdgedKubeletConfigurationToConfigKubeletConfiguration(&in, &out, nil); err != nil {
			t.Fatal(err)
		}
		if userns == nil {
			if out.UserNamespaces != nil {
				t.Error("nil userNamespaces should clear the output")
			}
		} else if out.UserNamespaces == nil || !reflect.DeepEqual(out.UserNamespaces.IDsPerPod, userns.IDsPerPod) {
			t.Errorf("userNamespaces = %+v, want %+v", out.UserNamespaces, userns)
		}
	}
}

func TestCrashLoopBackOffConversion(t *testing.T) {
	for _, tc := range []struct {
		name    string
		enabled bool
		period  *metav1.Duration
		want    *metav1.Duration
	}{
		{name: "disabled"},
		{name: "enabled default", enabled: true, want: &metav1.Duration{Duration: configv1beta1.MaxContainerBackOff}},
		{name: "enabled explicit", enabled: true, period: &metav1.Duration{Duration: 30 * time.Second}, want: &metav1.Duration{Duration: 30 * time.Second}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			in := v1alpha2.TailoredKubeletConfiguration{
				FeatureGates:     map[string]bool{string(features.KubeletCrashLoopBackOffMax): tc.enabled},
				CrashLoopBackOff: kubeletconfigv1beta1.CrashLoopBackOffConfig{MaxContainerRestartPeriod: tc.period},
			}
			var out kubeletconfig.KubeletConfiguration
			if err := ConvertEdgedKubeletConfigurationToConfigKubeletConfiguration(&in, &out, nil); err != nil {
				t.Fatal(err)
			}
			if !reflect.DeepEqual(out.CrashLoopBackOff.MaxContainerRestartPeriod, tc.want) {
				t.Errorf("maxContainerRestartPeriod = %v, want %v", out.CrashLoopBackOff.MaxContainerRestartPeriod, tc.want)
			}
		})
	}
}
