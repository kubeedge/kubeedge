package config

import (
	"sync"

	"github.com/kubeedge/api/apis/componentconfig/edgecore/v1alpha2"
)

var Config Configure
var once sync.Once

type Configure struct {
	v1alpha2.StreamRuleEndpoint
	NodeName string
}

func InitConfigure(s *v1alpha2.StreamRuleEndpoint, nodeName string) {
	once.Do(func() {
		Config = Configure{
			StreamRuleEndpoint: *s,
			NodeName:           nodeName,
		}
	})
}
