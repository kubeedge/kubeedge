package config

import (
	"sync"

	"github.com/kubeedge/kubeedge/pkg/apis/edgecore/v1alpha1"
)

var c Configure
var once sync.Once

type Configure struct {
	v1alpha1.DeviceTwin
	NodeName string
}

func InitConfigure(d *v1alpha1.DeviceTwin, nodeName string) {
	once.Do(func() {
		c = Configure{
			DeviceTwin: *d,
			NodeName:   nodeName,
		}
	})
}

func Get() *Configure {
	return &c
}
