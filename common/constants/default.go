package constants

import (
	"time"

	v1 "k8s.io/api/core/v1"
)

// Module name and group name
const (
	// SyncController
	DefaultContextSendModuleName = "cloudhub"
	ProjectName                  = "KubeEdge"
	SystemName                   = "kubeedge"
	SystemNamespace              = SystemName
	CloudConfigMapName           = "cloudcore"
	EdgeMappingCloudKey          = "cloudcore"

	// runtime
	DockerContainerRuntime = "docker"
	RemoteContainerRuntime = "remote"
)

// Resources
const (
	DefaultCAURL              = "/ca.crt"
	DefaultCertURL            = "/edge.crt"
	DefaultCheckNodeURL       = "/node/{nodename}"
	DefaultNodeUpgradeURL     = "/nodeupgrade"
	DefaultTaskStateReportURL = "/task/{taskType}/name/{taskID}/node/{nodeID}/status"

	// update PodSandboxImage version when bumping k8s vendor version, consistent with vendor/k8s.io/kubernetes/cmd/kubelet/app/options/container_runtime.go defaultPodSandboxImageVersion
	// When this value are updated, also update comments in pkg/apis/componentconfig/edgecore/v1alpha1/types.go
	DefaultHostnameOverride = "default-edge-node"

	CurrentSupportK8sVersion = "v1.30.7"

	// MetaManager
	DefaultMetaServerAddr = "127.0.0.1:10550"

	// Config
	DefaultKubeContentType         = "application/vnd.kubernetes.protobuf"
	DefaultKubeNamespace           = v1.NamespaceAll
	DefaultKubeQPS                 = 100.0
	DefaultKubeBurst               = 200
	DefaultNodeLimit               = 500
	DefaultKubeUpdateNodeFrequency = 20

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

	ServerAddress = "127.0.0.1"
	// ServerPort is the default port for the edgecore server on each host machine.
	// May be overridden by a flag at startup in the future.
	ServerPort = 10350

	// MessageSuccessfulContent is the successful content value of Message struct
	MessageSuccessfulContent string = "OK"
	// MaxRespBodyLength is the max length of http response body
	MaxRespBodyLength = 1 << 20 // 1 MiB

	EdgeNodeRoleKey   = "node-role.kubernetes.io/edge"
	EdgeNodeRoleValue = ""

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
