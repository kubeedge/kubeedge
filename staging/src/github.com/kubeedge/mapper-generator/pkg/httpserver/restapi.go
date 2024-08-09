package httpserver

// API path
const (
	// APIVersion description API version
	APIVersion = "v1"
	// APIBase to build RESTful API
	APIBase = "/api/" + APIVersion

	// APIPing ping API that get server status
	APIPing = APIBase + "/ping"

	// APIDeviceRoute to build device RESTful API
	APIDeviceRoute = APIBase + "/device"
	// APIDeviceReadRoute API that read device's property
	APIDeviceReadRoute = APIDeviceRoute + "/" + DeviceID + "/" + PropertyName

	// APIMetaRoute to build meta RESTful API
	APIMetaRoute = APIBase + "/meta"
	// APIMetaGetModelRoute API that get device model by device id
	APIMetaGetModelRoute = APIMetaRoute + "/model" + "/" + DeviceID

	// APIDataBaseRoute to build database RESTful API
	APIDataBaseRoute = APIBase + "/database"
	// APIDataBaseGetDataByID API that get data by device id
	APIDataBaseGetDataByID = APIDataBaseRoute + "/" + DeviceID
)

// API field pattern
const (
	// DeviceID pattern for deviceID
	DeviceID = "{id}"
	// PropertyName pattern for property
	PropertyName = "{property}"
)

// Response Header
const (
	// ContentType content header Key
	ContentType = "Content-Type"
	// ContentTypeJSON content type is json
	ContentTypeJSON = "application/json"

	// CorrelationHeader correlation header key
	CorrelationHeader = "X-Correlation-ID"
)
