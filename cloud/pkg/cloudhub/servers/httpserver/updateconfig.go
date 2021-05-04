package httpserver

import (
	hubconfig "github.com/kubeedge/kubeedge/cloud/pkg/cloudhub/config"
)

func UpdateConfig(ca, caKey, cert, key []byte) {
	if ca != nil {
		hubconfig.Config.Ca = ca
	}
	if caKey != nil {
		hubconfig.Config.CaKey = caKey
	}
	if cert != nil {
		hubconfig.Config.Cert = cert
	}
	if key != nil {
		hubconfig.Config.Key = key
	}
}
