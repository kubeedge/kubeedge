package util

import (
	"context"
	"fmt"

	"github.com/kubeedge/kubeedge/edge/pkg/lwproxy/relation"
	"k8s.io/klog"
)

type LWProxyContextType int

const (
	RespContentType LWProxyContextType = iota
	ReqContentType
	AppUserAgent
	RespContentEncoding
)

func WithRespContentType(parent context.Context, contentType string) context.Context {
	return context.WithValue(parent, RespContentType, contentType)
}

func GetRespContentType(ctx context.Context) (string, bool) {
	resp, ok := ctx.Value(RespContentType).(string)
	return resp, ok
}

func WithReqContentType(parent context.Context, contentType string) context.Context {
	return context.WithValue(parent, ReqContentType, contentType)
}

func GetReqContentType(parent context.Context) (string, bool) {
	req, ok := parent.Value(ReqContentType).(string)
	return req, ok
}

func WithRespContentEncoding(parent context.Context, algo string) context.Context {
	return context.WithValue(parent, RespContentEncoding, algo)
}

func GetRespContentEncoding(parent context.Context) (string, bool) {
	req, ok := parent.Value(RespContentEncoding).(string)
	return req, ok
}

func WithAppUserAgent(parent context.Context, ua string) context.Context {
	return context.WithValue(parent, AppUserAgent, ua)
}

func GetAppUserAgent(ctx context.Context) (string, bool) {
	ua, ok := ctx.Value(AppUserAgent).(string)
	return ua, ok
}

func GetResourceKind(gr string) string {
	o := relation.GetRelation(gr)
	kind := o.GetKind()
	if kind == "" {
		klog.Warningf("can not find kind of %s", gr)
	}
	return kind
}

func GetReourceList(gr string) string {
	o := relation.GetRelation(gr)
	list := o.GetList()
	if list == "" {
		klog.Warningf("can not find list kind of %s", gr)
	}
	return list
}

func BuildGroupResource(resource, group string) string {
	if group == "" {
		return resource
	}
	return fmt.Sprintf("%s.%s", resource, group)
}
