package mux

import (
	"fmt"

	"github.com/kubeedge/beehive/pkg/core/model"
	"github.com/kubeedge/viaduct/pkg/filter"
)

type ResponseWriter interface {
	WriteResponse(msg *model.Message, content interface{})
	WriteError(msg *model.Message, err string)
}

type Handler interface {
	ServeConn(msg *model.Message, writer ResponseWriter)
}

type MessageContainer struct {
	Message    *model.Message
	parameters map[string]string
}

type MessageMux struct {
	filter   *filter.MessageFilter
	muxEntry []*MessageMuxEntry
}

var (
	MuxDefault = NewMessageMux()
)

func (c *MessageContainer) Parameter(name string) string {
	return c.parameters[name]
}

func NewMessageMux() *MessageMux {
	return &MessageMux{}
}

func (mux *MessageMux) Entry(pattern *MessagePattern, handle func(*MessageContainer, ResponseWriter)) *MessageMux {
	entry := NewEntry(pattern, handle)
	mux.muxEntry = append(mux.muxEntry, entry)
	return mux
}

func (mux *MessageMux) extractParameters(expression *MessageExpression, resource string) map[string]string {
	parameters := make(map[string]string)
	matches := expression.Matcher.FindStringSubmatch(resource)
	for index := 1; index < len(matches); index++ {
		if len(expression.VarNames) >= index {
			parameters[expression.VarNames[index-1]] = matches[index]
		}
	}
	return parameters
}

func (mux *MessageMux) wrapMessage(msg *model.Message, params map[string]string) *MessageContainer {
	return &MessageContainer{
		Message:    msg,
		parameters: params,
	}
}

func (mux *MessageMux) dispatch(msg *model.Message, writer ResponseWriter) error {
	for _, entry := range mux.muxEntry {
		// select entry
		matched := entry.pattern.Match(msg)
		if !matched {
			continue
		}
		// extract parameters
		parameters := mux.extractParameters(entry.pattern.resExpr, msg.GetResource())
		// wrap message
		container := mux.wrapMessage(msg, parameters)
		// call user handle of entry
		entry.handleFunc(container, writer)
		return nil
	}
	return fmt.Errorf("failed to found entry for message")
}

func (mux *MessageMux) AddFilter(filter *filter.MessageFilter) {
	mux.filter = filter
}

func (mux *MessageMux) processFilter(msg *model.Message) error {
	if mux.filter != nil {
		return mux.filter.ProcessFilter(msg)
	}
	return nil
}

func (mux *MessageMux) ServeConn(msg *model.Message, writer ResponseWriter) {
	err := mux.processFilter(msg)
	if err != nil {
		return
	}
	mux.dispatch(msg, writer)
}

func Entry(pattern *MessagePattern, handle func(*MessageContainer, ResponseWriter)) *MessageMux {
	MuxDefault.Entry(pattern, handle)
	return MuxDefault
}
