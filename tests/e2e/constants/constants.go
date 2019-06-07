package constants

const (
	RunController = "cd ${GOPATH}/src/github.com/kubeedge/kubeedge/cloud/; sudo nohup ./edgecontroller > edgecontroller.log 2>&1 &"
	RunEdgecore   = "cd ${GOPATH}/src/github.com/kubeedge/kubeedge/edge/; sudo nohup ./edge_core > edge_core.log 2>&1 &"
	RunEdgeSite   = "cd ${GOPATH}/src/github.com/kubeedge/kubeedge/edgesite/; sudo nohup ./edgesite > edgesite.log 2>&1 &"

	AppHandler        = "/api/v1/namespaces/default/pods"
	NodeHandler       = "/api/v1/nodes"
	DeploymentHandler = "/apis/apps/v1/namespaces/default/deployments"
)