package common

// define channel type
const (
	// MsgCtxTypeChannel message type channel
	MsgCtxTypeChannel = "channel"
	// MsgCtxTypeUS message type us
	MsgCtxTypeUS = "unixpacket"

	// ResourceTypeModule resource type module
	ResourceTypeModule = "module"
	// OperationTypeModule operation type module
	OperationTypeModule = "add"
)

// ModuleInfo is module info
type ModuleInfo struct {
	ModuleName string
	ModuleType string
	// the below field ModuleSocket is only required for using socket.
	ModuleSocket
}

type ModuleSocket struct {
	IsRemote   bool
	Connection interface{} // only for socket remote mode
}
