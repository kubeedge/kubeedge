package config

import (
	"github.com/kubeedge/kubeedge/common/constants"
)

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

	QueryPersistentVolumeWorkers = constants.DefaultQueryPersistentVolumeWorkers

	QueryPersistentVolumeClaimWorkers = constants.DefaultQueryPersistentVolumeClaimWorkers

	QueryVolumeAttachmentWorkers = constants.DefaultQueryVolumeAttachmentWorkers
	QueryNodeWorkers = constants.DefaultQueryNodeWorkers

	UpdateNodeWorkers = constants.DefaultUpdateNodeWorkers
}
