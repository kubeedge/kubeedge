package config

import (
	"github.com/kubeedge/beehive/pkg/common/config"
	"github.com/kubeedge/beehive/pkg/common/log"
	"github.com/kubeedge/kubeedge/cloud/pkg/controller/constants"
)

// UpdatePodStatusBuffer is the size of channel which save update pod status message from edge
var UpdatePodStatusBuffer int

// UpdateNodeStatusBuffer is the size of channel which save update node status message from edge
var UpdateNodeStatusBuffer int

// QueryConfigMapBuffer is the size of channel which save query configmap message from edge
var QueryConfigMapBuffer int

// QuerySecretBuffer is the size of channel which save query secret message from edge
var QuerySecretBuffer int

func init() {
	if psb, err := config.CONFIG.GetValue("controller.buffer.update-pod-status").ToInt(); err != nil {
		UpdatePodStatusBuffer = constants.DefaultUpdatePodStatusBuffer
	} else {
		UpdatePodStatusBuffer = psb
	}
	log.LOGGER.Infof("update pod status buffer: %d", UpdatePodStatusBuffer)

	if nsb, err := config.CONFIG.GetValue("controller.buffer.update-node-status").ToInt(); err != nil {
		UpdateNodeStatusBuffer = constants.DefaultUpdateNodeStatusBuffer
	} else {
		UpdateNodeStatusBuffer = nsb
	}
	log.LOGGER.Infof("Update node status buffer: %d", UpdateNodeStatusBuffer)

	if qcb, err := config.CONFIG.GetValue("controller.buffer.query-configmap").ToInt(); err != nil {
		QueryConfigMapBuffer = constants.DefaultQueryConfigMapBuffer
	} else {
		QueryConfigMapBuffer = qcb
	}
	log.LOGGER.Infof("query config map buffer: %d", QueryConfigMapBuffer)

	if qsb, err := config.CONFIG.GetValue("controller.buffer.query-secret").ToInt(); err != nil {
		QuerySecretBuffer = constants.DefaultQuerySecretBuffer
	} else {
		QuerySecretBuffer = qsb
	}
	log.LOGGER.Infof("query secret buffer: %d", QuerySecretBuffer)
}
