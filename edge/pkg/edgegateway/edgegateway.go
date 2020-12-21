package edgegateway

import (
	"github.com/kubeedge/beehive/pkg/core"
	"github.com/kubeedge/kubeedge/edge/pkg/common/modules"
	fakekube "github.com/kubeedge/kubeedge/edge/pkg/edged/fake"
	gatewayconfig "github.com/kubeedge/kubeedge/edge/pkg/edgegateway/config"
	"github.com/kubeedge/kubeedge/edge/pkg/metamanager/client"
	"github.com/kubeedge/kubeedge/pkg/apis/componentconfig/edgecore/v1alpha1"
	clientset "k8s.io/client-go/kubernetes"
)

// edgeGateway struct
type edgeGateway struct {
	kubeClient clientset.Interface
	metaClient client.CoreInterface
	enable bool
}

func newEdgeGateway(enable bool) *edgeGateway {
	// create metaManager client
	metaClient := client.New()
	return &edgeGateway{
		metaClient: metaClient,
		kubeClient: fakekube.NewSimpleClientset(metaClient),
		enable: enable,
	}
}

// Register register edgeGateway
func Register(edgeGateway *v1alpha1.EdgeGateway,nodeName string)  {
	gatewayconfig.InitConfigure(edgeGateway,nodeName)
	core.Register(newEdgeGateway(edgeGateway.Enable))
}

//Name returns the name of EdgeGateway module
func (e edgeGateway) Name() string {
	return "edgeGateway"
}

//Group returns EdgeGateway group
func (e edgeGateway) Group() string {
	return modules.GatewayGroup
}

// Enable indicates whether this module is enabled
func (e edgeGateway) Enable() bool {
	return e.enable
}

//Start sets context and starts the controller
func (e edgeGateway) Start() {
	// edgeDiscovery



}

// proxy API
func ProxyInterface()  {


}
