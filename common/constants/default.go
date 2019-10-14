package constants

import (
	"time"

	v1 "k8s.io/api/core/v1"
)

// Config
const (
	DefaultKubeContentType         = "application/vnd.kubernetes.protobuf"
	DefaultKubeNamespace           = v1.NamespaceAll
	DefaultKubeQPS                 = 100.0
	DefaultKubeBurst               = 10
	DefaultKubeUpdateNodeFrequency = 20
)

const (
	// DefaultUpdatePodStatusWorkers is the count of goroutines of update pod status
	DefaultUpdatePodStatusWorkers = 1
	// DefaultUpdateNodeStatusWorkers is the count of goroutines of update node status
	DefaultUpdateNodeStatusWorkers = 1
	// DefaultQueryConfigMapWorkers is the count of goroutines of query configmap
	DefaultQueryConfigMapWorkers = 4
	// DefaultQuerySecretWorkers is the count of goroutines of query secret
	DefaultQuerySecretWorkers = 4
	// DefaultQueryServiceWorkers is the count of goroutines of query service
	DefaultQueryServiceWorkers = 4
	// DefaultQueryEndpointsWorkers is the count of goroutines of query endpoints
	DefaultQueryEndpointsWorkers = 4
	// DefaultQueryPersistentVolumeWorkers is the count of goroutines of query persistentvolume
	DefaultQueryPersistentVolumeWorkers = 4
	// DefaultQueryPersistentVolumeClaimWorkers is the count of goroutines of query persistentvolumeclaim
	DefaultQueryPersistentVolumeClaimWorkers = 4
	// DefaultQueryVolumeAttachmentWorkers is the count of goroutines of query volumeattachment
	DefaultQueryVolumeAttachmentWorkers = 4
	// DefaultQueryNodeWorkers is the count of goroutines of query node
	DefaultQueryNodeWorkers = 4
)

const (
	// DefaultUpdatePodStatusBuffer is the size of channel which save update pod status message from edge
	DefaultUpdatePodStatusBuffer = 1024
	// DefaultUpdateNodeStatusBuffer is the size of channel which save update node status message from edge
	DefaultUpdateNodeStatusBuffer = 1024
	// DefaultQueryConfigMapBuffer is the size of channel which save query configmap message from edge
	DefaultQueryConfigMapBuffer = 1024
	// DefaultQuerySecretBuffer is the size of channel which save query secret message from edge
	DefaultQuerySecretBuffer = 1024
	// DefaultQueryServiceBuffer is the size of channel which save query service message from edge
	DefaultQueryServiceBuffer = 1024
	// DefaultQueryEndpointsBuffer is the size of channel which save query endpoints message from edge
	DefaultQueryEndpointsBuffer = 1024
	// DefaultQueryPersistentVolumeBuffer is the size of channel which save query persistentvolume message from edge
	DefaultQueryPersistentVolumeBuffer = 1024
	// DefaultQueryPersistentVolumeClaimBuffer is the size of channel which save query persistentvolumeclaim message from edge
	DefaultQueryPersistentVolumeClaimBuffer = 1024
	// DefaultQueryVolumeAttachmentBuffer is the size of channel which save query volumeattachment message from edge
	DefaultQueryVolumeAttachmentBuffer = 1024
	// DefaultQueryNodeBuffer is the size of channel which save query node message from edge
	DefaultQueryNodeBuffer = 1024
	// DefaultUpdateNodeBuffer is the size of channel which save update node message from edge
	DefaultUpdateNodeBuffer = 1024

	// DefaultPodEventBuffer is the size of channel which save pod event from k8s
	DefaultPodEventBuffer = 1
	// DefaultConfigMapEventBuffer is the size of channel which save configmap event from k8s
	DefaultConfigMapEventBuffer = 1
	// DefaultSecretEventBuffer is the size of channel which save secret event from k8s
	DefaultSecretEventBuffer = 1
	// DefaultServiceEventBuffer is the size of channel which save service event from k8s
	DefaultServiceEventBuffer = 1
	// DefaultEndpointsEventBuffer is the size of channel which save endpoints event from k8s
	DefaultEndpointsEventBuffer = 1
)

const (
	// Resource sep
	ResourceSep = "/"

	ResourceTypeService       = "service"
	ResourceTypeServiceList   = "servicelist"
	ResourceTypeEndpoints     = "endpoints"
	ResourceTypeEndpointsList = "endpointslist"

	ResourceTypePersistentVolume      = "persistentvolume"
	ResourceTypePersistentVolumeClaim = "persistentvolumeclaim"
	ResourceTypeVolumeAttachment      = "volumeattachment"

	CSIResourceTypeVolume                     = "volume"
	CSIOperationTypeCreateVolume              = "createvolume"
	CSIOperationTypeDeleteVolume              = "deletevolume"
	CSIOperationTypeControllerPublishVolume   = "controllerpublishvolume"
	CSIOperationTypeControllerUnpublishVolume = "controllerunpublishvolume"
	CSISyncMsgRespTimeout                     = 1 * time.Minute

	CurrentSupportK8sVersion = "v1.15.3"
)

const (
	DefaultConfigDir = "/etc/kubeedge/config/"
	DefaultCADir     = "/etc/kubeedge/ca/"
	DefaultCertDir   = "/etc/kubeedge/certs/"
)

const (
	ProtocolWebsocket = "websocket"
	ProtocolQuic      = "quic"
)

const (
	RuntimeTypeDocker = "docker"
	RuntimeTypeRemote = "remote"
)
const (
	StrategyRoundRobin = "RoundRobin"
)

const (
	// DefaultUpdateDeviceStatusBuffer is the size of channel which save update device status message from edge
	DefaultUpdateDeviceStatusBuffer = 1024
	// DefaultDeviceEventBuffer is the size of channel which save device event from k8s
	DefaultDeviceEventBuffer = 1
	// DefaultDeviceModelEventBuffer is the size of channel which save devicemodel event from k8s
	DefaultDeviceModelEventBuffer = 1
	// DefaultUpdateDeviceStatusWorkers is the count of goroutines of update device status
	DefaultUpdateDeviceStatusWorkers = 1
)
