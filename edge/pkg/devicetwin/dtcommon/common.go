package dtcommon

import (
	"time"

	"github.com/kubeedge/kubeedge/cloud/pkg/apis/devices/v1alpha2"
	"github.com/kubeedge/kubeedge/cloud/pkg/devicecontroller/constants"
)

const (
	// RetryTimes for retry times
	RetryTimes = 5
	// RetryInterval for retry interval
	RetryInterval = 10 * time.Second

	// LifeCycleConnectETPrefix the topic prefix for connected event
	LifeCycleConnectETPrefix = "$hw/events/connected/"
	// LifeCycleDisconnectETPrefix the topic prefix for disconnected event
	LifeCycleDisconnectETPrefix = "$hw/events/disconnected/"

	// MemETPrefix the topic prefix for membership event
	MemETPrefix = "$hw/events/node/"
	// MemETAddSuffix the topic suffix for membership added event
	MemETAddSuffix = "/membership/added"
	// MemETDeleteSuffix the topic suffix for membership deleted event
	MemETDeleteSuffix = "/membership/deleted"
	// MemETDetailSuffix the topic suffix for membership detail
	MemETDetailSuffix = "/membership/detail"
	// MemETDetailResultSuffix the topic suffix for membership detail event
	MemETDetailResultSuffix = "/membership/detail/result"
	// MemETGetSuffix the topic suffix for membership get
	MemETGetSuffix = "/membership/get"
	// MemETGetResultSuffix the topic suffix for membership get event
	MemETGetResultSuffix = "/membership/get/result"

	// DeviceETPrefix the topic prefix for device event
	DeviceETPrefix = "$hw/events/device/"
	// TwinETUpdateSuffix the topic suffix for twin update event
	TwinETUpdateSuffix = "/twin/update"
	// TwinETUpdateResultSuffix the topic suffix for twin update result event
	TwinETUpdateResultSuffix = "/twin/update/result"
	// TwinETGetSuffix the topic suffix for twin get
	TwinETGetSuffix = "/twin/get"
	// TwinETGetResultSuffix the topic suffix for twin get event
	TwinETGetResultSuffix = "/twin/get/result"
	// TwinETCloudSyncSuffix the topic suffix for twin sync event
	TwinETCloudSyncSuffix = "/twin/cloud_updated"
	// TwinETEdgeSyncSuffix the topic suffix for twin sync event
	TwinETEdgeSyncSuffix = "/twin/edge_updated"
	// TwinETDeltaSuffix the topic suffix for twin delta event
	TwinETDeltaSuffix = "/twin/update/delta"
	// TwinETDocumentSuffix the topic suffix for twin document event
	TwinETDocumentSuffix = "/twin/update/document"

	// MemDetail membership detail
	MemDetail = "MemDetail"
	// MemGet get
	MemGet = "MemGet"
	// MemAdded membership added
	MemAdded = "MemAdded"
	// MemDeleted membership deleted
	MemDeleted = "MemDeleted"

	// TwinGet get twin
	TwinGet = "TwinGet"
	// TwinUpdate twin update
	TwinUpdate = "TwinUpdate"
	// TwinCloudSync twin cloud sync
	TwinCloudSync = "TwinCloudSync"
	// TwinEdgeSync twin edge sync
	TwinEdgeSync = "TwinEdgeSync"

	// SendToEdge send info to edge
	SendToEdge = "SendToEdge"
	// SendToCloud send info to cloud
	SendToCloud = "SendToCloud"
	// LifeCycle life cycle
	LifeCycle = "LifeCycle"
	// Connected event
	Connected = "connected"
	// Confirm event
	Confirm = "Confirm"
	// Disconnected event
	Disconnected = "disconnected"

	// CommModule communicate module
	CommModule = "CommModule"
	// DeviceModule device module
	DeviceModule = "DeviceModule"
	// MemModule membership module
	MemModule = "MemModule"
	// TwinModule twin module
	TwinModule = "TwinModule"
	// HubModule the name of hub module
	HubModule = "websocket"
	// EventHubModule the name of event hub module
	EventHubModule = "eventbus"
	// DeviceTwinModule the name of twin module
	DeviceTwinModule = "twin"

	// BadRequestCode bad request
	BadRequestCode = 400
	// NotFoundCode device not found
	NotFoundCode = 404
	// ConflictCode version conflict
	ConflictCode = 409
	// InternalErrorCode server internal error
	InternalErrorCode = 500
)

// generateDeviceID use name + namespace as unique key
func GenerateDeviceID(device *v1alpha2.Device) string {
	return device.Namespace + constants.DeviceUniqueKeySeperator + device.Name
}
