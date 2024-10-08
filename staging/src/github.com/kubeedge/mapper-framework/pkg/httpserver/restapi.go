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
	// APIDeviceMethodRoute to build deviceMethod RESTful API
	APIDeviceMethodRoute = APIBase + "/devicemethod"
	// APIGetDeviceMethodRoute to get all deviceMethod of the device
	APIGetDeviceMethodRoute = APIDeviceMethodRoute + "/" + DeviceNamespace + "/" + DeviceName
	// APIDeviceReadRoute API that read device's property
	APIDeviceReadRoute = APIDeviceRoute + "/" + DeviceNamespace + "/" + DeviceName + "/" + PropertyName
	// APIDeviceWriteRoute API that read device's property
	APIDeviceWriteRoute = APIDeviceMethodRoute + "/" + DeviceNamespace + "/" + DeviceName + "/" + DeviceMethodName + "/" + PropertyName + "/" + Data

	// APIMetaRoute to build meta RESTful API
	APIMetaRoute = APIBase + "/meta"
	// APIMetaGetModelRoute API that get device model by device id
	APIMetaGetModelRoute = APIMetaRoute + "/model" + "/" + DeviceNamespace + "/" + DeviceName

	// APIDataBaseRoute to build database RESTful API
	APIDataBaseRoute = APIBase + "/database"
	// APIDataBaseGetDataByID API that get data by device id
	APIDataBaseGetDataByID = APIDataBaseRoute + "/" + DeviceNamespace + "/" + DeviceName
)

// API field pattern
const (
	// DeviceName pattern for deviceName
	DeviceName = "{name}"
	// DeviceMethodName pattern for device method Name
	DeviceMethodName = "{methodname}"
	// Data pattern for device write data
	Data = "{data}"
	// DeviceNamespace pattern for deviceNamespace
	DeviceNamespace = "{namespace}"
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
