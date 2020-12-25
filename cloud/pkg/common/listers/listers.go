package listers

import (
	"sync"

	v1 "k8s.io/client-go/listers/core/v1"

	"github.com/kubeedge/kubeedge/cloud/pkg/client/listers/devices/v1alpha2"
	"github.com/kubeedge/kubeedge/cloud/pkg/client/listers/reliablesyncs/v1alpha1"
	"github.com/kubeedge/kubeedge/cloud/pkg/common/informers"
)

var globalListers Lister
var once sync.Once

func GetListers() Lister {
	once.Do(func() {
		globalListers = &listers{}
	})
	return globalListers
}

type Lister interface {
	PodLister() v1.PodLister
	ConfigMapLister() v1.ConfigMapLister
	SecretLister() v1.SecretLister
	ServiceLister() v1.ServiceLister
	EndpointsLister() v1.EndpointsLister
	NodeLister() v1.NodeLister
	ClusterObjectSyncLister() v1alpha1.ClusterObjectSyncLister
	ObjectSyncLister() v1alpha1.ObjectSyncLister
	DeviceLister() v1alpha2.DeviceLister
}

type listers struct {
	podinit   sync.Once
	podlister v1.PodLister
	cminit    sync.Once
	cmlister  v1.ConfigMapLister
	secinit   sync.Once
	seclister v1.SecretLister
	svcinit   sync.Once
	svclister v1.ServiceLister
	epinit    sync.Once
	eplister  v1.EndpointsLister
	noinit    sync.Once
	nolister  v1.NodeLister
	cosinit   sync.Once
	coslister v1alpha1.ClusterObjectSyncLister
	osinit    sync.Once
	oslister  v1alpha1.ObjectSyncLister
	devinit   sync.Once
	devlister v1alpha2.DeviceLister
}

func (l *listers) DeviceLister() v1alpha2.DeviceLister {
	l.devinit.Do(func() {
		l.devlister = v1alpha2.NewDeviceLister(informers.GetGlobalInformers().Device().GetIndexer())
	})
	return l.devlister
}

func (l *listers) PodLister() v1.PodLister {
	l.podinit.Do(func() {
		l.podlister = v1.NewPodLister(informers.GetGlobalInformers().Pod().GetIndexer())
	})
	return l.podlister
}
func (l *listers) ConfigMapLister() v1.ConfigMapLister {
	l.cminit.Do(func() {
		l.cmlister = v1.NewConfigMapLister(informers.GetGlobalInformers().ConfigMap().GetIndexer())
	})
	return l.cmlister
}

func (l *listers) SecretLister() v1.SecretLister {
	l.secinit.Do(func() {
		l.seclister = v1.NewSecretLister(informers.GetGlobalInformers().Secrets().GetIndexer())
	})
	return l.seclister
}

func (l *listers) ServiceLister() v1.ServiceLister {
	l.svcinit.Do(func() {
		l.svclister = v1.NewServiceLister(informers.GetGlobalInformers().Service().GetIndexer())
	})
	return l.svclister
}

func (l *listers) EndpointsLister() v1.EndpointsLister {
	l.epinit.Do(func() {
		l.eplister = v1.NewEndpointsLister(informers.GetGlobalInformers().Endpoints().GetIndexer())
	})
	return l.eplister
}

func (l *listers) NodeLister() v1.NodeLister {
	l.noinit.Do(func() {
		l.nolister = v1.NewNodeLister(informers.GetGlobalInformers().Node().GetIndexer())
	})
	return l.nolister
}

func (l *listers) ClusterObjectSyncLister() v1alpha1.ClusterObjectSyncLister {
	l.cosinit.Do(func() {
		l.coslister = v1alpha1.NewClusterObjectSyncLister(informers.GetGlobalInformers().ClusterObjectSync().GetIndexer())
	})
	return l.coslister
}

func (l *listers) ObjectSyncLister() v1alpha1.ObjectSyncLister {
	l.osinit.Do(func() {
		l.oslister = v1alpha1.NewObjectSyncLister(informers.GetGlobalInformers().ObjectSync().GetIndexer())
	})
	return l.oslister
}
