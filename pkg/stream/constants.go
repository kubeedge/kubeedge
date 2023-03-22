package stream

const (
	SessionKeyHostNameOverride = "SessionHostNameOverride"
	SessionKeyInternalIP       = "SessionInternalIP"
)

const (
	MessageTypeLogsConnect MessageType = iota
	MessageTypeExecConnect
	MessageTypeAttachConnect
	MessageTypeMetricConnect
	MessageTypeData
	MessageTypeRemoveConnect
	MessageTypeCloseConnect
)
