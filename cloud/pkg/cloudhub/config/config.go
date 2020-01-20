package config

import (
	"io/ioutil"
	"sync"

	"k8s.io/klog"

	"github.com/kubeedge/kubeedge/pkg/apis/cloudcore/v1alpha1"
)

var c Configure
var once sync.Once

type Configure struct {
	v1alpha1.CloudHub
	Ca   []byte
	Cert []byte
	Key  []byte
}

func InitConfigure(hub *v1alpha1.CloudHub) {
	once.Do(func() {
		ca, err := ioutil.ReadFile(hub.TLSCAFile)
		if err != nil {
			klog.Fatalf("read ca file %v error %v", hub.TLSCAFile, err)
		}
		cert, err := ioutil.ReadFile(hub.TLSCertFile)
		if err != nil {
			klog.Fatalf("read cert file %v error %v", hub.TLSCertFile, err)
		}
		key, err := ioutil.ReadFile(hub.TLSPrivateKeyFile)
		if err != nil {
			klog.Fatalf("read key file %v error %v", hub.TLSPrivateKeyFile, err)
		}
		c = Configure{
			CloudHub: *hub,
			Ca:       ca,
			Cert:     cert,
			Key:      key,
		}
	})
}

func Get() *Configure {
	return &c
}
