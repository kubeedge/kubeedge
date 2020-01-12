package config

import (
	"fmt"
	"os"
	"sync"
	"time"

	"k8s.io/klog"

	"github.com/kubeedge/beehive/pkg/common/config"
	"github.com/kubeedge/kubeedge/common/constants"
)

var c Configure
var once sync.Once

type Configure struct {
	// UpdatePodStatusBuffer is the size of channel which save update pod status message from edge
	UpdatePodStatusBuffer int
	// UpdateNodeStatusBuffer is the size of channel which save update node status message from edge
	UpdateNodeStatusBuffer int
	// QueryConfigMapBuffer is the size of channel which save query configmap message from edge
	QueryConfigMapBuffer int
	// QuerySecretBuffer is the size of channel which save query secret message from edge
	QuerySecretBuffer int
	// QueryServiceBuffer is the size of channel which save query service message from edge
	QueryServiceBuffer int
	// QueryEndpointsBuffer is the size of channel which save query endpoints message from edge
	QueryEndpointsBuffer int
	// PodEventBuffer is the size of channel which save pod event from k8s
	PodEventBuffer int
	// ConfigMapEventBuffer is the size of channel which save configmap event from k8s
	ConfigMapEventBuffer int
	// SecretEventBuffer is the size of channel which save secret event from k8s
	SecretEventBuffer int
	// ServiceEventBuffer is the size of channel which save service event from k8s
	ServiceEventBuffer int
	// EndpointsEventBuffer is the size of channel which save endpoints event from k8s
	EndpointsEventBuffer int
	// QueryPersistentVolumeBuffer is the size of channel which save query persistentvolume message from edge
	QueryPersistentVolumeBuffer int
	// QueryPersistentVolumeClaimBuffer is the size of channel which save query persistentvolumeclaim message from edge
	QueryPersistentVolumeClaimBuffer int
	// QueryVolumeAttachmentBuffer is the size of channel which save query volumeattachment message from edge
	QueryVolumeAttachmentBuffer int
	// QueryNodeBuffer is the size of channel which save query node message from edge
	QueryNodeBuffer int
	// UpdateNodeBuffer is the size of channel which save update node message from edge
	UpdateNodeBuffer int

	// ContextSendModule is the name send message to
	ContextSendModule string
	// ContextReceiveModule is the name receive message from
	ContextReceiveModule string
	// ContextResponseModule is the name response message from
	ContextResponseModule string

	// KubeMaster is the url of edge master(kube api server)
	KubeMaster string
	// KubeConfig is the config used connect to edge master
	KubeConfig string
	// KubeContentType is the content type communicate with edge master(default is "application/vnd.kubernetes.protobuf")
	KubeContentType string
	// KubeQPS is the QPS communicate with edge master(default is 100)
	KubeQPS float32
	// KubeBurst default is 200
	KubeBurst int
	// KubeUpdateNodeFrequency is the time duration for update node status(default is 20s)
	KubeUpdateNodeFrequency time.Duration

	// KubeNodeName for the current node
	KubeNodeName string
	//EdgeSiteEnabled is used to enable or disable EdgeSite feature. Default is disabled
	EdgeSiteEnabled bool

	// UpdatePodStatusWorkers is the count of goroutines of update pod status
	UpdatePodStatusWorkers int
	// UpdateNodeStatusWorkers is the count of goroutines of update node status
	UpdateNodeStatusWorkers int
	// QueryConfigMapWorkers is the count of goroutines of query configmap
	QueryConfigMapWorkers int
	// QuerySecretWorkers is the count of goroutines of query secret
	QuerySecretWorkers int
	// QueryServiceWorkers is the count of goroutines of query service
	QueryServiceWorkers int
	// QueryEndpointsWorkers is the count of goroutines of query endpoints
	QueryEndpointsWorkers int
	// QueryPersistentVolumeWorkers is the count of goroutines of query persistentvolume
	QueryPersistentVolumeWorkers int
	// QueryPersistentVolumeClaimWorkers is the count of goroutines of query persistentvolumeclaim
	QueryPersistentVolumeClaimWorkers int
	// QueryVolumeAttachmentWorkers is the count of goroutines of query volumeattachment
	QueryVolumeAttachmentWorkers int
	// QueryNodeWorkers is the count of goroutines of query node
	QueryNodeWorkers int
	// UpdateNodeWorkers is the count of goroutines of update node
	UpdateNodeWorkers int
}

func InitConfigure() {
	once.Do(func() {
		var errs []error

		psb, err := config.CONFIG.GetValue("controller.buffer.update-pod-status").ToInt()
		if err != nil {
			// Guaranteed forward compatibility @kadisi
			psb = constants.DefaultUpdatePodStatusBuffer
			klog.Infof("can not get controller.buffer.update-pod-status key, use default value %v", psb)
		}
		nsb, err := config.CONFIG.GetValue("controller.buffer.update-node-status").ToInt()
		if err != nil {
			// Guaranteed forward compatibility @kadisi
			nsb = constants.DefaultUpdateNodeStatusBuffer
			klog.Infof("can not get controller.buffer.update-node-status key, use default value %v", nsb)
		}
		qcb, err := config.CONFIG.GetValue("controller.buffer.query-configmap").ToInt()
		if err != nil {
			// Guaranteed forward compatibility @kadisi
			qcb = constants.DefaultQueryConfigMapBuffer
			klog.Infof("can not get controller.buffer.query-configmap key, use default value %v", qcb)
		}
		qsb, err := config.CONFIG.GetValue("controller.buffer.query-secret").ToInt()
		if err != nil {
			// Guaranteed forward compatibility @kadisi
			qsb = constants.DefaultQuerySecretBuffer
			klog.Infof("can not get controller.buffer.query-secret key, use default value %v", qsb)
		}
		qservicebuffer, err := config.CONFIG.GetValue("controller.buffer.query-service").ToInt()
		if err != nil {
			// Guaranteed forward compatibility @kadisi
			qservicebuffer = constants.DefaultQueryServiceBuffer
			klog.Infof("can not get controller.buffer.query-service key, use default value %v", qservicebuffer)
		}
		qeb, err := config.CONFIG.GetValue("controller.buffer.query-endpoints").ToInt()
		if err != nil {
			// Guaranteed forward compatibility @kadisi
			qeb = constants.DefaultQueryEndpointsBuffer
			klog.Infof("can not get controller.buffer.query-endpoint key, use default value %v", qeb)
		}
		peb, err := config.CONFIG.GetValue("controller.buffer.pod-event").ToInt()
		if err != nil {
			// Guaranteed forward compatibility @kadisi
			peb = constants.DefaultPodEventBuffer
			klog.Infof("can not get controller.buffer.pod-event key, use default value %v", peb)
		}
		cmeb, err := config.CONFIG.GetValue("controller.buffer.configmap-event").ToInt()
		if err != nil {
			// Guaranteed forward compatibility @kadisi
			cmeb = constants.DefaultConfigMapEventBuffer
			klog.Infof("can not get controller.buffer.configmap-event key, use default value %v", cmeb)
		}
		seb, err := config.CONFIG.GetValue("controller.buffer.secret-event").ToInt()
		if err != nil {
			// Guaranteed forward compatibility @kadisi
			seb = constants.DefaultSecretEventBuffer
			klog.Infof("can not get controller.buffer.secret-event key, use default value %v", seb)
		}
		sebuffer, err := config.CONFIG.GetValue("controller.buffer.service-event").ToInt()
		if err != nil {
			// Guaranteed forward compatibility @kadisi
			sebuffer = constants.DefaultServiceEventBuffer
			klog.Infof("can not get controller.buffer.service-event key, use default value %v", sebuffer)
		}
		epb, err := config.CONFIG.GetValue("controller.buffer.endpoints-event").ToInt()
		if err != nil {
			// Guaranteed forward compatibility @kadisi
			epb = constants.DefaultEndpointsEventBuffer
			klog.Infof("can not get controller.buffer.endpoints-event key, use default value %v", epb)
		}
		qpvb, err := config.CONFIG.GetValue("controller.buffer.query-persistentvolume").ToInt()
		if err != nil {
			// Guaranteed forward compatibility @kadisi
			qpvb = constants.DefaultQueryPersistentVolumeBuffer
			klog.Infof("can not get controller.buffer.query-persistentvolume key, use default value %v", qpvb)
		}
		qpvcb, err := config.CONFIG.GetValue("controller.buffer.query-persistentvolumeclaim").ToInt()
		if err != nil {
			// Guaranteed forward compatibility @kadisi
			qpvcb = constants.DefaultQueryPersistentVolumeClaimBuffer
			klog.Infof("can not get controller.buffer.query-persistentvolumeclaim key, use default value %v", qpvcb)
		}
		qvab, err := config.CONFIG.GetValue("controller.buffer.query-volumeattachment").ToInt()
		if err != nil {
			// Guaranteed forward compatibility @kadisi
			qvab = constants.DefaultQueryVolumeAttachmentBuffer
			klog.Infof("can not get controller.buffer.query-volumeattachment key, use default value %v", qvab)
		}
		qnb, err := config.CONFIG.GetValue("controller.buffer.query-node").ToInt()
		if err != nil {
			// Guaranteed forward compatibility @kadisi
			qnb = constants.DefaultQueryNodeBuffer
			klog.Infof("can not get controller.buffer.query-node key, use default value %v", qnb)
		}
		unb, err := config.CONFIG.GetValue("controller.buffer.update-node").ToInt()
		if err != nil {
			// Guaranteed forward compatibility @kadisi
			unb = constants.DefaultUpdateNodeBuffer
			klog.Infof("can not get controller.buffer.update-node key, use default value %v", unb)
		}
		smn, err := config.CONFIG.GetValue("controller.context.send-module").ToString()
		if err != nil {
			// Guaranteed forward compatibility @kadisi
			smn = constants.DefaultContextSendModuleName
			klog.Infof("can not get controller.context.send-module key, use default value %v", smn)
		}
		rmn, err := config.CONFIG.GetValue("controller.context.receive-module").ToString()
		if err != nil {
			// Guaranteed forward compatibility @kadisi
			rmn = constants.DefaultContextReceiveModuleName
			klog.Infof("can not get controller.context.receive-module key, use default value %v", rmn)
		}
		resn, err := config.CONFIG.GetValue("controller.context.response-module").ToString()
		if err != nil {
			// Guaranteed forward compatibility @kadisi
			resn = constants.DefaultContextResponseModuleName
			klog.Infof("can not get controller.context.response-module key, use default value %v", resn)
		}
		nodeName, err := config.CONFIG.GetValue("controller.kube.node-name").ToString()
		if err != nil {
			// Guaranteed forward compatibility @kadisi
			nodeName = ""
			klog.Infof("can not get controller.kube.node-name key, use default value %v", nodeName)
		}
		es, err := config.CONFIG.GetValue("metamanager.edgesite").ToBool()
		if err != nil {
			// Guaranteed forward compatibility @kadisi
			es = false
			klog.Infof("can not get metamanager.edgesite key, use default value %v", es)
		}
		km, err := config.CONFIG.GetValue("controller.kube.master").ToString()
		if err != nil {
			// Guaranteed forward compatibility @kadisi
			km = ""
			klog.Infof("can not get key controller.kube.master, use default value %v", km)
		}
		kc, err := config.CONFIG.GetValue("controller.kube.kubeconfig").ToString()
		if err != nil {
			kc = ""
			klog.Infof("can not get key controller.kube.kubeconfig, use default value %v", kc)
		}
		if km == "" && kc == "" {
			errs = append(errs, fmt.Errorf("controller.kube.kubeconfig and controller.kube.master are not both setd"))
		}

		kct, err := config.CONFIG.GetValue("controller.kube.content_type").ToString()
		if err != nil {
			// Guaranteed forward compatibility @kadisi
			kct = constants.DefaultKubeContentType
			klog.Infof("can not get key controller.kube.content_type, use default value %v", kct)
		}
		kqps, err := config.CONFIG.GetValue("controller.kube.qps").ToFloat64()
		if err != nil {
			// Guaranteed forward compatibility @kadisi
			kqps = constants.DefaultKubeQPS
			klog.Infof("can not get key controller.kube.qps, use default value %v", kqps)
		}
		kb, err := config.CONFIG.GetValue("controller.kube.burst").ToInt()
		if err != nil {
			// Guaranteed forward compatibility @kadisi
			kb = constants.DefaultKubeBurst
			klog.Infof("can not get key controller.kube.burst, use default value %v", kb)
		}
		kuf, err := config.CONFIG.GetValue("controller.kube.node_update_frequency").ToInt64()
		if err != nil {
			// Guaranteed forward compatibility @kadisi
			kuf = int64(constants.DefaultKubeUpdateNodeFrequency)
			klog.Infof("can not get key controller.kube.node_update_frequency, use default value %v", kuf)
		}
		psw, err := config.CONFIG.GetValue("controller.load.update-pod-status-workers").ToInt()
		if err != nil {
			psw = constants.DefaultUpdatePodStatusWorkers
			klog.Infof("can not get key controller.load.update-pod-status-workers, use default value %v", psw)
		}
		nsw, err := config.CONFIG.GetValue("controller.load.update-node-status-workers").ToInt()
		if err != nil {
			// Guaranteed forward compatibility @kadisi
			nsw = constants.DefaultUpdateNodeStatusWorkers
			klog.Infof("can not get key controller.load.update-node-status-workers, use default value %v", nsw)
		}
		qcw, err := config.CONFIG.GetValue("controller.load.query-configmap-workers").ToInt()
		if err != nil {
			// Guaranteed forward compatibility @kadisi
			qcw = constants.DefaultQueryConfigMapWorkers
			klog.Infof("can not get key controller.load.query-configmap-workers, use default value %v", qcw)
		}
		qsecretw, err := config.CONFIG.GetValue("controller.load.query-secret-workers").ToInt()
		if err != nil {
			// Guaranteed forward compatibility @kadisi
			qsecretw = constants.DefaultQuerySecretWorkers
			klog.Infof("can not get key controller.load.query-secret-workers, use default value %v", qsecretw)
		}
		qsw, err := config.CONFIG.GetValue("controller.load.query-service-workers").ToInt()
		if err != nil {
			// Guaranteed forward compatibility @kadisi
			qsw = constants.DefaultQueryServiceWorkers
			klog.Infof("can not get key controller.load.query-service-workers, use default value %v", qsw)
		}
		qew, err := config.CONFIG.GetValue("controller.load.query-endpoints-workers").ToInt()
		if err != nil {
			// Guaranteed forward compatibility @kadisi
			qew = constants.DefaultQueryEndpointsWorkers
			klog.Infof("can not get key controller.load.query-endpoint-workers, use default value %v", qew)
		}
		qpvw, err := config.CONFIG.GetValue("controller.load.query-persistentvolume-workers").ToInt()
		if err != nil {
			// Guaranteed forward compatibility @kadisi
			qpvw = constants.DefaultQueryPersistentVolumeWorkers
			klog.Infof("can not get key controller.load.query-persistentvolume-workers, use default value %v", qpvw)
		}
		qpvcw, err := config.CONFIG.GetValue("controller.load.query-persistentvolumeclaim-workers").ToInt()
		if err != nil {
			// Guaranteed forward compatibility @kadisi
			qpvcw = constants.DefaultQueryPersistentVolumeClaimWorkers
			klog.Infof("can not get key controller.load.query-persistentvolumeclaim-workers, use default value %v", qpvcw)
		}
		qvaw, err := config.CONFIG.GetValue("controller.load.query-volumeattachment-workers").ToInt()
		if err != nil {
			// Guaranteed forward compatibility @kadisi
			qvaw = constants.DefaultQueryVolumeAttachmentWorkers
			klog.Infof("can not get key controller.load.query-volumeattachment-workers, use default value %v", qvaw)
		}
		qnw, err := config.CONFIG.GetValue("controller.load.query-node-workers").ToInt()
		if err != nil {
			// Guaranteed forward compatibility @kadisi
			qnw = constants.DefaultQueryNodeWorkers
			klog.Infof("can not get key controller.load.query-node-workers, use default value %v", qnw)
		}
		unw, err := config.CONFIG.GetValue("controller.load.update-node-workers").ToInt()
		if err != nil {
			// Guaranteed forward compatibility @kadisi
			unw = constants.DefaultUpdateNodeWorkers
			klog.Infof("can not get key controller.load.update-node-workers, use default value %v", unw)
		}
		if len(errs) != 0 {
			for _, e := range errs {
				klog.Errorf("%v", e)
			}
			klog.Error("init edgecontroller config error")
			os.Exit(1)
		}
		c = Configure{
			UpdatePodStatusBuffer:             psb,
			UpdateNodeStatusBuffer:            nsb,
			QueryConfigMapBuffer:              qcb,
			QuerySecretBuffer:                 qsb,
			QueryServiceBuffer:                qservicebuffer,
			QueryEndpointsBuffer:              qeb,
			PodEventBuffer:                    peb,
			ConfigMapEventBuffer:              cmeb,
			SecretEventBuffer:                 seb,
			ServiceEventBuffer:                sebuffer,
			EndpointsEventBuffer:              epb,
			QueryPersistentVolumeBuffer:       qpvb,
			QueryPersistentVolumeClaimBuffer:  qpvcb,
			QueryVolumeAttachmentBuffer:       qvab,
			QueryNodeBuffer:                   qnb,
			UpdateNodeBuffer:                  unb,
			ContextSendModule:                 smn,
			ContextReceiveModule:              rmn,
			ContextResponseModule:             resn,
			KubeNodeName:                      nodeName,
			EdgeSiteEnabled:                   es,
			KubeMaster:                        km,
			KubeConfig:                        kc,
			KubeContentType:                   kct,
			KubeQPS:                           float32(kqps),
			KubeBurst:                         kb,
			KubeUpdateNodeFrequency:           time.Duration(kuf) * time.Second,
			UpdatePodStatusWorkers:            psw,
			UpdateNodeStatusWorkers:           nsw,
			QueryConfigMapWorkers:             qcw,
			QuerySecretWorkers:                qsecretw,
			QueryServiceWorkers:               qsw,
			QueryEndpointsWorkers:             qew,
			QueryPersistentVolumeWorkers:      qpvw,
			QueryPersistentVolumeClaimWorkers: qpvcw,
			QueryVolumeAttachmentWorkers:      qvaw,
			QueryNodeWorkers:                  qnw,
			UpdateNodeWorkers:                 unw,
		}
		klog.Infof("init edgecontroller config successfully, config info %++v", c)
	})
}

func Get() *Configure {
	return &c
}
