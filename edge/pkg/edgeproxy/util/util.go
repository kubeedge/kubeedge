package util

import (
	"context"
)

type EdgeProxyContextType int

const (
	RespContentType EdgeProxyContextType = iota
	ReqContentType
	AppUserAgent
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

func WithAppUserAgent(parent context.Context, ua string) context.Context {
	return context.WithValue(parent, AppUserAgent, ua)
}

func GetAppUserAgent(ctx context.Context) (string, bool) {
	ua, ok := ctx.Value(AppUserAgent).(string)
	return ua, ok
}

var (
	resourceToKind = map[string]string{
		"nodes":      "Node",
		"pods":       "Pod",
		"services":   "Service",
		"namespaces": "Namespace",
		"endpoints":  "Endpoints",
		"configmaps": "ConfigMap",
		"secrets":    "Secret",
	}
	resourceToList = map[string]string{
		"nodes":      "NodeList",
		"pods":       "PodList",
		"services":   "ServiceList",
		"namespaces": "NamespaceList",
		"endpoints":  "EndpointsList",
		"configmaps": "ConfigMapList",
		"secrets":    "SecretList",
	}
)

func GetResourceKind(resource string) string {
	return resourceToKind[resource]
}

func GetReourceList(resource string) string {
	return resourceToList[resource]
}
