package config

import (
	"io/ioutil"
	"k8s.io/klog"
	"sync"

	"github.com/kubeedge/kubeedge/cloud/pkg/client/clientset/versioned"
	syncinformer "github.com/kubeedge/kubeedge/cloud/pkg/client/informers/externalversions/reliablesyncs/v1alpha1"
	synclister "github.com/kubeedge/kubeedge/cloud/pkg/client/listers/reliablesyncs/v1alpha1"
	"github.com/kubeedge/kubeedge/pkg/apis/componentconfig/cloudcore/v1alpha1"
	"k8s.io/client-go/tools/cache"
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

func InitConfigure(hub *v1alpha1.CloudHub, kubeAPIConfig *v1alpha1.KubeAPIConfig) {
	once.Do(func() {
		Config = Configure{
			CloudHub:      *hub,
			KubeAPIConfig: kubeAPIConfig,
		}

		ca, _ := ioutil.ReadFile(hub.TLSCAFile)
		caKey, _ := ioutil.ReadFile(hub.TLSCAKeyFile)

		if ca != nil && caKey != nil {
			Config.Ca = ca
			Config.CaKey = caKey
		} else if !(ca == nil && caKey == nil) {
			klog.Fatal("Both of ca and caKey should be specified!")
		}

		cert, _ := ioutil.ReadFile(hub.TLSCertFile)
		key, _ := ioutil.ReadFile(hub.TLSPrivateKeyFile)

		if cert != nil && key != nil {
			Config.Cert = cert
			Config.Key = key
		} else if !(cert == nil && key == nil) {
			klog.Fatal("Both of cert and key should be specified!")
		}
	})
}

// ObjectSyncController use beehive context message layer
type ObjectSyncController struct {
	CrdClient versioned.Interface

	// informer
	ClusterObjectSyncInformer syncinformer.ClusterObjectSyncInformer
	ObjectSyncInformer        syncinformer.ObjectSyncInformer

	// synced
	ClusterObjectSyncSynced cache.InformerSynced
	ObjectSyncSynced        cache.InformerSynced

	// lister
	ClusterObjectSyncLister synclister.ClusterObjectSyncLister
	ObjectSyncLister        synclister.ObjectSyncLister
}
