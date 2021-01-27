package config
import (
	"github.com/kubeedge/kubeedge/pkg/apis/componentconfig/edgecore/v1alpha1"
	"sync"
)

var Config Configure
var once sync.Once

type Configure struct {
	v1alpha1.EdgeGateway
	NodeName string
}

func InitConfigure(edgeGateway *v1alpha1.EdgeGateway ,nodeName string)  {
	once.Do(func() {
		Config = Configure{
			EdgeGateway: *edgeGateway,
			NodeName: nodeName,
		}
	})
}
