package config

import (
	"time"

	"github.com/kubeedge/beehive/pkg/common/config"
	"github.com/kubeedge/beehive/pkg/common/log"
	"github.com/kubeedge/kubeedge/common/types"
)

// Kube container Kubernetes related configuration
var Kube *types.KubeInfo

func init() {
	Kube = types.NewKubeInfo()

	if km, err := config.CONFIG.GetValue("controller.kube.master").ToString(); err != nil {
		log.LOGGER.Errorf("Controller kube master not set")
	} else {
		Kube.KubeMaster = km
	}
	log.LOGGER.Infof("Controller kube master: %s", Kube.KubeMaster)

	if kc, err := config.CONFIG.GetValue("controller.kube.kubeconfig").ToString(); err != nil {
		log.LOGGER.Errorf("Controller kube config not set")
	} else {
		Kube.KubeConfig = kc
	}
	log.LOGGER.Infof("Controller kube config: %s", Kube.KubeConfig)

	if kn, err := config.CONFIG.GetValue("controller.kube.namespace").ToString(); err == nil {
		Kube.KubeNamespace = kn
	}
	log.LOGGER.Infof("Controller kube namespace: %s", Kube.KubeNamespace)

	if kct, err := config.CONFIG.GetValue("controller.kube.content_type").ToString(); err == nil {
		Kube.KubeContentType = kct
	}
	log.LOGGER.Infof("Controller kube content type: %s", Kube.KubeContentType)

	if kqps, err := config.CONFIG.GetValue("controller.kube.qps").ToFloat64(); err == nil {
		Kube.KubeQPS = float32(kqps)
	}
	log.LOGGER.Infof("Controller kube QPS: %f", Kube.KubeQPS)

	if kb, err := config.CONFIG.GetValue("controller.kube.burst").ToInt(); err == nil {
		Kube.KubeBurst = kb
	}
	log.LOGGER.Infof("Controller kube burst: %d", Kube.KubeBurst)

	if kuf, err := config.CONFIG.GetValue("controller.kube.node_update_frequency").ToInt64(); err == nil {
		Kube.KubeUpdateNodeFrequency = time.Duration(kuf) * time.Second
	}
	log.LOGGER.Infof("Controller kube update frequency: %v", Kube.KubeUpdateNodeFrequency)
}
