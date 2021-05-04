package config

import (
	"sync"

	"github.com/kubeedge/kubeedge/pkg/apis/componentconfig/edgecore/v1alpha1"
)

var config Configure
var once sync.Once

type Configure struct {
	v1alpha1.DeviceTwin
	NodeName string
}

func InitConfigure(deviceTwin *v1alpha1.DeviceTwin, nodeName string) {
	once.Do(func() {
		config = Configure{
			DeviceTwin: *deviceTwin,
			NodeName:   nodeName,
		}
	})
}

func Get() *Configure {
	return &config
}
