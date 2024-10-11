package mux

type HandlerFunc func(*MessageContainer, ResponseWriter)

type MessageMuxEntry struct {
	pattern    *MessagePattern
	handleFunc HandlerFunc
}

func NewEntry(pattern *MessagePattern, handle func(*MessageContainer, ResponseWriter)) *MessageMuxEntry {
	return &MessageMuxEntry{
		pattern:    pattern,
		handleFunc: handle,
	}
}

func (entry *MessageMuxEntry) Pattern(pattern *MessagePattern) *MessageMuxEntry {
	entry.pattern = pattern
	return entry
}

func (entry *MessageMuxEntry) Handle(handle func(*MessageContainer, ResponseWriter)) *MessageMuxEntry {
	entry.handleFunc = handle
	return entry
}
