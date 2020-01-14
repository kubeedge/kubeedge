package config

import (
	"sync"

	"github.com/kubeedge/kubeedge/pkg/apis/edgecore/v1alpha1"
)

var c Configure
var once sync.Once

type Configure struct {
	v1alpha1.EdgeMesh
}

func InitConfigure(e *v1alpha1.EdgeMesh) {
	once.Do(func() {
		c = Configure{
			EdgeMesh: *e,
		}
	})
}

func Get() *Configure {
	return &c
}
