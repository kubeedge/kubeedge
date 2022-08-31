package constants

import "time"

const (
	AppHandler        = "/api/v1/namespaces/default/pods"
	DeploymentHandler = "/apis/apps/v1/namespaces/default/deployments"

	Interval = 5 * time.Second
	Timeout  = 10 * time.Minute
)
