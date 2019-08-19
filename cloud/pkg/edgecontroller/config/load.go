package config

import (
	"k8s.io/klog"

	"github.com/kubeedge/beehive/pkg/common/config"
	"github.com/kubeedge/kubeedge/common/constants"
)

// UpdatePodStatusWorkers is the count of goroutines of update pod status
var UpdatePodStatusWorkers int

// UpdateNodeStatusWorkers is the count of goroutines of update node status
var UpdateNodeStatusWorkers int

// QueryConfigMapWorkers is the count of goroutines of query configmap
var QueryConfigMapWorkers int

// QuerySecretWorkers is the count of goroutines of query secret
var QuerySecretWorkers int

// QueryServiceWorkers is the count of goroutines of query service
var QueryServiceWorkers int

// QueryEndpointsWorkers is the count of goroutines of query endpoints
var QueryEndpointsWorkers int

// QueryPersistentVolumeWorkers is the count of goroutines of query persistentvolume
var QueryPersistentVolumeWorkers int

// QueryPersistentVolumeClaimWorkers is the count of goroutines of query persistentvolumeclaim
var QueryPersistentVolumeClaimWorkers int

// QueryVolumeAttachmentWorkers is the count of goroutines of query volumeattachment
var QueryVolumeAttachmentWorkers int

// QueryNodeWorkers is the count of goroutines of query node
var QueryNodeWorkers int

// UpdateNodeWorkers is the count of goroutines of update node
var UpdateNodeWorkers int

func InitLoadConfig() {
	if psw, err := config.CONFIG.GetValue("controller.load.update-pod-status-workers").ToInt(); err != nil {
		UpdatePodStatusWorkers = constants.DefaultUpdatePodStatusWorkers
	} else {
		UpdatePodStatusWorkers = psw
	}
	klog.Infof("update pod status workers: %d", UpdatePodStatusWorkers)

	if nsw, err := config.CONFIG.GetValue("controller.load.update-node-status-workers").ToInt(); err != nil {
		UpdateNodeStatusWorkers = constants.DefaultUpdateNodeStatusWorkers
	} else {
		UpdateNodeStatusWorkers = nsw
	}
	klog.Infof("update node status workers: %d", UpdateNodeStatusWorkers)

	if qcw, err := config.CONFIG.GetValue("controller.load.query-configmap-workers").ToInt(); err != nil {
		QueryConfigMapWorkers = constants.DefaultQueryConfigMapWorkers
	} else {
		QueryConfigMapWorkers = qcw
	}
	klog.Infof("query config map workers: %d", QueryConfigMapWorkers)

	if qsw, err := config.CONFIG.GetValue("controller.load.query-secret-workers").ToInt(); err != nil {
		QuerySecretWorkers = constants.DefaultQuerySecretWorkers
	} else {
		QuerySecretWorkers = qsw
	}
	klog.Infof("query secret workers: %d", QuerySecretWorkers)

	if qsw, err := config.CONFIG.GetValue("controller.load.query-service-workers").ToInt(); err != nil {
		QueryServiceWorkers = constants.DefaultQueryServiceWorkers
	} else {
		QueryServiceWorkers = qsw
	}
	klog.Infof("query service workers: %d", QueryServiceWorkers)

	if qew, err := config.CONFIG.GetValue("controller.load.query-endpoints-workers").ToInt(); err != nil {
		QueryEndpointsWorkers = constants.DefaultQueryEndpointsWorkers
	} else {
		QueryEndpointsWorkers = qew
	}
	klog.Infof("query endpoints workers: %d", QueryEndpointsWorkers)

	if qpvw, err := config.CONFIG.GetValue("controller.load.query-persistentvolume-workers").ToInt(); err != nil {
		QueryPersistentVolumeWorkers = constants.DefaultQueryPersistentVolumeWorkers
	} else {
		QueryPersistentVolumeWorkers = qpvw
	}
	klog.Infof("query persistentvolume workers: %d", QueryPersistentVolumeWorkers)

	if qpvcw, err := config.CONFIG.GetValue("controller.load.query-persistentvolumeclaim-workers").ToInt(); err != nil {
		QueryPersistentVolumeClaimWorkers = constants.DefaultQueryPersistentVolumeClaimWorkers
	} else {
		QueryPersistentVolumeClaimWorkers = qpvcw
	}
	klog.Infof("query persistentvolumeclaim workers: %d", QueryPersistentVolumeClaimWorkers)

	if qvaw, err := config.CONFIG.GetValue("controller.load.query-volumeattachment-workers").ToInt(); err != nil {
		QueryVolumeAttachmentWorkers = constants.DefaultQueryVolumeAttachmentWorkers
	} else {
		QueryVolumeAttachmentWorkers = qvaw
	}
	klog.Infof("query volumeattachment workers: %d", QueryVolumeAttachmentWorkers)

	if qnw, err := config.CONFIG.GetValue("controller.load.query-node-workers").ToInt(); err != nil {
		QueryNodeWorkers = constants.DefaultQueryNodeWorkers
	} else {
		QueryNodeWorkers = qnw
	}
	klog.Infof("query node workers: %d", QueryNodeWorkers)

	if unw, err := config.CONFIG.GetValue("controller.load.update-node-workers").ToInt(); err != nil {
		UpdateNodeWorkers = constants.DefaultUpdateNodeWorkers
	} else {
		UpdateNodeWorkers = unw
	}
	klog.Infof("update node workers: %d", UpdateNodeWorkers)
}
