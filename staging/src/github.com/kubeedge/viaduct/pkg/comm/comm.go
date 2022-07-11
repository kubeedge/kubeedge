package comm

const (
	// control message type
	ControlTypeHeader = "header"
	ControlTypeConfig = "config"
	ControlTypePing   = "ping"
	ControlTypePong   = "pong"

	// control message action
	ControlActionHeader = "/control/header"
	ControlActionConfig = "/control/config"
	ControlActionPing   = "/control/ping"
	ControlActionPong   = "/control/pong"

	// response type
	RespTypeAck  = "ack"
	RespTypeNack = "nack"

	// the max size of message fifo
	MessageFiFoSizeMax = 100

	// MaxReadLength is the max length of http response body
	MaxReadLength = 1 << 20 // 1 MiB
)
