package config

import (
	"sync"

	"github.com/google/cadvisor/utils"

	"io/ioutil"

	"k8s.io/klog"

	"github.com/kubeedge/kubeedge/pkg/apis/componentconfig/edgecore/v1alpha1"
)

var Config Configure
var once sync.Once

type Configure struct {
	v1alpha1.EdgePorxy
	CaData   []byte
	CertData []byte
	KeyData  []byte
}

func InitConfigure(ep *v1alpha1.EdgePorxy) {
	once.Do(func() {
		Config = Configure{
			EdgePorxy: *ep,
		}
		caData, err := loadFromFile(ep.CAFile)
		if err != nil {
			klog.Errorf("load edgeproxy ca file failed! err: %v", err)
			panic(err)
		}
		Config.CaData = caData
		if utils.FileExists(ep.CertFile) {
			certData, err := loadFromFile(ep.CertFile)
			if err != nil {
				klog.Errorf("load edgeproxy cert file failed! err: %v", err)
				panic(err)
			}
			Config.CertData = certData
		}
		if utils.FileExists(ep.PrivateKeyFile) {
			keyData, err := loadFromFile(ep.PrivateKeyFile)
			if err != nil {
				klog.Errorf("load edgeproxy key file failed! err: %v", err)
				panic(err)
			}
			Config.KeyData = keyData
		}
	})
}

func loadFromFile(file string) ([]byte, error) {
	data, err := ioutil.ReadFile(file)
	if err != nil {
		return []byte{}, err
	}
	return data, nil
}
