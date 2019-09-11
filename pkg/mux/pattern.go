package mux

import (
	"strings"

	"k8s.io/klog"

	"github.com/kubeedge/beehive/pkg/core/model"
)

type MessagePattern struct {
	resource  string
	operation string
	resExpr   *MessageExpression
}

func NewPattern(resource string) *MessagePattern {
	expression := NewExpression()
	resExpr := expression.GetExpression(resource)
	if resExpr == nil {
		klog.Errorf("bad resource(%s) for expression", resource)
		return nil
	}

	return &MessagePattern{
		resource: resource,
		resExpr:  resExpr,
	}
}

func (pattern *MessagePattern) Res(resource string) *MessagePattern {
	pattern.resource = resource
	return pattern
}

func (pattern *MessagePattern) Op(operation string) *MessagePattern {
	pattern.operation = operation
	return pattern
}

func (pattern *MessagePattern) matchOp(message *model.Message) bool {
	return strings.Compare(pattern.operation, message.GetOperation()) == 0 ||
		strings.Compare(pattern.operation, "*") == 0
}

func (pattern *MessagePattern) Match(message *model.Message) bool {
	return pattern.resExpr.Matcher.Match([]byte(message.GetResource())) && pattern.matchOp(message)
}
