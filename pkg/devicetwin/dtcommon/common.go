package dtcommon

import "time"

var (
	// GroupID the name for env
	GroupID = "GROUP_ID"
	// NodeID the name for env
	NodeID = "NODE_ID"
	//SQLiteDBName the db name for save document
	SQLiteDBName = "logicdevice.db"
	// RetryTimes for retry times
	RetryTimes = 5
	// RetryInterval for retry interval
	RetryInterval = 10 * time.Second
	//LifeCycleConnectETPrefix the topic prefix for connected event
	LifeCycleConnectETPrefix = "$hw/events/connected/"
	//LifeCycleDisconnectETPrefix the topic prefix for disconnected event
	LifeCycleDisconnectETPrefix = "$hw/events/disconnected/"
	//MemETPrefix the topic prefix for membership event
	MemETPrefix = "$hw/events/node/"
	//MemETUpdateSuffix the topic suffix for membership updated event
	MemETUpdateSuffix = "/membership/updated"
	//MemETDetailSuffix the topic suffix for membership detail
	MemETDetailSuffix = "/membership/detail"
	//MemETDetailResultSuffix the topic suffix for membership detail event
	MemETDetailResultSuffix = "/membership/detail/result"
	//MemETGetSuffix the topic suffix for membership get
	MemETGetSuffix = "/membership/get"
	//MemETGetResultSuffix the topic suffix for membership get event
	MemETGetResultSuffix = "/membership/get/result"
	//DeviceETPrefix the topic prefix for device event
	DeviceETPrefix = "$hw/events/device/"
	//TwinETUpdateSuffix the topic suffix for twin update event
	TwinETUpdateSuffix = "/twin/update"
	//TwinETUpdateResultSuffix the topic suffix for twin update result event
	TwinETUpdateResultSuffix = "/twin/update/result"
	//TwinETGetSuffix the topic suffix for twin get
	TwinETGetSuffix = "/twin/get"
	//TwinETGetResultSuffix the topic suffix for twin get event
	TwinETGetResultSuffix = "/twin/get/result"
	//TwinETCloudSyncSuffix the topic suffix for twin sync event
	TwinETCloudSyncSuffix = "/twin/cloud_updated"
	//TwinETEdgeSyncSuffix the topic suffix for twin sync event
	TwinETEdgeSyncSuffix = "/twin/edge_updated"
	//DeviceETUpdatedSuffix the topic suffix for device updated event
	DeviceETUpdatedSuffix = "/updated"
	//DeviceETStateUpdateSuffix the topic suffix for device state update event
	DeviceETStateUpdateSuffix = "/state/update"
	//DeviceETStateGetSuffix the topipc suffix for device state get event
	DeviceETStateGetSuffix = "/state/get"
	//TwinETDeltaSuffix the topic suffix for twin delta event
	TwinETDeltaSuffix = "/twin/update/delta"
	//TwinETDocumentSuffix the topic suffix for twin document event
	TwinETDocumentSuffix = "/twin/update/document"
	//MemDetailResult membership detail result
	MemDetailResult = "MemDetailResult"
	//MemDetail membership detail
	MemDetail = "MemDetail"
	//MemGet get
	MemGet = "MemGet"
	//MemUpdated membership updated
	MemUpdated = "MemUpdated"
	//TwinGet get twin
	TwinGet = "TwinGet"
	//TwinUpdate twin update
	TwinUpdate = "TwinUpdate"
	//TwinCloudSync twin cloud sync
	TwinCloudSync = "TwinCloudSync"
	//TwinEdgeSync twin edge sync
	TwinEdgeSync = "TwinEdgeSync"
	//DeviceUpdated device attributes update
	DeviceUpdated = "DeviceUpdated"
	//DeviceStateGet device state get
	DeviceStateGet = "DeviceStateGet"
	//DeviceStateUpdate device state update
	DeviceStateUpdate = "DeviceStateUpdate"
	//SendToEdge send info to edge
	SendToEdge = "SendToEdge"
	//SendToCloud send info to cloud
	SendToCloud = "SendToCloud"
	//LifeCycle life cycle
	LifeCycle = "LifeCycle"
	//Connected event
	Connected = "connected"
	//Confirm event
	Confirm = "Confirm"

	//Disconnected event
	Disconnected = "disconnected"
	//CommModule communicate module
	CommModule = "CommModule"
	//DeviceModule device module
	DeviceModule = "DeviceModule"
	//MemModule membership module
	MemModule = "MemModule"
	//TwinModule twin module
	TwinModule = "TwinModule"
	//HubModule the name of hub module
	HubModule = "websocket"
	//EventHubModule the name of event hub module
	EventHubModule = "eventbus"
	//DeviceTwinModule the name of twin module
	DeviceTwinModule = "twin"

	//GetMembershipEvent event type
	GetMembershipEvent = "group_membership_event"
	//GetResult operation get_result
	GetResult = "get_result"
	//DeviceTwinEvent event type
	DeviceTwinEvent = "device_twin_event"
	//EdgeUpdated operation
	EdgeUpdated = "edge_updated"
	//Document operation
	Document = "document"
	//Delta operation
	Delta = "delta"
	//DeviceEvent event type
	DeviceEvent = "device_event"
	//UpdateResult operation
	UpdateResult = "update_result"

	//OperaMembershipResult get memebership
	OperaMembershipResult = "membership_result"

	// InsertDocument insert document
	InsertDocument = "INSERT INTO document(deviceid, deviceName, expected, actual, metadata,attributes, lastsate,versionset) values(?,?,?,?,?,?,?,?)"
	// DeleteDocument delete document
	DeleteDocument = "DELETE FROM document where deviceid = ?"

	//DealExpectedUpdateType deal expected update
	DealExpectedUpdateType = "expected"
	//DealActualUpdateType deal actual update
	DealActualUpdateType = "actual"

	//BadRequestCode bad request
	BadRequestCode = 400
	//NotFoundCode device not found
	NotFoundCode = 404
	//ConflictCode version conflict
	ConflictCode = 409
	//InternalErrorCode server internal error
	InternalErrorCode = 500
)
