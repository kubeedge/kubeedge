// +build !providerless

/*
Copyright 2019 The Kubernetes Authors.

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

package app

import (
	// Credential providers
	_ "k8s.io/kubernetes/pkg/credentialprovider/aws"
	_ "k8s.io/kubernetes/pkg/credentialprovider/azure"
	_ "k8s.io/kubernetes/pkg/credentialprovider/gcp"

	"k8s.io/component-base/featuregate"
	"k8s.io/klog/v2"
	"k8s.io/kubernetes/pkg/volume"
	"k8s.io/kubernetes/pkg/volume/csimigration"
)

type probeFn func() []volume.VolumePlugin

func appendPluginBasedOnFeatureFlags(plugins []volume.VolumePlugin, inTreePluginName string,
	featureGate featuregate.FeatureGate, pluginInfo pluginInfo) ([]volume.VolumePlugin, error) {
	// Skip appending the in-tree plugin to the list of plugins to be probed/initialized
	// if the CSIMigration feature flag and plugin specific feature flag indicating
	// CSI migration is complete
	migrationComplete, err := csimigration.CheckMigrationFeatureFlags(featureGate, pluginInfo.pluginMigrationFeature,
		pluginInfo.pluginMigrationCompleteFeature, pluginInfo.pluginUnregisterFeature)
	if err != nil {
		klog.InfoS("Unexpected CSI Migration Feature Flags combination detected, CSI Migration may not take effect", "err", err)
		// TODO: fail and return here once alpha only tests can set the feature flags for a plugin correctly
	}
	// TODO: This can be removed after feature flag CSIMigrationvSphereComplete is removed.
	if migrationComplete {
		klog.InfoS("Skipped registration of plugin since migration is completed", "pluginName", inTreePluginName)
		return plugins, nil
	}
	if featureGate.Enabled(pluginInfo.pluginUnregisterFeature) {
		klog.InfoS("Skipped registration of plugin since feature flag is enabled", "pluginName", inTreePluginName, "featureFlag", pluginInfo.pluginUnregisterFeature)
		return plugins, nil
	}

	plugins = append(plugins, pluginInfo.pluginProbeFunction()...)
	return plugins, nil
}

type pluginInfo struct {
	pluginMigrationFeature featuregate.Feature
	// deprecated, only to keep here for vSphere
	pluginMigrationCompleteFeature featuregate.Feature
	pluginUnregisterFeature        featuregate.Feature
	pluginProbeFunction            probeFn
}

func appendLegacyProviderVolumes(allPlugins []volume.VolumePlugin, featureGate featuregate.FeatureGate) ([]volume.VolumePlugin, error) {
	pluginMigrationStatus := make(map[string]pluginInfo)

	var err error
	for pluginName, pluginInfo := range pluginMigrationStatus {
		allPlugins, err = appendPluginBasedOnFeatureFlags(allPlugins, pluginName, featureGate, pluginInfo)
		if err != nil {
			return allPlugins, err
		}
	}
	return allPlugins, nil
}
