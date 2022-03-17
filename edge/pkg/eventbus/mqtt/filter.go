package mqtt

import (
	"bytes"
	"fmt"
	"regexp"
	"strings"

	"k8s.io/klog/v2"
)

type HandlerFunc func(topic string, payload []byte)

// MessageMuxEntry message mux entry
type MessageMuxEntry struct {
	pattern     *MessagePattern
	handlerFunc HandlerFunc
}

// MessagePattern message pattern
type MessagePattern struct {
	resExpr   *MessageExpression
	resource  string
	operation string
}

// MessageExpression message expression
type MessageExpression struct {
	Matcher  *regexp.Regexp
	VarNames []string
	VarCount int
}

// MessageMux message mux
type MessageMux struct {
	muxEntry []*MessageMuxEntry
}

// NewExpression new expression
func NewExpression() *MessageExpression {
	return &MessageExpression{}
}

// GetExpression get expression
func (exp *MessageExpression) GetExpression(topic string) *MessageExpression {
	var buffer bytes.Buffer
	var varNames []string
	var varCount int
	buffer.WriteString("^")
	if strings.HasPrefix(topic, "/") {
		buffer.WriteString("/")
	}
	fields := strings.Split(strings.Trim(topic, "/"), "/")
	for _, field := range fields {
		if field == "" {
			continue
		}
		if strings.HasPrefix(field, "{") {
			colon := strings.Index(field, ":")
			var varName string
			if colon != -1 {
				varName = strings.TrimSpace(field[1:colon])
				paramExpr := strings.TrimSpace(field[colon+1 : len(field)-1])
				if paramExpr == "*" { // special case
					buffer.WriteString("(.*)")
				} else {
					buffer.WriteString(fmt.Sprintf("(%s)", paramExpr))
				}
			} else {
				varName = strings.TrimSpace(field[1 : len(field)-1])
				buffer.WriteString("([^/]+?)")
			}
			varNames = append(varNames, varName)
			varCount++
		} else {
			buffer.WriteString(regexp.QuoteMeta(field))
		}
		buffer.WriteString("/")
	}

	expression := strings.TrimRight(buffer.String(), "/") + "(/.*)?$"
	compiled, err := regexp.Compile(expression)
	if err != nil {
		klog.Errorf("failed to compile resource, error: %+v", err)
		return nil
	}

	return &MessageExpression{
		Matcher:  compiled,
		VarNames: varNames,
		VarCount: varCount,
	}
}

var mux MessageMux

// NewMessageMux new message mux
func NewMessageMux() *MessageMux {
	return &mux
}

// NewPattern new pattern
func NewPattern(resource string) *MessagePattern {
	expression := NewExpression()
	resExpr := expression.GetExpression(resource)
	if resExpr == nil {
		klog.Errorf("bad resource for expression: %s", resource)
		return nil
	}

	return &MessagePattern{
		resource: resource,
		resExpr:  resExpr,
	}
}

// NewEntry new entry
func NewEntry(pattern *MessagePattern, handle func(topic string, payload []byte)) *MessageMuxEntry {
	return &MessageMuxEntry{
		pattern:     pattern,
		handlerFunc: handle,
	}
}

// Match /path/{param}/sub
func (pattern *MessagePattern) Match(topic string) bool {
	return pattern.resExpr.Matcher.Match([]byte(topic))
}

// Entry mux := NewMessageMux(ctx, module)
// mux.Entry(NewPattern(res).Op(opr), handle))
func (mux *MessageMux) Entry(pattern *MessagePattern, handle func(topic string, payload []byte)) *MessageMux {
	entry := NewEntry(pattern, handle)
	mux.muxEntry = append(mux.muxEntry, entry)
	return mux
}

func (mux *MessageMux) Dispatch(topic string, payload []byte) {
	for _, entry := range mux.muxEntry {
		matched := entry.pattern.Match(topic)
		if !matched {
			continue
		}
		entry.handlerFunc(topic, payload)
		return
	}
	handleUploadTopic(topic, payload)
}

// RegisterMsgHandler register handler for message if topic is matched in pattern
// for "$hw/events/device/+/twin/+", "$hw/events/node/+/membership/get", send to twin
// for other, send to hub
// for "SYS/dis/upload_records", no need to base64 topic
func RegisterMsgHandler() {
	mux.Entry(NewPattern("$hw/events/device/"), handleDeviceTwin)
	mux.Entry(NewPattern("$hw/events/node/"), handleDeviceTwin)
	mux.Entry(NewPattern("SYS/dis/upload_records"), handleUploadTopic)
}
