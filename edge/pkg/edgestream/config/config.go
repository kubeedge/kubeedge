package config

import (
	"sync"

	"github.com/kubeedge/kubeedge/pkg/apis/componentconfig/edgecore/v1alpha1"
)

var Config Configure
var once sync.Once

type Configure struct {
	v1alpha1.EdgeStream
}

func InitConfigure(stream *v1alpha1.EdgeStream) {
	once.Do(func() {
		Config = Configure{
			EdgeStream: *stream,
		}
	})
}
