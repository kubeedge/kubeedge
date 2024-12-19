package constants

import (
	"time"
)

// Resources
const (
	DefaultServiceAccountIssuer = "https://kubernetes.default.svc.cluster.local"

	// Edged
	DefaultDockerAddress       = "unix:///var/run/docker.sock"
	DefaultRuntimeType         = "remote"
	DefaultEdgedMemoryCapacity = 7852396000

	// update PodSandboxImage version when bumping k8s vendor version, consistent with vendor/k8s.io/kubernetes/cmd/kubelet/app/options/container_runtime.go defaultPodSandboxImageVersion
	// When this value are updated, also update comments in pkg/apis/componentconfig/edgecore/v1alpha1/types.go
	DefaultPodSandboxImage             = "kubeedge/pause:3.6"
	DefaultImagePullProgressDeadline   = time.Minute
	DefaultImageGCHighThreshold        = 80
	DefaultImageGCLowThreshold         = 40
	DefaultMaximumDeadContainersPerPod = 1
	DefaultHostnameOverride            = "default-edge-node"
	DefaultRegisterNodeNamespace       = "default"
	DefaultNetworkPluginMTU            = 1500
	DefaultConcurrentConsumers         = 5
	DefaultCgroupRoot                  = ""
	DefaultVolumeStatsAggPeriod        = time.Minute
	DefaultWebSocketPort               = 10000
	DefaultQuicPort                    = 10001
	DefaultTunnelPort                  = 10004
	DefaultClusterDomain               = "cluster.local"

	// MetaManager
	DefaultRemoteQueryTimeout = 60
	DefaultMetaServerAddr     = "127.0.0.1:10550"
	DefaultDummyServerAddr    = "169.254.30.10:10550"

	// Config
	DefaultKubeContentType = "application/vnd.kubernetes.protobuf"
	DefaultNodeLimit       = 500

	// EdgeController
	DefaultUpdatePodStatusWorkers            = 1
	DefaultUpdateNodeStatusWorkers           = 1
	DefaultProcessEventWorkers               = 4
	DefaultQueryConfigMapWorkers             = 100
	DefaultQuerySecretWorkers                = 100
	DefaultQueryPersistentVolumeWorkers      = 4
	DefaultQueryPersistentVolumeClaimWorkers = 4
	DefaultQueryVolumeAttachmentWorkers      = 4
	DefaultCreateNodeWorkers                 = 100
	DefaultUpdateNodeWorkers                 = 4
	DefaultPatchPodWorkers                   = 100
	DefaultDeletePodWorkers                  = 100
	DefaultUpdateRuleStatusWorkers           = 4
	DefaultQueryLeaseWorkers                 = 100
	DefaultServiceAccountTokenWorkers        = 100
	DefaultCreatePodWorkers                  = 4
	DefaultCertificateSigningRequestWorkers  = 4

	DefaultUpdatePodStatusBuffer            = 1024
	DefaultProcessEventBuffer               = 1024
	DefaultUpdateNodeStatusBuffer           = 1024
	DefaultQueryConfigMapBuffer             = 1024
	DefaultQuerySecretBuffer                = 1024
	DefaultQueryPersistentVolumeBuffer      = 1024
	DefaultQueryPersistentVolumeClaimBuffer = 1024
	DefaultQueryVolumeAttachmentBuffer      = 1024
	DefaultCreateNodeBuffer                 = 1024
	DefaultUpdateNodeBuffer                 = 1024
	DefaultPatchPodBuffer                   = 1024
	DefaultDeletePodBuffer                  = 1024
	DefaultQueryLeaseBuffer                 = 1024
	DefaultServiceAccountTokenBuffer        = 1024
	DefaultCreatePodBuffer                  = 1024
	DefaultCertificateSigningRequestBuffer  = 1024

	DefaultPodEventBuffer           = 1
	DefaultConfigMapEventBuffer     = 1
	DefaultSecretEventBuffer        = 1
	DefaultRulesEventBuffer         = 1
	DefaultRuleEndpointsEventBuffer = 1

	// DeviceController
	DefaultUpdateDeviceTwinsBuffer   = 1024
	DefaultUpdateDeviceStatesBuffer  = 1024
	DefaultDeviceEventBuffer         = 1
	DefaultDeviceModelEventBuffer    = 1
	DefaultUpdateDeviceStatusWorkers = 1

	// TaskManager
	DefaultNodeUpgradeJobStatusBuffer = 1024
	DefaultNodeUpgradeJobEventBuffer  = 1
	DefaultNodeUpgradeJobWorkers      = 1

	ServerAddress = "127.0.0.1"
	// ServerPort is the default port for the edgecore server on each host machine.
	// May be overridden by a flag at startup in the future.
	ServerPort = 10350

	// MessageSuccessfulContent is the successful content value of Message struct
	DefaultQPS   = 30
	DefaultBurst = 60

	// DeviceTwin
	DefaultDMISockPath = "/etc/kubeedge/dmi.sock"

	// DefaultMosquittoContainerName ...
	// Deprecated: the mqtt broker is alreay managed by the DaemonSet in the cloud
	DefaultMosquittoContainerName = "mqtt-kubeedge"
	// DeployMqttContainerEnv ...
	// Deprecated: the mqtt broker is alreay managed by the DaemonSet in the cloud
	DeployMqttContainerEnv = "DEPLOY_MQTT_CONTAINER"
	// DefaultMosquittoImage ...
	// Deprecated: the mqtt broker is alreay managed by the DaemonSet in the cloud
	DefaultMosquittoImage = "eclipse-mosquitto:1.6.15"
)
