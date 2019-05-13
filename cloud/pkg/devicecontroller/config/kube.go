package config

import (
	"github.com/kubeedge/beehive/pkg/common/config"
	"github.com/kubeedge/beehive/pkg/common/log"
	"github.com/kubeedge/kubeedge/cloud/pkg/devicecontroller/constants"
)

// KubeMaster is the url of edge master(kube api server)
var KubeMaster string

// KubeConfig is the config used connect to edge master
var KubeConfig string

// KubeNamespace is the namespace to watch(default is NamespaceAll)
var KubeNamespace string

// KubeContentType is the content type communicate with edge master(default is "application/vnd.kubernetes.protobuf")
var KubeContentType string

// KubeQPS is the QPS communicate with edge master(default is 1024)
var KubeQPS float32

// KubeBurst default is 10
var KubeBurst int

func init() {
	if km, err := config.CONFIG.GetValue("devicecontroller.kube.master").ToString(); err != nil {
		log.LOGGER.Errorf("Devicecontroller kube master not set")
	} else {
		KubeMaster = km
	}
	log.LOGGER.Infof("Devicecontroller kube master: %s", KubeMaster)

	if kc, err := config.CONFIG.GetValue("devicecontroller.kube.kubeconfig").ToString(); err != nil {
		log.LOGGER.Errorf("Devicecontroller kube config not set")
	} else {
		KubeConfig = kc
	}
	log.LOGGER.Infof("Devicecontroller kube config: %s", KubeConfig)

	if kn, err := config.CONFIG.GetValue("devicecontroller.kube.namespace").ToString(); err != nil {
		KubeNamespace = constants.DefaultKubeNamespace
	} else {
		KubeNamespace = kn
	}
	log.LOGGER.Infof("Devicecontroller kube namespace: %s", KubeNamespace)

	if kct, err := config.CONFIG.GetValue("devicecontroller.kube.content_type").ToString(); err != nil {
		KubeContentType = constants.DefaultKubeContentType
	} else {
		KubeContentType = kct
	}
	log.LOGGER.Infof("Devicecontroller kube content type: %s", KubeContentType)

	if kqps, err := config.CONFIG.GetValue("devicecontroller.kube.qps").ToFloat64(); err != nil {
		KubeQPS = constants.DefaultKubeQPS
	} else {
		KubeQPS = float32(kqps)
	}
	log.LOGGER.Infof("Devicecontroller kube QPS: %f", KubeQPS)

	if kb, err := config.CONFIG.GetValue("controller.kube.burst").ToInt(); err != nil {
		KubeBurst = constants.DefaultKubeBurst
	} else {
		KubeBurst = kb
	}
	log.LOGGER.Infof("Devicecontroller kube burst: %d", KubeBurst)
}
