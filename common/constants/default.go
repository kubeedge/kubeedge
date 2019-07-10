package constants

import (
	"k8s.io/api/core/v1"
)

// Config
const (
	DefaultKubeContentType         = "application/vnd.kubernetes.protobuf"
	DefaultKubeNamespace           = v1.NamespaceAll
	DefaultKubeQPS                 = 100.0
	DefaultKubeBurst               = 10
	DefaultKubeUpdateNodeFrequency = 20

	DefaultUpdatePodStatusWorkers  = 1
	DefaultUpdateNodeStatusWorkers = 1
	DefaultQueryConfigMapWorkers   = 4
	DefaultQuerySecretWorkers      = 4
	DefaultQueryServiceWorkers     = 4
	DefaultQueryEndpointsWorkers   = 4

	DefaultUpdatePodStatusBuffer  = 1024
	DefaultUpdateNodeStatusBuffer = 1024
	DefaultQueryConfigMapBuffer   = 1024
	DefaultQuerySecretBuffer      = 1024
	DefaultQueryServiceBuffer     = 1024
	DefaultQueryEndpointsBuffer   = 1024

	DefaultETCDTimeout = 10

	DefaultEnableElection = false
	DefaultElectionTTL    = 30
	DefaultElectionPrefix = "/controller/leader"

	DefaultMessageLayer = "context"

	DefaultContextSendModuleName     = "cloudhub"
	DefaultContextReceiveModuleName  = "controller"
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
)
