/*
Copyright 2018 The Kubernetes Authors.

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
	"math"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/util/sets"
	componentbaseconfig "k8s.io/component-base/config"
)

const (
	// SchedulerDefaultLockObjectNamespace defines default scheduler lock object namespace ("kube-system")
	SchedulerDefaultLockObjectNamespace string = metav1.NamespaceSystem

	// SchedulerDefaultLockObjectName defines default scheduler lock object name ("kube-scheduler")
	SchedulerDefaultLockObjectName = "kube-scheduler"

	// SchedulerPolicyConfigMapKey defines the key of the element in the
	// scheduler's policy ConfigMap that contains scheduler's policy config.
	SchedulerPolicyConfigMapKey = "policy.cfg"

	// SchedulerDefaultProviderName defines the default provider names
	SchedulerDefaultProviderName = "DefaultProvider"
)

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// KubeSchedulerConfiguration configures a scheduler
type KubeSchedulerConfiguration struct {
	metav1.TypeMeta

	// AlgorithmSource specifies the scheduler algorithm source.
	// TODO(#87526): Remove AlgorithmSource from this package
	// DEPRECATED: AlgorithmSource is removed in the v1alpha2 ComponentConfig
	AlgorithmSource SchedulerAlgorithmSource

	// LeaderElection defines the configuration of leader election client.
	LeaderElection KubeSchedulerLeaderElectionConfiguration

	// ClientConnection specifies the kubeconfig file and client connection
	// settings for the proxy server to use when communicating with the apiserver.
	ClientConnection componentbaseconfig.ClientConnectionConfiguration
	// HealthzBindAddress is the IP address and port for the health check server to serve on,
	// defaulting to 0.0.0.0:10251
	HealthzBindAddress string
	// MetricsBindAddress is the IP address and port for the metrics server to
	// serve on, defaulting to 0.0.0.0:10251.
	MetricsBindAddress string

	// DebuggingConfiguration holds configuration for Debugging related features
	// TODO: We might wanna make this a substruct like Debugging componentbaseconfig.DebuggingConfiguration
	componentbaseconfig.DebuggingConfiguration

	// DisablePreemption disables the pod preemption feature.
	DisablePreemption bool

	// PercentageOfNodeToScore is the percentage of all nodes that once found feasible
	// for running a pod, the scheduler stops its search for more feasible nodes in
	// the cluster. This helps improve scheduler's performance. Scheduler always tries to find
	// at least "minFeasibleNodesToFind" feasible nodes no matter what the value of this flag is.
	// Example: if the cluster size is 500 nodes and the value of this flag is 30,
	// then scheduler stops finding further feasible nodes once it finds 150 feasible ones.
	// When the value is 0, default percentage (5%--50% based on the size of the cluster) of the
	// nodes will be scored.
	PercentageOfNodesToScore int32

	// Duration to wait for a binding operation to complete before timing out
	// Value must be non-negative integer. The value zero indicates no waiting.
	// If this value is nil, the default value will be used.
	BindTimeoutSeconds int64

	// PodInitialBackoffSeconds is the initial backoff for unschedulable pods.
	// If specified, it must be greater than 0. If this value is null, the default value (1s)
	// will be used.
	PodInitialBackoffSeconds int64

	// PodMaxBackoffSeconds is the max backoff for unschedulable pods.
	// If specified, it must be greater than or equal to podInitialBackoffSeconds. If this value is null,
	// the default value (10s) will be used.
	PodMaxBackoffSeconds int64

	// Profiles are scheduling profiles that kube-scheduler supports. Pods can
	// choose to be scheduled under a particular profile by setting its associated
	// scheduler name. Pods that don't specify any scheduler name are scheduled
	// with the "default-scheduler" profile, if present here.
	Profiles []KubeSchedulerProfile

	// Extenders are the list of scheduler extenders, each holding the values of how to communicate
	// with the extender. These extenders are shared by all scheduler profiles.
	Extenders []Extender
}

// KubeSchedulerProfile is a scheduling profile.
type KubeSchedulerProfile struct {
	// SchedulerName is the name of the scheduler associated to this profile.
	// If SchedulerName matches with the pod's "spec.schedulerName", then the pod
	// is scheduled with this profile.
	SchedulerName string

	// Plugins specify the set of plugins that should be enabled or disabled.
	// Enabled plugins are the ones that should be enabled in addition to the
	// default plugins. Disabled plugins are any of the default plugins that
	// should be disabled.
	// When no enabled or disabled plugin is specified for an extension point,
	// default plugins for that extension point will be used if there is any.
	// If a QueueSort plugin is specified, the same QueueSort Plugin and
	// PluginConfig must be specified for all profiles.
	Plugins *Plugins

	// PluginConfig is an optional set of custom plugin arguments for each plugin.
	// Omitting config args for a plugin is equivalent to using the default config
	// for that plugin.
	PluginConfig []PluginConfig
}

// SchedulerAlgorithmSource is the source of a scheduler algorithm. One source
// field must be specified, and source fields are mutually exclusive.
type SchedulerAlgorithmSource struct {
	// Policy is a policy based algorithm source.
	Policy *SchedulerPolicySource
	// Provider is the name of a scheduling algorithm provider to use.
	Provider *string
}

// SchedulerPolicySource configures a means to obtain a scheduler Policy. One
// source field must be specified, and source fields are mutually exclusive.
type SchedulerPolicySource struct {
	// File is a file policy source.
	File *SchedulerPolicyFileSource
	// ConfigMap is a config map policy source.
	ConfigMap *SchedulerPolicyConfigMapSource
}

// SchedulerPolicyFileSource is a policy serialized to disk and accessed via
// path.
type SchedulerPolicyFileSource struct {
	// Path is the location of a serialized policy.
	Path string
}

// SchedulerPolicyConfigMapSource is a policy serialized into a config map value
// under the SchedulerPolicyConfigMapKey key.
type SchedulerPolicyConfigMapSource struct {
	// Namespace is the namespace of the policy config map.
	Namespace string
	// Name is the name of the policy config map.
	Name string
}

// KubeSchedulerLeaderElectionConfiguration expands LeaderElectionConfiguration
// to include scheduler specific configuration.
type KubeSchedulerLeaderElectionConfiguration struct {
	componentbaseconfig.LeaderElectionConfiguration
}

// Plugins include multiple extension points. When specified, the list of plugins for
// a particular extension point are the only ones enabled. If an extension point is
// omitted from the config, then the default set of plugins is used for that extension point.
// Enabled plugins are called in the order specified here, after default plugins. If they need to
// be invoked before default plugins, default plugins must be disabled and re-enabled here in desired order.
type Plugins struct {
	// QueueSort is a list of plugins that should be invoked when sorting pods in the scheduling queue.
	QueueSort *PluginSet

	// PreFilter is a list of plugins that should be invoked at "PreFilter" extension point of the scheduling framework.
	PreFilter *PluginSet

	// Filter is a list of plugins that should be invoked when filtering out nodes that cannot run the Pod.
	Filter *PluginSet

	// PreScore is a list of plugins that are invoked before scoring.
	PreScore *PluginSet

	// Score is a list of plugins that should be invoked when ranking nodes that have passed the filtering phase.
	Score *PluginSet

	// Reserve is a list of plugins invoked when reserving a node to run the pod.
	Reserve *PluginSet

	// Permit is a list of plugins that control binding of a Pod. These plugins can prevent or delay binding of a Pod.
	Permit *PluginSet

	// PreBind is a list of plugins that should be invoked before a pod is bound.
	PreBind *PluginSet

	// Bind is a list of plugins that should be invoked at "Bind" extension point of the scheduling framework.
	// The scheduler call these plugins in order. Scheduler skips the rest of these plugins as soon as one returns success.
	Bind *PluginSet

	// PostBind is a list of plugins that should be invoked after a pod is successfully bound.
	PostBind *PluginSet

	// Unreserve is a list of plugins invoked when a pod that was previously reserved is rejected in a later phase.
	Unreserve *PluginSet
}

// PluginSet specifies enabled and disabled plugins for an extension point.
// If an array is empty, missing, or nil, default plugins at that extension point will be used.
type PluginSet struct {
	// Enabled specifies plugins that should be enabled in addition to default plugins.
	// These are called after default plugins and in the same order specified here.
	Enabled []Plugin
	// Disabled specifies default plugins that should be disabled.
	// When all default plugins need to be disabled, an array containing only one "*" should be provided.
	Disabled []Plugin
}

// Plugin specifies a plugin name and its weight when applicable. Weight is used only for Score plugins.
type Plugin struct {
	// Name defines the name of plugin
	Name string
	// Weight defines the weight of plugin, only used for Score plugins.
	Weight int32
}

// PluginConfig specifies arguments that should be passed to a plugin at the time of initialization.
// A plugin that is invoked at multiple extension points is initialized once. Args can have arbitrary structure.
// It is up to the plugin to process these Args.
type PluginConfig struct {
	// Name defines the name of plugin being configured
	Name string
	// Args defines the arguments passed to the plugins at the time of initialization. Args can have arbitrary structure.
	Args runtime.Unknown
}

/*
 * NOTE: The following variables and methods are intentionally left out of the staging mirror.
 */
const (
	// DefaultPercentageOfNodesToScore defines the percentage of nodes of all nodes
	// that once found feasible, the scheduler stops looking for more nodes.
	// A value of 0 means adaptive, meaning the scheduler figures out a proper default.
	DefaultPercentageOfNodesToScore = 0

	// MaxCustomPriorityScore is the max score UtilizationShapePoint expects.
	MaxCustomPriorityScore int64 = 10

	// MaxTotalScore is the maximum total score.
	MaxTotalScore int64 = math.MaxInt64

	// MaxWeight defines the max weight value allowed for custom PriorityPolicy
	MaxWeight = MaxTotalScore / MaxCustomPriorityScore
)

func appendPluginSet(dst *PluginSet, src *PluginSet) *PluginSet {
	if dst == nil {
		dst = &PluginSet{}
	}
	if src != nil {
		dst.Enabled = append(dst.Enabled, src.Enabled...)
		dst.Disabled = append(dst.Disabled, src.Disabled...)
	}
	return dst
}

// Append appends src Plugins to current Plugins. If a PluginSet is nil, it will
// be created.
func (p *Plugins) Append(src *Plugins) {
	if p == nil || src == nil {
		return
	}
	p.QueueSort = appendPluginSet(p.QueueSort, src.QueueSort)
	p.PreFilter = appendPluginSet(p.PreFilter, src.PreFilter)
	p.Filter = appendPluginSet(p.Filter, src.Filter)
	p.PreScore = appendPluginSet(p.PreScore, src.PreScore)
	p.Score = appendPluginSet(p.Score, src.Score)
	p.Reserve = appendPluginSet(p.Reserve, src.Reserve)
	p.Permit = appendPluginSet(p.Permit, src.Permit)
	p.PreBind = appendPluginSet(p.PreBind, src.PreBind)
	p.Bind = appendPluginSet(p.Bind, src.Bind)
	p.PostBind = appendPluginSet(p.PostBind, src.PostBind)
	p.Unreserve = appendPluginSet(p.Unreserve, src.Unreserve)
}

// Apply merges the plugin configuration from custom plugins, handling disabled sets.
func (p *Plugins) Apply(customPlugins *Plugins) {
	if customPlugins == nil {
		return
	}

	p.QueueSort = mergePluginSets(p.QueueSort, customPlugins.QueueSort)
	p.PreFilter = mergePluginSets(p.PreFilter, customPlugins.PreFilter)
	p.Filter = mergePluginSets(p.Filter, customPlugins.Filter)
	p.PreScore = mergePluginSets(p.PreScore, customPlugins.PreScore)
	p.Score = mergePluginSets(p.Score, customPlugins.Score)
	p.Reserve = mergePluginSets(p.Reserve, customPlugins.Reserve)
	p.Permit = mergePluginSets(p.Permit, customPlugins.Permit)
	p.PreBind = mergePluginSets(p.PreBind, customPlugins.PreBind)
	p.Bind = mergePluginSets(p.Bind, customPlugins.Bind)
	p.PostBind = mergePluginSets(p.PostBind, customPlugins.PostBind)
	p.Unreserve = mergePluginSets(p.Unreserve, customPlugins.Unreserve)
}

func mergePluginSets(defaultPluginSet, customPluginSet *PluginSet) *PluginSet {
	if customPluginSet == nil {
		customPluginSet = &PluginSet{}
	}

	if defaultPluginSet == nil {
		defaultPluginSet = &PluginSet{}
	}

	disabledPlugins := sets.NewString()
	for _, disabledPlugin := range customPluginSet.Disabled {
		disabledPlugins.Insert(disabledPlugin.Name)
	}

	enabledPlugins := []Plugin{}
	if !disabledPlugins.Has("*") {
		for _, defaultEnabledPlugin := range defaultPluginSet.Enabled {
			if disabledPlugins.Has(defaultEnabledPlugin.Name) {
				continue
			}

			enabledPlugins = append(enabledPlugins, defaultEnabledPlugin)
		}
	}

	enabledPlugins = append(enabledPlugins, customPluginSet.Enabled...)

	return &PluginSet{Enabled: enabledPlugins}
}
