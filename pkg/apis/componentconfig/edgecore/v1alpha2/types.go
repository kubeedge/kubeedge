/*
Copyright 2022 The KubeEdge Authors.

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

package v1alpha2

import (
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	componentbaseconfigv1alpha1 "k8s.io/component-base/config/v1alpha1"
	tailoredkubeletconfigv1beta1 "k8s.io/kubelet/config/v1beta1"
	"k8s.io/kubernetes/pkg/apis/core"

	metaconfig "github.com/kubeedge/kubeedge/pkg/apis/componentconfig/meta/v1alpha1"
)

const (
	MqttModeInternal MqttMode = 0
	MqttModeBoth     MqttMode = 1
	MqttModeExternal MqttMode = 2
)

const (
	CGroupDriverCGroupFS = "cgroupfs"
	CGroupDriverSystemd  = "systemd"
)

const (
	// DataBaseDriverName is sqlite3
	DataBaseDriverName = "sqlite3"
	// DataBaseAliasName is default
	DataBaseAliasName = "default"
	// DataBaseDataSource is edge.db
	DataBaseDataSource = "/var/lib/kubeedge/edgecore.db"
)

type ProtocolName string
type MqttMode int

// EdgeCoreConfig indicates the EdgeCore config which read from EdgeCore config file
type EdgeCoreConfig struct {
	metav1.TypeMeta
	// DataBase indicates database info
	// +Required
	DataBase *DataBase `json:"database,omitempty"`
	// Modules indicates EdgeCore modules config
	// +Required
	Modules *Modules `json:"modules,omitempty"`
	// FeatureGates is a map of feature names to bools that enable or disable alpha/experimental features.
	FeatureGates map[string]bool `json:"featureGates,omitempty"`
}

// DataBase indicates the database info
type DataBase struct {
	// DriverName indicates database driver name
	// default "sqlite3"
	DriverName string `json:"driverName,omitempty"`
	// AliasName indicates alias name
	// default "default"
	AliasName string `json:"aliasName,omitempty"`
	// DataSource indicates the data source path
	// default "/var/lib/kubeedge/edgecore.db"
	DataSource string `json:"dataSource,omitempty"`
}

// Modules indicates the modules which edgeCore will be used
type Modules struct {
	// Edged indicates edged module config
	// +Required
	Edged *Edged `json:"edged,omitempty"`
	// EdgeHub indicates edgeHub module config
	// +Required
	EdgeHub *EdgeHub `json:"edgeHub,omitempty"`
	// EventBus indicates eventBus config for edgeCore
	// +Required
	EventBus *EventBus `json:"eventBus,omitempty"`
	// MetaManager indicates meta module config
	// +Required
	MetaManager *MetaManager `json:"metaManager,omitempty"`
	// ServiceBus indicates serviceBus module config
	ServiceBus *ServiceBus `json:"serviceBus,omitempty"`
	// DeviceTwin indicates deviceTwin module config
	DeviceTwin *DeviceTwin `json:"deviceTwin,omitempty"`
	// DBTest indicates dbTest module config
	DBTest *DBTest `json:"dbTest,omitempty"`
	// EdgeStream indicates edgestream module config
	// +Required
	EdgeStream *EdgeStream `json:"edgeStream,omitempty"`
}

// Edged indicates the config fo edged module
// edged is lighted-kubelet
type Edged struct {
	// Enable indicates whether EdgeHub is enabled,
	// if set to false (for debugging etc.), skip checking other EdgeHub configs.
	// default true
	Enable bool `json:"enable"`
	// TailoredKubeletConfig contains the configuration for the Kubelet, tailored by KubeEdge
	TailoredKubeletConfig *TailoredKubeletConfiguration `json:"tailoredKubeletConfig"`
	// TailoredKubeletFlag
	TailoredKubeletFlag
	// CustomInterfaceName indicates the name of the network interface used for obtaining the IP address.
	// Setting this will override the setting 'NodeIP' if provided.
	// If this is not defined the IP address is obtained by the hostname.
	// default ""
	CustomInterfaceName string `json:"customInterfaceName,omitempty"`
	//RegisterNodeNamespace indicates register node namespace
	// default "default"
	RegisterNodeNamespace string `json:"registerNodeNamespace,omitempty"`
}

// TailoredKubeletConfiguration indicates the tailored kubelet configuration.
// It is derived from Kubernetes code `KubeletConfiguration` in package `k8s.io/kubelet/config/v1beta1` and made some variant.
type TailoredKubeletConfiguration struct {
	// syncFrequency is the max period between synchronizing running
	// containers and config.
	// Default: "1m"
	// +optional
	SyncFrequency metav1.Duration `json:"syncFrequency,omitempty"`
	// address is the IP address for the Edged to serve on (set to 0.0.0.0
	// for all interfaces).
	// Default: "127.0.0.1"
	// +optional
	Address string `json:"address,omitempty"`
	// readOnlyPort is the read-only port for the Edged to serve on with
	// no authentication/authorization.
	// The port number must be between 1 and 65535, inclusive.
	// Setting this field to 0 disables the read-only service.
	// Default: 10350
	// +optional
	ReadOnlyPort int32 `json:"readOnlyPort,omitempty"`
	// registryPullQPS is the limit of registry pulls per second.
	// The value must not be a negative number.
	// Setting it to 0 means no limit.
	// Default: 5
	// +optional
	RegistryPullQPS *int32 `json:"registryPullQPS,omitempty"`
	// registryBurst is the maximum size of bursty pulls, temporarily allows
	// pulls to burst to this number, while still not exceeding registryPullQPS.
	// The value must not be a negative number.
	// Only used if registryPullQPS is greater than 0.
	// Default: 10
	// +optional
	RegistryBurst int32 `json:"registryBurst,omitempty"`
	// eventRecordQPS is the maximum event creations per second. If 0, there
	// is no limit enforced. The value cannot be a negative number.
	// Default: 0
	// +optional
	EventRecordQPS *int32 `json:"eventRecordQPS,omitempty"`
	// eventBurst is the maximum size of a burst of event creations, temporarily
	// allows event creations to burst to this number, while still not exceeding
	// eventRecordQPS. This field canot be a negative number and it is only used
	// when eventRecordQPS > 0.
	// Default: 0
	// +optional
	EventBurst int32 `json:"eventBurst,omitempty"`
	// enableDebuggingHandlers enables server endpoints for log access
	// and local running of containers and commands, including the exec,
	// attach, logs, and portforward features.
	// Default: true
	// +optional
	EnableDebuggingHandlers *bool `json:"enableDebuggingHandlers,omitempty"`
	// enableContentionProfiling enables lock contention profiling, if enableDebuggingHandlers is true.
	// Default: false
	// +optional
	EnableContentionProfiling bool `json:"enableContentionProfiling,omitempty"`
	// oomScoreAdj is The oom-score-adj value for edged process. Values
	// must be within the range [-1000, 1000].
	// Default: -999
	// +optional
	OOMScoreAdj *int32 `json:"oomScoreAdj,omitempty"`
	// clusterDomain is the DNS domain for this cluster. If set, edged will
	// configure all containers to search this domain in addition to the
	// host's search domains.
	// Default: "cluster.local"
	// +optional
	ClusterDomain string `json:"clusterDomain,omitempty"`
	// clusterDNS is a list of IP addresses for the cluster DNS server. If set,
	// edged will configure all containers to use this for DNS resolution
	// instead of the host's DNS servers.
	// Default: nil
	// +optional
	ClusterDNS []string `json:"clusterDNS,omitempty"`
	// streamingConnectionIdleTimeout is the maximum time a streaming connection
	// can be idle before the connection is automatically closed.
	// Default: "4h"
	// +optional
	StreamingConnectionIdleTimeout metav1.Duration `json:"streamingConnectionIdleTimeout,omitempty"`
	// nodeStatusUpdateFrequency is the frequency that edged computes node
	// status. If node lease feature is not enabled, it is also the frequency that
	// edged posts node status to master.
	// Note: When node lease feature is not enabled, be cautious when changing the
	// constant, it must work with nodeMonitorGracePeriod in nodecontroller.
	// Default: "10s"
	// +optional
	NodeStatusUpdateFrequency metav1.Duration `json:"nodeStatusUpdateFrequency,omitempty"`
	// nodeStatusReportFrequency is the frequency that edged posts node
	// status to master if node status does not change. edged will ignore this
	// frequency and post node status immediately if any change is detected. It is
	// only used when node lease feature is enabled. nodeStatusReportFrequency's
	// default value is 5m. But if nodeStatusUpdateFrequency is set explicitly,
	// nodeStatusReportFrequency's default value will be set to
	// nodeStatusUpdateFrequency for backward compatibility.
	// Default: "5m"
	// +optional
	NodeStatusReportFrequency metav1.Duration `json:"nodeStatusReportFrequency,omitempty"`
	// nodeLeaseDurationSeconds is the duration the edged will set on its corresponding Lease,
	// when the NodeLease feature is enabled. This feature provides an indicator of node
	// health by having the edged create and periodically renew a lease, named after the node,
	// in the kube-node-lease namespace. If the lease expires, the node can be considered unhealthy.
	// The lease is currently renewed every 10s, per KEP-0009. In the future, the lease renewal interval
	// may be set based on the lease duration.
	// The field value must be greater than 0.
	// Requires the NodeLease feature gate to be enabled.
	// Default: 40
	// +optional
	NodeLeaseDurationSeconds int32 `json:"nodeLeaseDurationSeconds,omitempty"`
	// imageMinimumGCAge is the minimum age for an unused image before it is
	// garbage collected.
	// Default: "2m"
	// +optional
	ImageMinimumGCAge metav1.Duration `json:"imageMinimumGCAge,omitempty"`
	// imageGCHighThresholdPercent is the percent of disk usage after which
	// image garbage collection is always run. The percent is calculated by
	// dividing this field value by 100, so this field must be between 0 and
	// 100, inclusive. When specified, the value must be greater than
	// imageGCLowThresholdPercent.
	// Default: 85
	// +optional
	ImageGCHighThresholdPercent *int32 `json:"imageGCHighThresholdPercent,omitempty"`
	// imageGCLowThresholdPercent is the percent of disk usage before which
	// image garbage collection is never run. Lowest disk usage to garbage
	// collect to. The percent is calculated by dividing this field value by 100,
	// so the field value must be between 0 and 100, inclusive. When specified, the
	// value must be less than imageGCHighThresholdPercent.
	// Default: 80
	// +optional
	ImageGCLowThresholdPercent *int32 `json:"imageGCLowThresholdPercent,omitempty"`
	// volumeStatsAggPeriod is the frequency for calculating and caching volume
	// disk usage for all pods.
	// Default: "1m"
	// +optional
	VolumeStatsAggPeriod metav1.Duration `json:"volumeStatsAggPeriod,omitempty"`
	// kubeletCgroups is the absolute name of cgroups to isolate the kubelet in
	// Default: ""
	// +optional
	KubeletCgroups string `json:"kubeletCgroups,omitempty"`
	// systemCgroups is absolute name of cgroups in which to place
	// all non-kernel processes that are not already in a container. Empty
	// for no container. Rolling back the flag requires a reboot.
	// The cgroupRoot must be specified if this field is not empty.
	// Default: ""
	// +optional
	SystemCgroups string `json:"systemCgroups,omitempty"`
	// cgroupRoot is the root cgroup to use for pods. This is handled by the
	// container runtime on a best effort basis.
	// Default: ""
	// +optional
	CgroupRoot string `json:"cgroupRoot,omitempty"`
	// cgroupsPerQOS enable QoS based CGroup hierarchy: top level CGroups for QoS classes
	// and all Burstable and BestEffort Pods are brought up under their specific top level
	// QoS CGroup.
	// Default: true
	// +optional
	CgroupsPerQOS *bool `json:"cgroupsPerQOS,omitempty"`
	// cgroupDriver is the driver edged uses to manipulate CGroups on the host (cgroupfs
	// or systemd).
	// Default: "cgroupfs"
	// +optional
	CgroupDriver string `json:"cgroupDriver,omitempty"`
	// cpuManagerPolicy is the name of the policy to use.
	// Requires the CPUManager feature gate to be enabled.
	// Default: "None"
	// +optional
	CPUManagerPolicy string `json:"cpuManagerPolicy,omitempty"`
	// cpuManagerPolicyOptions is a set of key=value which 	allows to set extra options
	// to fine tune the behaviour of the cpu manager policies.
	// Requires  both the "CPUManager" and "CPUManagerPolicyOptions" feature gates to be enabled.
	// Default: nil
	// +optional
	CPUManagerPolicyOptions map[string]string `json:"cpuManagerPolicyOptions,omitempty"`
	// cpuManagerReconcilePeriod is the reconciliation period for the CPU Manager.
	// Requires the CPUManager feature gate to be enabled.
	// Default: "10s"
	// +optional
	CPUManagerReconcilePeriod metav1.Duration `json:"cpuManagerReconcilePeriod,omitempty"`
	// memoryManagerPolicy is the name of the policy to use by memory manager.
	// Requires the MemoryManager feature gate to be enabled.
	// Default: "none"
	// +optional
	MemoryManagerPolicy string `json:"memoryManagerPolicy,omitempty"`
	// topologyManagerPolicy is the name of the topology manager policy to use.
	// Valid values include:
	//
	// - `restricted`: edged only allows pods with optimal NUMA node alignment for
	//   requested resources;
	// - `best-effort`: edged will favor pods with NUMA alignment of CPU and device
	//   resources;
	// - `none`: edged has no knowledge of NUMA alignment of a pod's CPU and device resources.
	// - `single-numa-node`: edged only allows pods with a single NUMA alignment
	//   of CPU and device resources.
	//
	// Policies other than "none" require the TopologyManager feature gate to be enabled.
	// Default: "none"
	// +optional
	TopologyManagerPolicy string `json:"topologyManagerPolicy,omitempty"`
	// topologyManagerScope represents the scope of topology hint generation
	// that topology manager requests and hint providers generate. Valid values include:
	//
	// - `container`: topology policy is applied on a per-container basis.
	// - `pod`: topology policy is applied on a per-pod basis.
	//
	// "pod" scope requires the TopologyManager feature gate to be enabled.
	// Default: "container"
	// +optional
	TopologyManagerScope string `json:"topologyManagerScope,omitempty"`
	// qosReserved is a set of resource name to percentage pairs that specify
	// the minimum percentage of a resource reserved for exclusive use by the
	// guaranteed QoS tier.
	// Currently supported resources: "memory"
	// Requires the QOSReserved feature gate to be enabled.
	// Default: nil
	// +optional
	QOSReserved map[string]string `json:"qosReserved,omitempty"`
	// runtimeRequestTimeout is the timeout for all runtime requests except long running
	// requests - pull, logs, exec and attach.
	// Default: "2m"
	// +optional
	RuntimeRequestTimeout metav1.Duration `json:"runtimeRequestTimeout,omitempty"`
	// hairpinMode specifies how the edged should configure the container
	// bridge for hairpin packets.
	// Setting this flag allows endpoints in a Service to loadbalance back to
	// themselves if they should try to access their own Service. Values:
	//
	// - "promiscuous-bridge": make the container bridge promiscuous.
	// - "hairpin-veth":       set the hairpin flag on container veth interfaces.
	// - "none":               do nothing.
	//
	// Generally, one must set `--hairpin-mode=hairpin-veth to` achieve hairpin NAT,
	// because promiscuous-bridge assumes the existence of a container bridge named cbr0.
	// Default: "promiscuous-bridge"
	// +optional
	HairpinMode string `json:"hairpinMode,omitempty"`
	// maxPods is the maximum number of Pods that can run on this Kubelet.
	// The value must be a non-negative integer.
	// Default: 110
	// +optional
	MaxPods int32 `json:"maxPods,omitempty"`
	// podCIDR is the CIDR to use for pod IP addresses, only used in standalone mode.
	// In cluster mode, this is obtained from the control plane.
	// Default: ""
	// +optional
	PodCIDR string `json:"podCIDR,omitempty"`
	// podPidsLimit is the maximum number of PIDs in any pod.
	// Default: -1
	// +optional
	PodPidsLimit *int64 `json:"podPidsLimit,omitempty"`
	// resolvConf is the resolver configuration file used as the basis
	// for the container DNS resolution configuration.
	// Default: "/etc/resolv.conf"
	// +optional
	ResolverConfig *string `json:"resolvConf,omitempty"`
	// cpuCFSQuota enables CPU CFS quota enforcement for containers that
	// specify CPU limits.
	// Default: true
	// +optional
	CPUCFSQuota *bool `json:"cpuCFSQuota,omitempty"`
	// cpuCFSQuotaPeriod is the CPU CFS quota period value, `cpu.cfs_period_us`.
	// The value must be between 1 us and 1 second, inclusive.
	// Requires the CustomCPUCFSQuotaPeriod feature gate to be enabled.
	// Default: "100ms"
	// +optional
	CPUCFSQuotaPeriod *metav1.Duration `json:"cpuCFSQuotaPeriod,omitempty"`
	// nodeStatusMaxImages caps the number of images reported in Node.status.images.
	// The value must be greater than -2.
	// Note: If -1 is specified, no cap will be applied. If 0 is specified, no image is returned.
	// Default: 0
	// +optional
	NodeStatusMaxImages *int32 `json:"nodeStatusMaxImages,omitempty"`
	// maxOpenFiles is Number of files that can be opened by edged process.
	// The value must be a non-negative number.
	// Default: 1000000
	// +optional
	MaxOpenFiles int64 `json:"maxOpenFiles,omitempty"`
	// contentType is contentType of requests sent to apiserver.
	// Default: "application/json"
	// +optional
	ContentType string `json:"contentType,omitempty"`
	// serializeImagePulls when enabled, tells the edged to pull images one
	// at a time. We recommend *not* changing the default value on nodes that
	// run docker daemon with version  < 1.9 or an Aufs storage backend.
	// Issue #10959 has more details.
	// Default: true
	// +optional
	SerializeImagePulls *bool `json:"serializeImagePulls,omitempty"`
	// evictionHard is a map of signal names to quantities that defines hard eviction
	// thresholds. For example: `{"memory.available": "300Mi"}`.
	// To explicitly disable, pass a 0% or 100% threshold on an arbitrary resource.
	// Default:
	//   memory.available:  "100Mi"
	//   nodefs.available:  "10%"
	//   nodefs.inodesFree: "5%"
	//   imagefs.available: "15%"
	// +optional
	EvictionHard map[string]string `json:"evictionHard,omitempty"`
	// evictionSoft is a map of signal names to quantities that defines soft eviction thresholds.
	// For example: `{"memory.available": "300Mi"}`.
	// Default: nil
	// +optional
	EvictionSoft map[string]string `json:"evictionSoft,omitempty"`
	// evictionSoftGracePeriod is a map of signal names to quantities that defines grace
	// periods for each soft eviction signal. For example: `{"memory.available": "30s"}`.
	// Default: nil
	// +optional
	EvictionSoftGracePeriod map[string]string `json:"evictionSoftGracePeriod,omitempty"`
	// evictionPressureTransitionPeriod is the duration for which the kubelet has to wait
	// before transitioning out of an eviction pressure condition.
	// Default: "5m"
	// +optional
	EvictionPressureTransitionPeriod metav1.Duration `json:"evictionPressureTransitionPeriod,omitempty"`
	// evictionMaxPodGracePeriod is the maximum allowed grace period (in seconds) to use
	// when terminating pods in response to a soft eviction threshold being met. This value
	// effectively caps the Pod's terminationGracePeriodSeconds value during soft evictions.
	// Note: Due to issue #64530, the behavior has a bug where this value currently just
	// overrides the grace period during soft eviction, which can increase the grace
	// period from what is set on the Pod. This bug will be fixed in a future release.
	// Default: 0
	// +optional
	EvictionMaxPodGracePeriod int32 `json:"evictionMaxPodGracePeriod,omitempty"`
	// evictionMinimumReclaim is a map of signal names to quantities that defines minimum reclaims,
	// which describe the minimum amount of a given resource the kubelet will reclaim when
	// performing a pod eviction while that resource is under pressure.
	// For example: `{"imagefs.available": "2Gi"}`.
	// Default: nil
	// +optional
	EvictionMinimumReclaim map[string]string `json:"evictionMinimumReclaim,omitempty"`
	// podsPerCore is the maximum number of pods per core. Cannot exceed maxPods.
	// The value must be a non-negative integer.
	// If 0, there is no limit on the number of Pods.
	// Default: 0
	// +optional
	PodsPerCore int32 `json:"podsPerCore,omitempty"`
	// enableControllerAttachDetach enables the Attach/Detach controller to
	// manage attachment/detachment of volumes scheduled to this node, and
	// disables kubelet from executing any attach/detach operations.
	// Default: true
	// +optional
	EnableControllerAttachDetach *bool `json:"enableControllerAttachDetach,omitempty"`
	// protectKernelDefaults, if true, causes the edged to error if kernel
	// flags are not as it expects. Otherwise the edged will attempt to modify
	// kernel flags to match its expectation.
	// Default: false
	// +optional
	ProtectKernelDefaults bool `json:"protectKernelDefaults,omitempty"`
	// makeIPTablesUtilChains, if true, causes the edged ensures a set of iptables rules
	// are present on host.
	// These rules will serve as utility rules for various components, e.g. kube-proxy.
	// The rules will be created based on iptablesMasqueradeBit and iptablesDropBit.
	// Default: true
	// +optional
	MakeIPTablesUtilChains *bool `json:"makeIPTablesUtilChains,omitempty"`
	// iptablesMasqueradeBit is the bit of the iptables fwmark space to mark for SNAT.
	// Values must be within the range [0, 31]. Must be different from other mark bits.
	// Warning: Please match the value of the corresponding parameter in kube-proxy.
	// TODO: clean up IPTablesMasqueradeBit in kube-proxy.
	// Default: 14
	// +optional
	IPTablesMasqueradeBit *int32 `json:"iptablesMasqueradeBit,omitempty"`
	// iptablesDropBit is the bit of the iptables fwmark space to mark for dropping packets.
	// Values must be within the range [0, 31]. Must be different from other mark bits.
	// Default: 15
	// +optional
	IPTablesDropBit *int32 `json:"iptablesDropBit,omitempty"`
	// featureGates is a map of feature names to bools that enable or disable experimental
	// features. This field modifies piecemeal the built-in default values from
	// "k8s.io/kubernetes/pkg/features/kube_features.go".
	// Default: nil
	// +optional
	FeatureGates map[string]bool `json:"featureGates,omitempty"`
	// failSwapOn tells the edged to fail to start if swap is enabled on the node.
	// Default: false
	// +optional
	FailSwapOn *bool `json:"failSwapOn,omitempty"`
	// memorySwap configures swap memory available to container workloads.
	// +featureGate=NodeSwap
	// +optional
	MemorySwap tailoredkubeletconfigv1beta1.MemorySwapConfiguration `json:"memorySwap,omitempty"`
	// containerLogMaxSize is a quantity defining the maximum size of the container log
	// file before it is rotated. For example: "5Mi" or "256Ki".
	// Default: "10Mi"
	// +optional
	ContainerLogMaxSize string `json:"containerLogMaxSize,omitempty"`
	// containerLogMaxFiles specifies the maximum number of container log files that can
	// be present for a container.
	// Default: 5
	// +optional
	ContainerLogMaxFiles *int32 `json:"containerLogMaxFiles,omitempty"`
	// configMapAndSecretChangeDetectionStrategy is a mode in which ConfigMap and Secret
	// managers are running. Valid values include:
	//
	// - `Get`: edged fetches necessary objects directly from the API server;
	// - `Cache`: edged uses TTL cache for object fetched from the API server;
	// - `Watch`: edged uses watches to observe changes to objects that are in its interest.
	//
	// Default: "Get"
	// +optional
	ConfigMapAndSecretChangeDetectionStrategy tailoredkubeletconfigv1beta1.ResourceChangeDetectionStrategy `json:"configMapAndSecretChangeDetectionStrategy,omitempty"`

	/* the following fields are meant for Node Allocatable */

	// systemReserved is a set of ResourceName=ResourceQuantity (e.g. cpu=200m,memory=150G)
	// pairs that describe resources reserved for non-kubernetes components.
	// Currently only cpu and memory are supported.
	// See http://kubernetes.io/docs/user-guide/compute-resources for more detail.
	// Default: nil
	// +optional
	SystemReserved map[string]string `json:"systemReserved,omitempty"`
	// kubeReserved is a set of ResourceName=ResourceQuantity (e.g. cpu=200m,memory=150G) pairs
	// that describe resources reserved for kubernetes system components.
	// Currently cpu, memory and local storage for root file system are supported.
	// See https://kubernetes.io/docs/concepts/configuration/manage-resources-containers/
	// for more details.
	// Default: nil
	// +optional
	KubeReserved map[string]string `json:"kubeReserved,omitempty"`
	// The reservedSystemCPUs option specifies the CPU list reserved for the host
	// level system threads and kubernetes related threads. This provide a "static"
	// CPU list rather than the "dynamic" list by systemReserved and kubeReserved.
	// This option does not support systemReservedCgroup or kubeReservedCgroup.
	ReservedSystemCPUs string `json:"reservedSystemCPUs,omitempty"`
	// showHiddenMetricsForVersion is the previous version for which you want to show
	// hidden metrics.
	// Only the previous minor version is meaningful, other values will not be allowed.
	// The format is `<major>.<minor>`, e.g.: `1.16`.
	// The purpose of this format is make sure you have the opportunity to notice
	// if the next release hides additional metrics, rather than being surprised
	// when they are permanently removed in the release after that.
	// Default: ""
	// +optional
	ShowHiddenMetricsForVersion string `json:"showHiddenMetricsForVersion,omitempty"`
	// systemReservedCgroup helps the edged identify absolute name of top level CGroup used
	// to enforce `systemReserved` compute resource reservation for OS system daemons.
	// Refer to [Node Allocatable](https://git.k8s.io/community/contributors/design-proposals/node/node-allocatable.md)
	// doc for more information.
	// Default: ""
	// +optional
	SystemReservedCgroup string `json:"systemReservedCgroup,omitempty"`
	// kubeReservedCgroup helps the edged identify absolute name of top level CGroup used
	// to enforce `KubeReserved` compute resource reservation for Kubernetes node system daemons.
	// Refer to [Node Allocatable](https://git.k8s.io/community/contributors/design-proposals/node/node-allocatable.md)
	// doc for more information.
	// Default: ""
	// +optional
	KubeReservedCgroup string `json:"kubeReservedCgroup,omitempty"`
	// This flag specifies the various Node Allocatable enforcements that edged needs to perform.
	// This flag accepts a list of options. Acceptable options are `none`, `pods`,
	// `system-reserved` and `kube-reserved`.
	// If `none` is specified, no other options may be specified.
	// When `system-reserved` is in the list, systemReservedCgroup must be specified.
	// When `kube-reserved` is in the list, kubeReservedCgroup must be specified.
	// This field is supported only when `cgroupsPerQOS` is set to true.
	// Refer to [Node Allocatable](https://git.k8s.io/community/contributors/design-proposals/node/node-allocatable.md)
	// for more information.
	// Default: ["pods"]
	// +optional
	EnforceNodeAllocatable []string `json:"enforceNodeAllocatable,omitempty"`
	// A comma separated whitelist of unsafe sysctls or sysctl patterns (ending in `*`).
	// Unsafe sysctl groups are `kernel.shm*`, `kernel.msg*`, `kernel.sem`, `fs.mqueue.*`,
	// and `net.*`. For example: "`kernel.msg*,net.ipv4.route.min_pmtu`"
	// Default: []
	// +optional
	AllowedUnsafeSysctls []string `json:"allowedUnsafeSysctls,omitempty"`
	// volumePluginDir is the full path of the directory in which to search
	// for additional third party volume plugins.
	// Default: "/usr/libexec/kubernetes/kubelet-plugins/volume/exec/"
	// +optional
	VolumePluginDir string `json:"volumePluginDir,omitempty"`
	// kernelMemcgNotification, if set, instructs the edged to integrate with the
	// kernel memcg notification for determining if memory eviction thresholds are
	// exceeded rather than polling.
	// Default: false
	// +optional
	KernelMemcgNotification bool `json:"kernelMemcgNotification,omitempty"`
	// logging specifies the options of logging.
	// Refer to [Logs Options](https://github.com/kubernetes/component-base/blob/master/logs/options.go)
	// for more information.
	// Default:
	//   Format: text
	// + optional
	Logging componentbaseconfigv1alpha1.LoggingConfiguration `json:"logging,omitempty"`
	// enableSystemLogHandler enables system logs via web interface host:port/logs/
	// Default: true
	// +optional
	EnableSystemLogHandler *bool `json:"enableSystemLogHandler,omitempty"`
	// shutdownGracePeriod specifies the total duration that the node should delay the
	// shutdown and total grace period for pod termination during a node shutdown.
	// Default: "0s"
	// +featureGate=GracefulNodeShutdown
	// +optional
	ShutdownGracePeriod metav1.Duration `json:"shutdownGracePeriod,omitempty"`
	// shutdownGracePeriodCriticalPods specifies the duration used to terminate critical
	// pods during a node shutdown. This should be less than shutdownGracePeriod.
	// For example, if shutdownGracePeriod=30s, and shutdownGracePeriodCriticalPods=10s,
	// during a node shutdown the first 20 seconds would be reserved for gracefully
	// terminating normal pods, and the last 10 seconds would be reserved for terminating
	// critical pods.
	// Default: "0s"
	// +featureGate=GracefulNodeShutdown
	// +optional
	ShutdownGracePeriodCriticalPods metav1.Duration `json:"shutdownGracePeriodCriticalPods,omitempty"`
	// shutdownGracePeriodByPodPriority specifies the shutdown grace period for Pods based
	// on their associated priority class value.
	// When a shutdown request is received, the Kubelet will initiate shutdown on all pods
	// running on the node with a grace period that depends on the priority of the pod,
	// and then wait for all pods to exit.
	// Each entry in the array represents the graceful shutdown time a pod with a priority
	// class value that lies in the range of that value and the next higher entry in the
	// list when the node is shutting down.
	// For example, to allow critical pods 10s to shutdown, priority>=10000 pods 20s to
	// shutdown, and all remaining pods 30s to shutdown.
	//
	// shutdownGracePeriodByPodPriority:
	//   - priority: 2000000000
	//     shutdownGracePeriodSeconds: 10
	//   - priority: 10000
	//     shutdownGracePeriodSeconds: 20
	//   - priority: 0
	//     shutdownGracePeriodSeconds: 30
	//
	// The time the Kubelet will wait before exiting will at most be the maximum of all
	// shutdownGracePeriodSeconds for each priority class range represented on the node.
	// When all pods have exited or reached their grace periods, the Kubelet will release
	// the shutdown inhibit lock.
	// Requires the GracefulNodeShutdown feature gate to be enabled.
	// This configuration must be empty if either ShutdownGracePeriod or ShutdownGracePeriodCriticalPods is set.
	// Default: nil
	// +featureGate=GracefulNodeShutdownBasedOnPodPriority
	// +optional
	ShutdownGracePeriodByPodPriority []tailoredkubeletconfigv1beta1.ShutdownGracePeriodByPodPriority `json:"shutdownGracePeriodByPodPriority,omitempty"`
	// reservedMemory specifies a comma-separated list of memory reservations for NUMA nodes.
	// The parameter makes sense only in the context of the memory manager feature.
	// The memory manager will not allocate reserved memory for container workloads.
	// For example, if you have a NUMA0 with 10Gi of memory and the reservedMemory was
	// specified to reserve 1Gi of memory at NUMA0, the memory manager will assume that
	// only 9Gi is available for allocation.
	// You can specify a different amount of NUMA node and memory types.
	// You can omit this parameter at all, but you should be aware that the amount of
	// reserved memory from all NUMA nodes should be equal to the amount of memory specified
	// by the [node allocatable](https://kubernetes.io/docs/tasks/administer-cluster/reserve-compute-resources/#node-allocatable).
	// If at least one node allocatable parameter has a non-zero value, you will need
	// to specify at least one NUMA node.
	// Also, avoid specifying:
	//
	// 1. Duplicates, the same NUMA node, and memory type, but with a different value.
	// 2. zero limits for any memory type.
	// 3. NUMAs nodes IDs that do not exist under the machine.
	// 4. memory types except for memory and hugepages-<size>
	//
	// Default: nil
	// +optional
	ReservedMemory []tailoredkubeletconfigv1beta1.MemoryReservation `json:"reservedMemory,omitempty"`
	// enableProfilingHandler enables profiling via web interface host:port/debug/pprof/
	// Default: true
	// +optional
	EnableProfilingHandler *bool `json:"enableProfilingHandler,omitempty"`
	// enableDebugFlagsHandler enables flags endpoint via web interface host:port/debug/flags/v
	// Default: true
	// +optional
	EnableDebugFlagsHandler *bool `json:"enableDebugFlagsHandler,omitempty"`
	// SeccompDefault enables the use of `RuntimeDefault` as the default seccomp profile for all workloads.
	// This requires the corresponding SeccompDefault feature gate to be enabled as well.
	// Default: false
	// +optional
	SeccompDefault *bool `json:"seccompDefault,omitempty"`
	// MemoryThrottlingFactor specifies the factor multiplied by the memory limit or node allocatable memory
	// when setting the cgroupv2 memory.high value to enforce MemoryQoS.
	// Decreasing this factor will set lower high limit for container cgroups and put heavier reclaim pressure
	// while increasing will put less reclaim pressure.
	// See http://kep.k8s.io/2570 for more details.
	// Default: 0.8
	// +featureGate=MemoryQoS
	// +optional
	MemoryThrottlingFactor *float64 `json:"memoryThrottlingFactor,omitempty"`
	// registerWithTaints are an array of taints to add to a node object when
	// the kubelet registers itself. This only takes effect when registerNode
	// is true and upon the initial registration of the node.
	// Default: nil
	// +optional
	RegisterWithTaints []v1.Taint `json:"registerWithTaints,omitempty"`
	// registerNode enables automatic registration with the apiserver.
	// Default: true
	// +optional
	RegisterNode *bool `json:"registerNode,omitempty"`
}

// TailoredKubeletFlag indicates the tailored kubelet flag
type TailoredKubeletFlag struct {
	// HostnameOverride is the hostname used to identify the kubelet instead
	// of the actual hostname.
	// default os.Hostname()
	HostnameOverride string `json:"hostnameOverride,omitempty"`
	// NodeIP is IP address of the node.
	// If set, edged will use this IP address for the node.
	NodeIP string `json:"nodeIP,omitempty"`
	// Container-runtime-specific options.
	ContainerRuntimeOptions
	// rootDirectory is the directory path to place kubelet files (volume
	// mounts,etc).
	// default "/var/lib/edged"
	RootDirectory string `json:"rootDirectory,omitempty"`
	// registerNode enables automatic registration with the apiserver.
	// default true
	// DEPRECATED: This parameter should be set via the TailoredKubeletConfig
	RegisterNode bool `json:"registerNode,omitempty"`
	// registerWithTaints are an array of taints to add to a node object when
	// the edgecore registers itself. This only takes effect when registerNode
	// is true and upon the initial registration of the node.
	RegisterWithTaints []core.Taint `json:"registerWithTaints,omitempty"`
	// WindowsService should be set to true if kubelet is running as a service on Windows.
	// Its corresponding flag only gets registered in Windows builds.
	WindowsService bool `json:"windowsService,omitempty"`
	// WindowsPriorityClass sets the priority class associated with the Kubelet process
	// Its corresponding flag only gets registered in Windows builds
	// The default priority class associated with any process in Windows is NORMAL_PRIORITY_CLASS. Keeping it as is
	// to maintain backwards compatibility.
	// Source: https://docs.microsoft.com/en-us/windows/win32/procthread/scheduling-priorities
	WindowsPriorityClass string `json:"windowsPriorityClass,omitempty"`
	// remoteRuntimeEndpoint is the endpoint of remote runtime service
	// default "unix:///var/run/dockershim.sock"
	RemoteRuntimeEndpoint string `json:"remoteRuntimeEndpoint,omitempty"`
	// remoteImageEndpoint is the endpoint of remote image service
	// default "unix:///var/run/dockershim.sock"
	RemoteImageEndpoint string `json:"remoteImageEndpoint,omitempty"`
	// experimentalMounterPath is the path of mounter binary. Leave empty to use the default mount path
	ExperimentalMounterPath string `json:"experimentalMounterPath,omitempty"`
	// This flag, if set, enables a check prior to mount operations to verify that the required components
	// (binaries, etc.) to mount the volume are available on the underlying node. If the check is enabled
	// and fails the mount operation fails.
	ExperimentalCheckNodeCapabilitiesBeforeMount bool `json:"experimentalCheckNodeCapabilitiesBeforeMount,omitempty"`
	// This flag, if set, will avoid including `EvictionHard` limits while computing Node Allocatable.
	// Refer to [Node Allocatable](https://git.k8s.io/community/contributors/design-proposals/node/node-allocatable.md) doc for more information.
	ExperimentalNodeAllocatableIgnoreEvictionThreshold bool `json:"experimentalNodeAllocatableIgnoreEvictionThreshold,omitempty"`
	// Node Labels are the node labels to add when registering the node in the cluster
	NodeLabels map[string]string `json:"nodeLabels,omitempty"`
	// seccompProfileRoot is the directory path for seccomp profiles.
	SeccompProfileRoot string `json:"seccompProfileRoot,omitempty"`
	// DEPRECATED FLAGS
	// minimumGCAge is the minimum age for a finished container before it is
	// garbage collected.
	MinimumGCAge metav1.Duration `json:"minimumGCAge,omitempty"`
	// maxPerPodContainerCount is the maximum number of old instances to
	// retain per container. Each container takes up some disk space.
	MaxPerPodContainerCount int32 `json:"maxPerPodContainerCount,omitempty"`
	// maxContainerCount is the maximum number of old instances of containers
	// to retain globally. Each container takes up some disk space.
	MaxContainerCount int32 `json:"maxContainerCount,omitempty"`
	// masterServiceNamespace is The namespace from which the kubernetes
	// master services should be injected into pods.
	MasterServiceNamespace string `json:"masterServiceNamespace,omitempty"`
	// registerSchedulable tells the edgecore to register the node as
	// schedulable. Won't have any effect if register-node is false.
	// DEPRECATED: use registerWithTaints instead
	RegisterSchedulable bool `json:"registerSchedulable,omitempty"`
	// nonMasqueradeCIDR configures masquerading: traffic to IPs outside this range will use IP masquerade.
	NonMasqueradeCIDR string `json:"nonMasqueradeCidr,omitempty"`
	// This flag, if set, instructs the edged to keep volumes from terminated pods mounted to the node.
	// This can be useful for debugging volume related issues.
	KeepTerminatedPodVolumes bool `json:"keepTerminatedPodVolumes,omitempty"`
	// SeccompDefault enables the use of `RuntimeDefault` as the default seccomp profile for all workloads on the node.
	// To use this flag, the corresponding SeccompDefault feature gate must be enabled.
	SeccompDefault bool `json:"seccompDefault,omitempty"`
}

// ContainerRuntimeOptions defines options for the container runtime.
type ContainerRuntimeOptions struct {
	// General Options.

	// ContainerRuntime is the container runtime to use.
	// default "docker"
	ContainerRuntime string `json:"containerRuntime,omitempty"`
	// RuntimeCgroups that container runtime is expected to be isolated in.
	RuntimeCgroups string `json:"runtimeCgroups,omitempty"`
	// Docker-specific options.

	// DockershimRootDirectory is the path to the dockershim root directory. Defaults to
	// /var/lib/dockershim if unset. Exposed for integration testing (e.g. in OpenShift).
	DockershimRootDirectory string `json:"dockershimRootDirectory,omitempty"`
	// PodSandboxImage is the image whose network/ipc namespaces
	// containers in each pod will use.
	// default kubeedge/pause:3.1
	PodSandboxImage string `json:"podSandboxImage,omitempty"`
	// DockerEndpoint is the path to the docker endpoint to communicate with.
	DockerEndpoint string `json:"dockerEndpoint,omitempty"`
	// If no pulling progress is made before the deadline imagePullProgressDeadline,
	// the image pulling will be cancelled.
	// Defaults 1m0s.
	// +optional
	ImagePullProgressDeadline metav1.Duration `json:"imagePullProgressDeadline,omitempty"`
	// Network plugin options.

	// networkPluginName is the name of the network plugin to be invoked for
	// various events in kubelet/pod lifecycle
	// default ""
	NetworkPluginName string `json:"networkPluginName,omitempty"`
	// NetworkPluginMTU is the MTU to be passed to the network plugin,
	// and overrides the default MTU for cases where it cannot be automatically
	// computed (such as IPSEC).
	// default 1500
	NetworkPluginMTU int32 `json:"networkPluginMTU,omitempty"`
	// CNIConfDir is the full path of the directory in which to search for
	// CNI config files
	// default "/etc/cni/net.d"
	CNIConfDir string `json:"cniConfDir,omitempty"`
	// CNIBinDir is the full path of the directory in which to search for
	// CNI plugin binaries
	// default "/opt/cni/bin"
	CNIBinDir string `json:"cniBinDir,omitempty"`
	// CNICacheDir is the full path of the directory in which CNI should store
	// cache files
	// default "/var/lib/cni/cache"
	CNICacheDir string `json:"cniCacheDir,omitempty"`
}

// EdgeHub indicates the EdgeHub module config
type EdgeHub struct {
	// Enable indicates whether EdgeHub is enabled,
	// if set to false (for debugging etc.), skip checking other EdgeHub configs.
	// default true
	Enable bool `json:"enable"`
	// Heartbeat indicates heart beat (second)
	// default 15
	Heartbeat int32 `json:"heartbeat,omitempty"`
	// MessageQPS is the QPS to allow while send message to cloudHub.
	// DefaultQPS: 30
	MessageQPS int32 `json:"messageQPS,omitempty"`
	// MessageBurst is the burst to allow while send message to cloudHub.
	// DefaultBurst: 60
	MessageBurst int32 `json:"messageBurst,omitempty"`
	// ProjectID indicates project id
	// default e632aba927ea4ac2b575ec1603d56f10
	ProjectID string `json:"projectID,omitempty"`
	// TLSCAFile set ca file path
	// default "/etc/kubeedge/ca/rootCA.crt"
	TLSCAFile string `json:"tlsCaFile,omitempty"`
	// TLSCertFile indicates the file containing x509 Certificate for HTTPS
	// default "/etc/kubeedge/certs/server.crt"
	TLSCertFile string `json:"tlsCertFile,omitempty"`
	// TLSPrivateKeyFile indicates the file containing x509 private key matching tlsCertFile
	// default "/etc/kubeedge/certs/server.key"
	TLSPrivateKeyFile string `json:"tlsPrivateKeyFile,omitempty"`
	// Quic indicates quic config for EdgeHub module
	// Optional if websocket is configured
	Quic *EdgeHubQUIC `json:"quic,omitempty"`
	// WebSocket indicates websocket config for EdgeHub module
	// Optional if quic is configured
	WebSocket *EdgeHubWebSocket `json:"websocket,omitempty"`
	// Token indicates the priority of joining the cluster for the edge
	// Deprecated: will be removed in future release, will not be saved in configuration file
	Token string `json:"token"`
	// HTTPServer indicates the server for edge to apply for the certificate.
	HTTPServer string `json:"httpServer,omitempty"`
	// RotateCertificates indicates whether edge certificate can be rotated
	// default true
	RotateCertificates bool `json:"rotateCertificates,omitempty"`
}

// EdgeHubQUIC indicates the quic client config
type EdgeHubQUIC struct {
	// Enable indicates whether enable this protocol
	// default false
	Enable bool `json:"enable"`
	// HandshakeTimeout indicates hand shake timeout (second)
	// default 30
	HandshakeTimeout int32 `json:"handshakeTimeout,omitempty"`
	// ReadDeadline indicates read dead line (second)
	// default 15
	ReadDeadline int32 `json:"readDeadline,omitempty"`
	// Server indicates quic server address (ip:port)
	// +Required
	Server string `json:"server,omitempty"`
	// WriteDeadline indicates write dead line (second)
	// default 15
	WriteDeadline int32 `json:"writeDeadline,omitempty"`
}

// EdgeHubWebSocket indicates the websocket client config
type EdgeHubWebSocket struct {
	// Enable indicates whether enable this protocol
	// default true
	Enable bool `json:"enable"`
	// HandshakeTimeout indicates handshake timeout (second)
	// default  30
	HandshakeTimeout int32 `json:"handshakeTimeout,omitempty"`
	// ReadDeadline indicates read dead line (second)
	// default 15
	ReadDeadline int32 `json:"readDeadline,omitempty"`
	// Server indicates websocket server address (ip:port)
	// +Required
	Server string `json:"server,omitempty"`
	// WriteDeadline indicates write dead line (second)
	// default 15
	WriteDeadline int32 `json:"writeDeadline,omitempty"`
}

// EventBus indicates the event bus module config
type EventBus struct {
	// Enable indicates whether EventBus is enabled, if set to false (for debugging etc.),
	// skip checking other EventBus configs.
	// default true
	Enable bool `json:"enable"`
	// MqttQOS indicates mqtt qos
	// 0: QOSAtMostOnce, 1: QOSAtLeastOnce, 2: QOSExactlyOnce
	// default 0
	// Note: Can not use "omitempty" option,  It will affect the output of the default configuration file
	MqttQOS uint8 `json:"mqttQOS"`
	// MqttRetain indicates whether server will store the message and can be delivered to future subscribers,
	// if this flag set true, sever will store the message and can be delivered to future subscribers
	// default false
	// Note: Can not use "omitempty" option,  It will affect the output of the default configuration file
	MqttRetain bool `json:"mqttRetain"`
	// MqttSessionQueueSize indicates the size of how many sessions will be handled.
	// default 100
	MqttSessionQueueSize int32 `json:"mqttSessionQueueSize,omitempty"`
	// MqttServerInternal indicates internal mqtt broker url
	// default "tcp://127.0.0.1:1884"
	MqttServerInternal string `json:"mqttServerInternal,omitempty"`
	// MqttServerExternal indicates external mqtt broker url
	// default "tcp://127.0.0.1:1883"
	MqttServerExternal string `json:"mqttServerExternal,omitempty"`
	// MqttSubClientID indicates mqtt subscribe ClientID
	// default ""
	MqttSubClientID string `json:"mqttSubClientID"`
	// MqttPubClientID indicates mqtt publish ClientID
	// default ""
	MqttPubClientID string `json:"mqttPubClientID"`
	// MqttUsername indicates mqtt username
	// default ""
	MqttUsername string `json:"mqttUsername"`
	// MqttPassword indicates mqtt password
	// default ""
	MqttPassword string `json:"mqttPassword"`
	// MqttMode indicates which broker type will be choose
	// 0: internal mqtt broker enable only.
	// 1: internal and external mqtt broker enable.
	// 2: external mqtt broker enable only
	// +Required
	// default: 2
	MqttMode MqttMode `json:"mqttMode"`
	// Tls indicates tls config for EventBus module
	TLS *EventBusTLS `json:"eventBusTLS,omitempty"`
}

// EventBusTLS indicates the EventBus tls config with MQTT broker
type EventBusTLS struct {
	// Enable indicates whether enable tls connection
	// default false
	Enable bool `json:"enable"`
	// TLSMqttCAFile sets ca file path
	// default "/etc/kubeedge/ca/rootCA.crt"
	TLSMqttCAFile string `json:"tlsMqttCAFile,omitempty"`
	// TLSMqttCertFile indicates the file containing x509 Certificate for HTTPS
	// default "/etc/kubeedge/certs/server.crt"
	TLSMqttCertFile string `json:"tlsMqttCertFile,omitempty"`
	// TLSMqttPrivateKeyFile indicates the file containing x509 private key matching tlsMqttCertFile
	// default "/etc/kubeedge/certs/server.key"
	TLSMqttPrivateKeyFile string `json:"tlsMqttPrivateKeyFile,omitempty"`
}

// MetaManager indicates the MetaManager module config
type MetaManager struct {
	// Enable indicates whether MetaManager is enabled,
	// if set to false (for debugging etc.), skip checking other MetaManager configs.
	// default true
	Enable bool `json:"enable"`
	// ContextSendGroup indicates send group
	ContextSendGroup metaconfig.GroupName `json:"contextSendGroup,omitempty"`
	// ContextSendModule indicates send module
	ContextSendModule metaconfig.ModuleName `json:"contextSendModule,omitempty"`
	// RemoteQueryTimeout indicates remote query timeout (second)
	// default 60
	RemoteQueryTimeout int32 `json:"remoteQueryTimeout,omitempty"`
	// The config of MetaServer
	MetaServer *MetaServer `json:"metaServer,omitempty"`
}

type MetaServer struct {
	Enable            bool   `json:"enable"`
	Server            string `json:"server"`
	TLSCaFile         string `json:"tlsCaFile"`
	TLSCertFile       string `json:"tlsCertFile"`
	TLSPrivateKeyFile string `json:"tlsPrivateKeyFile"`
}

// ServiceBus indicates the ServiceBus module config
type ServiceBus struct {
	// Enable indicates whether ServiceBus is enabled,
	// if set to false (for debugging etc.), skip checking other ServiceBus configs.
	// default false
	Enable bool `json:"enable"`
	// Address indicates address for http server
	Server string `json:"server"`
	// Port indicates port for http server
	Port int `json:"port"`
	// Timeout indicates timeout for servicebus receive mseeage
	Timeout int `json:"timeout"`
}

// DeviceTwin indicates the DeviceTwin module config
type DeviceTwin struct {
	// Enable indicates whether DeviceTwin is enabled,
	// if set to false (for debugging etc.), skip checking other DeviceTwin configs.
	// default true
	Enable bool `json:"enable"`
}

// DBTest indicates the DBTest module config
type DBTest struct {
	// Enable indicates whether DBTest is enabled,
	// if set to false (for debugging etc.), skip checking other DBTest configs.
	// default false
	Enable bool `json:"enable"`
}

// EdgeStream indicates the stream controller
type EdgeStream struct {
	// Enable indicates whether edgestream is enabled, if set to false (for debugging etc.), skip checking other configs.
	// default true
	Enable bool `json:"enable"`

	// TLSTunnelCAFile indicates ca file path
	// default /etc/kubeedge/ca/rootCA.crt
	TLSTunnelCAFile string `json:"tlsTunnelCAFile,omitempty"`

	// TLSTunnelCertFile indicates the file containing x509 Certificate for HTTPS
	// default /etc/kubeedge/certs/server.crt
	TLSTunnelCertFile string `json:"tlsTunnelCertFile,omitempty"`
	// TLSTunnelPrivateKeyFile indicates the file containing x509 private key matching tlsCertFile
	// default /etc/kubeedge/certs/server.key
	TLSTunnelPrivateKeyFile string `json:"tlsTunnelPrivateKeyFile,omitempty"`

	// HandshakeTimeout indicates handshake timeout (second)
	// default  30
	HandshakeTimeout int32 `json:"handshakeTimeout,omitempty"`
	// ReadDeadline indicates read dead line (second)
	// default 15
	ReadDeadline int32 `json:"readDeadline,omitempty"`
	// TunnelServer indicates websocket server address (ip:port)
	// +Required
	TunnelServer string `json:"server,omitempty"`
	// WriteDeadline indicates write dead line (second)
	// default 15
	WriteDeadline int32 `json:"writeDeadline,omitempty"`
}
