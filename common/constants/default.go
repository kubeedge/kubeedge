package constants

import (
	"time"

	v1 "k8s.io/api/core/v1"
)

// Module name and group name
const (
	// SyncController
	DefaultContextSendModuleName = "cloudhub"

	// NodeName is for the clearer log of cloudcore.
	NodeName = "NodeName"

	ProjectName = "KubeEdge"

	SystemName      = "kubeedge"
	SystemNamespace = SystemName
)

// Resources
const (
	// Certificates
	DefaultConfigDir = "/etc/kubeedge/config/"
	DefaultCAFile    = "/etc/kubeedge/ca/rootCA.crt"
	DefaultCAKeyFile = "/etc/kubeedge/ca/rootCA.key"
	DefaultCertFile  = "/etc/kubeedge/certs/server.crt"
	DefaultKeyFile   = "/etc/kubeedge/certs/server.key"

	DefaultCAURL   = "/ca.crt"
	DefaultCertURL = "/edge.crt"

	DefaultStreamCAFile   = "/etc/kubeedge/ca/streamCA.crt"
	DefaultStreamCertFile = "/etc/kubeedge/certs/stream.crt"
	DefaultStreamKeyFile  = "/etc/kubeedge/certs/stream.key"

	DefaultMqttCAFile   = "/etc/kubeedge/ca/rootCA.crt"
	DefaultMqttCertFile = "/etc/kubeedge/certs/server.crt"
	DefaultMqttKeyFile  = "/etc/kubeedge/certs/server.key"

	// Edged
	DefaultDockerAddress               = "unix:///var/run/docker.sock"
	DefaultRuntimeType                 = "docker"
	DefaultEdgedMemoryCapacity         = 7852396000
	DefaultRemoteRuntimeEndpoint       = "unix:///var/run/dockershim.sock"
	DefaultRemoteImageEndpoint         = "unix:///var/run/dockershim.sock"
	DefaultPodSandboxImage             = "kubeedge/pause:3.1"
	DefaultNodeStatusUpdateFrequency   = 10
	DefaultImagePullProgressDeadline   = 60
	DefaultRuntimeRequestTimeout       = 2
	DefaultImageGCHighThreshold        = 80
	DefaultImageGCLowThreshold         = 40
	DefaultMaximumDeadContainersPerPod = 1
	DefaultHostnameOverride            = "default-edge-node"
	DefaultRegisterNodeNamespace       = "default"
	DefaultCNIConfDir                  = "/etc/cni/net.d"
	DefaultCNIBinDir                   = "/opt/cni/bin"
	DefaultCNICacheDir                 = "/var/lib/cni/cache"
	DefaultNetworkPluginMTU            = 1500
	DefaultConcurrentConsumers         = 5
	DefaultCgroupRoot                  = ""
	DefaultVolumeStatsAggPeriod        = time.Minute
	DefaultTunnelPort                  = 10004

	CurrentSupportK8sVersion = "v1.22.6"

	// MetaManager
	DefaultRemoteQueryTimeout = 60
	DefaultMetaServerAddr     = "127.0.0.1:10550"

	// Config
	DefaultKubeContentType         = "application/vnd.kubernetes.protobuf"
	DefaultKubeNamespace           = v1.NamespaceAll
	DefaultKubeQPS                 = 100.0
	DefaultKubeBurst               = 200
	DefaultKubeUpdateNodeFrequency = 20

	// EdgeController
	DefaultUpdatePodStatusWorkers            = 1
	DefaultUpdateNodeStatusWorkers           = 1
	DefaultQueryConfigMapWorkers             = 4
	DefaultQuerySecretWorkers                = 4
	DefaultQueryServiceWorkers               = 4
	DefaultQueryEndpointsWorkers             = 4
	DefaultQueryPersistentVolumeWorkers      = 4
	DefaultQueryPersistentVolumeClaimWorkers = 4
	DefaultQueryVolumeAttachmentWorkers      = 4
	DefaultQueryNodeWorkers                  = 4
	DefaultUpdateNodeWorkers                 = 4
	DefaultDeletePodWorkers                  = 4
	DefaultUpdateRuleStatusWorkers           = 4
	DefaultServiceAccountTokenWorkers        = 4

	DefaultUpdatePodStatusBuffer            = 1024
	DefaultUpdateNodeStatusBuffer           = 1024
	DefaultQueryConfigMapBuffer             = 1024
	DefaultQuerySecretBuffer                = 1024
	DefaultQueryServiceBuffer               = 1024
	DefaultQueryEndpointsBuffer             = 1024
	DefaultQueryPersistentVolumeBuffer      = 1024
	DefaultQueryPersistentVolumeClaimBuffer = 1024
	DefaultQueryVolumeAttachmentBuffer      = 1024
	DefaultQueryNodeBuffer                  = 1024
	DefaultUpdateNodeBuffer                 = 1024
	DefaultDeletePodBuffer                  = 1024
	DefaultServiceAccountTokenBuffer        = 1024

	DefaultPodEventBuffer           = 1
	DefaultConfigMapEventBuffer     = 1
	DefaultSecretEventBuffer        = 1
	DefaultServiceEventBuffer       = 1
	DefaultEndpointsEventBuffer     = 1
	DefaultRulesEventBuffer         = 1
	DefaultRuleEndpointsEventBuffer = 1

	// DeviceController
	DefaultUpdateDeviceStatusBuffer  = 1024
	DefaultDeviceEventBuffer         = 1
	DefaultDeviceModelEventBuffer    = 1
	DefaultUpdateDeviceStatusWorkers = 1

	// Resource sep
	ResourceSep = "/"

	ResourceTypeService   = "service"
	ResourceTypeEndpoints = "endpoints"

	ResourceTypePersistentVolume      = "persistentvolume"
	ResourceTypePersistentVolumeClaim = "persistentvolumeclaim"
	ResourceTypeVolumeAttachment      = "volumeattachment"

	CSIResourceTypeVolume                     = "volume"
	CSIOperationTypeCreateVolume              = "createvolume"
	CSIOperationTypeDeleteVolume              = "deletevolume"
	CSIOperationTypeControllerPublishVolume   = "controllerpublishvolume"
	CSIOperationTypeControllerUnpublishVolume = "controllerunpublishvolume"
	CSISyncMsgRespTimeout                     = 1 * time.Minute

	// ServerPort is the default port for the edgecore server on each host machine.
	// May be overridden by a flag at startup in the future.
	ServerPort = 10350

	// MessageSuccessfulContent is the successful content value of Message struct
	MessageSuccessfulContent string = "OK"
	DefaultQPS                      = 30
	DefaultBurst                    = 60
	// MaxRespBodyLength is the max length of http response body
	MaxRespBodyLength = 1 << 20 // 1 MiB
)
