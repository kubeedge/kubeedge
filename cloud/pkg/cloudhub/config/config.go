package config

import (
	"io/ioutil"
	"os"
	"sync"

	"k8s.io/klog"

	cconfig "github.com/kubeedge/kubeedge/cloud/pkg/apis/cloudcore/config"
)

var hubConfig Config
var once sync.Once

func InitHubConfig(h cconfig.CloudHubConfig) {
	once.Do(func() {
		var err error
		hubConfig.CloudHubConfig = h

		hubConfig.Ca, err = ioutil.ReadFile(h.TLSCaFile)
		if err != nil {
			klog.Errorf("Read Ca file %s error", h.TLSCaFile)
			os.Exit(1)
		}

		hubConfig.Cert, err = ioutil.ReadFile(h.TLSCertFile)
		if err != nil {
			klog.Errorf("Read cert file %s error", h.TLSCaFile)
			os.Exit(1)
		}
		hubConfig.Key, err = ioutil.ReadFile(h.TLSPrivateKeyFile)
		if err != nil {
			klog.Errorf("Read key file %s error", h.TLSCaFile)
			os.Exit(1)
		}
	})
}

// HubConfig represents configuration options for http access
type Config struct {
	cconfig.CloudHubConfig
	Ca   []byte
	Cert []byte
	Key  []byte
}

func Conf() *Config {
	return &hubConfig
}
