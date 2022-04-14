package stream

const (
	SessionKeyHostNameOverride = "SessionHostNameOverride"
	SessionKeyInternalIP       = "SessionInternalIP"
)

const (
	MessageTypeLogsConnect MessageType = iota
	MessageTypeExecConnect
	MessageTypeMetricConnect
	MessageTypeData
	MessageTypeRemoveConnect
	MessageTypeCloseConnect
)
