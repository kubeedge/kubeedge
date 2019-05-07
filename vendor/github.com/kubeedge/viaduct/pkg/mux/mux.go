package mux

import (
	"fmt"
	"net/http"

	"github.com/kubeedge/beehive/pkg/core/model"
	"github.com/kubeedge/viaduct/pkg/filter"
)

type ResponseWriter interface {
	WriteResponse(msg *model.Message, content interface{})
	WriteError(msg *model.Message, err string)
}

type Handler interface {
	ServeConn(req *MessageRequest, writer ResponseWriter)
}

type MessageRequest struct {
	Header  http.Header
	Message *model.Message
}

type MessageContainer struct {
	Header     http.Header
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

func (mux *MessageMux) wrapMessage(header http.Header, msg *model.Message, params map[string]string) *MessageContainer {
	return &MessageContainer{
		Message:    msg,
		parameters: params,
		Header:     header,
	}
}

func (mux *MessageMux) dispatch(req *MessageRequest, writer ResponseWriter) error {
	for _, entry := range mux.muxEntry {
		// select entry
		matched := entry.pattern.Match(req.Message)
		if !matched {
			continue
		}

		// extract parameters
		parameters := mux.extractParameters(entry.pattern.resExpr, req.Message.GetResource())
		// wrap message
		container := mux.wrapMessage(req.Header, req.Message, parameters)
		// call user handle of entry
		entry.handleFunc(container, writer)
		return nil
	}
	return fmt.Errorf("failed to found entry for message")
}

func (mux *MessageMux) AddFilter(filter *filter.MessageFilter) {
	mux.filter = filter
}

func (mux *MessageMux) processFilter(req *MessageRequest) error {
	if mux.filter != nil {
		return mux.filter.ProcessFilter(req.Message)
	}
	return nil
}

func (mux *MessageMux) ServeConn(req *MessageRequest, writer ResponseWriter) {
	err := mux.processFilter(req)
	if err != nil {
		return
	}
	mux.dispatch(req, writer)
}

func Entry(pattern *MessagePattern, handle func(*MessageContainer, ResponseWriter)) *MessageMux {
	MuxDefault.Entry(pattern, handle)
	return MuxDefault
}
