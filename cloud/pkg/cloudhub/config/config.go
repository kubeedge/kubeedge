package config

import (
	"sync"

	"k8s.io/klog/v2"

	"github.com/kubeedge/api/apis/componentconfig/cloudcore/v1alpha1"
	"github.com/kubeedge/kubeedge/pkg/security/certs"
)

var Config Configure
var once sync.Once

type Configure struct {
	v1alpha1.CloudHub
	KubeAPIConfig *v1alpha1.KubeAPIConfig
	Ca            []byte
	CaKey         []byte
	Cert          []byte
	Key           []byte
}

func InitConfigure(hub *v1alpha1.CloudHub) {
	once.Do(func() {
		if len(hub.AdvertiseAddress) == 0 {
			klog.Exit("AdvertiseAddress must be specified!")
		}

		Config = Configure{CloudHub: *hub}

		var ca, caKey, cert, key []byte

		if hub.TLSCAFile != "" {
			if block, err := certs.ReadPEMFile(hub.TLSCAFile); err == nil {
				ca = block.Bytes
				klog.Info("succeed in loading CA certificate from local directory")
			} else {
				klog.Warningf("failed to load the CA certificate file %s, err: %v", hub.TLSCAFile, err)
			}
		}
		if hub.TLSCAKeyFile != "" {
			if block, err := certs.ReadPEMFile(hub.TLSCAKeyFile); err == nil {
				caKey = block.Bytes
				klog.Info("succeed in loading CA key from local directory")
			} else {
				klog.Warningf("failed to load the CA key file %s, err: %v", hub.TLSCAKeyFile, err)
			}
		}
		if ca != nil && caKey != nil {
			Config.Ca = ca
			Config.CaKey = caKey
		} else if !(ca == nil && caKey == nil) {
			klog.Exit("Both of ca and caKey should be specified!")
		}

		if hub.TLSCertFile != "" {
			if block, err := certs.ReadPEMFile(hub.TLSCertFile); err == nil {
				cert = block.Bytes
				klog.Info("succeed in loading certificate from local directory")
			} else {
				klog.Warningf("failed to load the certificate file %s, err: %v", hub.TLSCertFile, err)
			}
		}
		if hub.TLSPrivateKeyFile != "" {
			if block, err := certs.ReadPEMFile(hub.TLSPrivateKeyFile); err == nil {
				key = block.Bytes
				klog.Info("succeed in loading private key from local directory")
			} else {
				klog.Warningf("failed to load the private key file %s, err: %v", hub.TLSPrivateKeyFile, err)
			}
		}
		if cert != nil && key != nil {
			Config.Cert = cert
			Config.Key = key
		} else if !(cert == nil && key == nil) {
			klog.Exit("Both of cert and key should be specified!")
		}
	})
}

func (c *Configure) UpdateCA(ca, caKey []byte) {
	if ca != nil {
		c.Ca = ca
	}
	if caKey != nil {
		c.CaKey = caKey
	}
}

func (c *Configure) UpdateCerts(cert, key []byte) {
	if cert != nil {
		c.Cert = cert
	}
	if key != nil {
		c.Key = key
	}
}
