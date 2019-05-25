package config

import (
	"time"

	"github.com/kubeedge/beehive/pkg/common/config"
	"github.com/kubeedge/beehive/pkg/common/log"
	"github.com/kubeedge/kubeedge/common/types"
)

// Kube contains Kubernetes related configuration
var Kube *types.KubeInfo

func init() {
	Kube = types.NewKubeInfo()

	if km, err := config.CONFIG.GetValue("controller.kube.master").ToString(); err != nil {
		log.LOGGER.Errorf("kube master not set")
	} else {
		Kube.KubeMaster = km
	}
	log.LOGGER.Infof("kube master: %s", Kube.KubeMaster)

	if kc, err := config.CONFIG.GetValue("controller.kube.kubeconfig").ToString(); err != nil {
		log.LOGGER.Errorf("kube config not set")
	} else {
		Kube.KubeConfig = kc
	}
	log.LOGGER.Infof("kube config: %s", Kube.KubeConfig)

	if kn, err := config.CONFIG.GetValue("controller.kube.namespace").ToString(); err == nil {
		Kube.KubeNamespace = kn
	}
	log.LOGGER.Infof("kube namespace: %s", Kube.KubeNamespace)

	if kct, err := config.CONFIG.GetValue("controller.kube.content_type").ToString(); err == nil {
		Kube.KubeContentType = kct
	}
	log.LOGGER.Infof("kube content type: %s", Kube.KubeContentType)

	if kqps, err := config.CONFIG.GetValue("controller.kube.qps").ToFloat64(); err == nil {
		Kube.KubeQPS = float32(kqps)
	}
	log.LOGGER.Infof("kube QPS: %f", Kube.KubeQPS)

	if kb, err := config.CONFIG.GetValue("controller.kube.burst").ToInt(); err == nil {
		Kube.KubeBurst = kb
	}
	log.LOGGER.Infof("kube burst: %d", Kube.KubeBurst)

	if kuf, err := config.CONFIG.GetValue("controller.kube.node_update_frequency").ToInt64(); err == nil {
		Kube.KubeUpdateNodeFrequency = time.Duration(kuf) * time.Second
	}
	log.LOGGER.Infof("kube update frequency: %v", Kube.KubeUpdateNodeFrequency)

	if id, err := config.CONFIG.GetValue("controller.kube.node-id").ToString(); err == nil {
		Kube.KubeNodeID = id
	}
	log.LOGGER.Infof("kube Node ID: %s", Kube.KubeNodeID)

	if name, err := config.CONFIG.GetValue("controller.kube.node-name").ToString(); err == nil {
		Kube.KubeNodeName = name
	}
	log.LOGGER.Infof("kube Node Name: %s", Kube.KubeNodeName)
}
