package config

import (
	"io/ioutil"
	"sync"

	"k8s.io/client-go/tools/cache"
	"k8s.io/klog"

	"github.com/kubeedge/kubeedge/cloud/pkg/client/clientset/versioned"
	syncinformer "github.com/kubeedge/kubeedge/cloud/pkg/client/informers/externalversions/reliablesyncs/v1alpha1"
	synclister "github.com/kubeedge/kubeedge/cloud/pkg/client/listers/reliablesyncs/v1alpha1"
	"github.com/kubeedge/kubeedge/pkg/apis/cloudcore/v1alpha1"
)

var c Configure
var once sync.Once

type Configure struct {
	v1alpha1.CloudHub
	KubeAPIConfig *v1alpha1.KubeAPIConfig
	Ca            []byte
	Cert          []byte
	Key           []byte
}

func InitConfigure(hub *v1alpha1.CloudHub, kubeAPIConfig *v1alpha1.KubeAPIConfig) {
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
			CloudHub:      *hub,
			KubeAPIConfig: kubeAPIConfig,
			Ca:            ca,
			Cert:          cert,
			Key:           key,
		}
	})
}

func Get() *Configure {
	return &c
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
