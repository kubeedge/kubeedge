package constants

import (
	"time"

	v1 "k8s.io/api/core/v1"
)

const (
	DefaultConfigDir = "/etc/kubeedge/config/"
	DefaultCADir     = "/etc/kubeedge/ca/"
	DefaultCertDir   = "/etc/kubeedge/certs/"
)

// Config
const (
	DefaultKubeContentType         = "application/vnd.kubernetes.protobuf"
	DefaultKubeNamespace           = v1.NamespaceAll
	DefaultKubeQPS                 = 100.0
	DefaultKubeBurst               = 200
	DefaultKubeUpdateNodeFrequency = 20

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

	DefaultETCDTimeout = 10

	DefaultEnableElection = false
	DefaultElectionTTL    = 30
	DefaultElectionPrefix = "/controller/leader"

	DefaultMessageLayer = "context"

	DefaultContextSendModuleName     = "cloudhub"
	DefaultContextReceiveModuleName  = "edgecontroller"
	DefaultContextResponseModuleName = "cloudhub"

	DefaultPodEventBuffer       = 1
	DefaultConfigMapEventBuffer = 1
	DefaultSecretEventBuffer    = 1
	DefaultServiceEventBuffer   = 1
	DefaultEndpointsEventBuffer = 1

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
