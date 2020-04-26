package httpserver

import (
	"bytes"
	hubconfig "github.com/kubeedge/kubeedge/cloud/pkg/cloudhub/config"
)

func UpdateConfig(ca,caKey,cert,key []byte) {
	blank:=[]byte("")
	if !bytes.Equal(blank,ca){
		hubconfig.Config.Ca=ca
	}
	if !bytes.Equal(blank,caKey){
		hubconfig.Config.CaKey=caKey
	}
	if !bytes.Equal(blank,cert){
		hubconfig.Config.Cert=cert
	}
	if !bytes.Equal(blank,key){
		hubconfig.Config.Key=key
	}
}