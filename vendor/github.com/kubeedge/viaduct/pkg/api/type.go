package api

const (
	// the protocol type supported
	ProtocolTypeQuic = "quic"
	ProtocolTypeWS   = "websocket"

	// connection stat
	StatConnected    = "connected"
	StatDisconnected = "disconnected"

	// connection use type
	// connection only for message
	UseTypeMessage UseType = "msg"
	// connection only for stream
	UseTypeStream UseType = "str"
	// connection only can be used for message and stream
	UseTypeShare UseType = "shr"

	// the length of use type
	UseLen = len(UseTypeMessage)
)

type UseType string
