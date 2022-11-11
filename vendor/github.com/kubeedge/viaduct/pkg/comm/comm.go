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
)
