package config

import (
	"sync"

	"github.com/kubeedge/kubeedge/pkg/apis/componentconfig/cloudcore/v1alpha1"
)

var Config Configure
var once sync.Once

type Configure struct {
	v1alpha1.StreamController
}

func InitConfigure(stream *v1alpha1.StreamController) {
	once.Do(func() {
		Config = Configure{
			StreamController: *stream,
		}
	})
}
