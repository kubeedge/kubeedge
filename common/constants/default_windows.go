//go:build windows

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

	CloudConfigMapName = "cloudcore"

	// runtime
	DockerContainerRuntime = "docker"
	RemoteContainerRuntime = "remote"
)

// Resources
const (
	// Certificates
	DefaultConfigDir            = "c:\\etc\\kubeedge\\config\\"
	DefaultCAFile               = "c:\\etc\\kubeedge\\ca\\rootCA.crt"
	DefaultCAKeyFile            = "c:\\etc\\kubeedge\\ca\\rootCA.key"
	DefaultCertFile             = "c:\\etc\\kubeedge\\certs\\server.crt"
	DefaultKeyFile              = "c:\\etc\\kubeedge\\certs\\server.key"
	DefaultServiceAccountIssuer = "https://kubernetes.default.svc.cluster.local"

	DefaultCAURL          = "/ca.crt"
	DefaultCertURL        = "/edge.crt"
	DefaultNodeUpgradeURL = "/nodeupgrade"

	DefaultStreamCAFile   = "c:\\etc\\kubeedge\\ca\\streamCA.crt"
	DefaultStreamCertFile = "c:\\etc\\kubeedge\\certs\\stream.crt"
	DefaultStreamKeyFile  = "c:\\etc\\kubeedge\\certs\\stream.key"

	DefaultMqttCAFile   = "c:\\etc\\kubeedge\\ca\\rootCA.crt"
	DefaultMqttCertFile = "c:\\etc\\kubeedge\\certs\\server.crt"
	DefaultMqttKeyFile  = "c:\\etc\\kubeedge\\certs\\server.key"

	// Bootstrap file, contains token used by edgecore to apply for ca/cert
	BootstrapFile = "c:\\etc\\kubeedge\\bootstrap-edgecore.conf"

	// Edged
	DefaultRootDir               = "c:\\var\\lib\\edged"
	DefaultDockerAddress         = "unix:///var/run/docker.sock"
	DefaultRuntimeType           = "remote"
	DefaultDockershimRootDir     = "c:\\var\\lib\\dockershim"
	DefaultEdgedMemoryCapacity   = 7852396000
	DefaultRemoteRuntimeEndpoint = "npipe://./pipe/containerd-containerd"
	DefaultRemoteImageEndpoint   = "npipe://./pipe/containerd-containerd"
	DefaultMosquittoImage        = "eclipse-mosquitto:1.6.15"
	// update PodSandboxImage version when bumping k8s vendor version, consistent with vendor/k8s.io/kubernetes/cmd/kubelet/app/options/container_runtime.go defaultPodSandboxImageVersion
	// When this value are updated, also update comments in pkg/apis/componentconfig/edgecore/v1alpha1/types.go
	DefaultPodSandboxImage             = "kubeedge/pause:3.6"
	DefaultImagePullProgressDeadline   = time.Minute
	DefaultImageGCHighThreshold        = 80
	DefaultImageGCLowThreshold         = 40
	DefaultMaximumDeadContainersPerPod = 1
	DefaultHostnameOverride            = "default-edge-node"
	DefaultRegisterNodeNamespace       = "default"
	DefaultCNIConfDir                  = "c:\\etc\\cni\\net.d"
	DefaultCNIBinDir                   = "c:\\opt\\cni\\bin"
	DefaultCNICacheDir                 = "c:\\var\\lib\\cni\\cache"
	DefaultNetworkPluginMTU            = 1500
	DefaultConcurrentConsumers         = 5
	DefaultCgroupRoot                  = ""
	DefaultVolumeStatsAggPeriod        = time.Minute
	DefaultTunnelPort                  = 10004
	DefaultClusterDomain               = "cluster.local"
	DefaultVolumePluginDir             = "C:\\usr\\libexec\\kubernetes\\kubelet-plugins\\volume\\exec\\"

	CurrentSupportK8sVersion = "v1.24.14"

	// MetaManager
	DefaultRemoteQueryTimeout = 60
	DefaultMetaServerAddr     = "127.0.0.1:10550"

	// Config
	DefaultKubeContentType         = "application/vnd.kubernetes.protobuf"
	DefaultKubeNamespace           = v1.NamespaceAll
	DefaultKubeQPS                 = 100.0
	DefaultKubeBurst               = 200
	DefaultNodeLimit               = 500
	DefaultKubeUpdateNodeFrequency = 20

	// EdgeController
	DefaultUpdatePodStatusWorkers            = 1
	DefaultUpdateNodeStatusWorkers           = 1
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

	DefaultUpdatePodStatusBuffer            = 1024
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

	DefaultPodEventBuffer           = 1
	DefaultConfigMapEventBuffer     = 1
	DefaultSecretEventBuffer        = 1
	DefaultRulesEventBuffer         = 1
	DefaultRuleEndpointsEventBuffer = 1

	// DeviceController
	DefaultUpdateDeviceStatusBuffer  = 1024
	DefaultDeviceEventBuffer         = 1
	DefaultDeviceModelEventBuffer    = 1
	DefaultUpdateDeviceStatusWorkers = 1

	// NodeUpgradeJobController
	DefaultNodeUpgradeJobStatusBuffer = 1024
	DefaultNodeUpgradeJobEventBuffer  = 1
	DefaultNodeUpgradeJobWorkers      = 1

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
	ServerAddress = "127.0.0.1"
	ServerPort    = 10350

	// MessageSuccessfulContent is the successful content value of Message struct
	MessageSuccessfulContent string = "OK"
	DefaultQPS                      = 30
	DefaultBurst                    = 60
	// MaxRespBodyLength is the max length of http response body
	MaxRespBodyLength = 1 << 20 // 1 MiB

	EdgeNodeRoleKey   = "node-role.kubernetes.io/edge"
	EdgeNodeRoleValue = ""

	DeafultMosquittoContainerName = "mqtt-kubeedge"
	DeployMqttContainerEnv        = "DEPLOY_MQTT_CONTAINER"
)
