package config

import (
	"github.com/kubeedge/beehive/pkg/common/config"
	"github.com/kubeedge/beehive/pkg/common/log"
	"github.com/kubeedge/kubeedge/common/constants"
)

// UpdatePodStatusBuffer is the size of channel which save update pod status message from edge
var UpdatePodStatusBuffer int

// UpdateNodeStatusBuffer is the size of channel which save update node status message from edge
var UpdateNodeStatusBuffer int

// QueryConfigMapBuffer is the size of channel which save query configmap message from edge
var QueryConfigMapBuffer int

// QuerySecretBuffer is the size of channel which save query secret message from edge
var QuerySecretBuffer int

// QueryServiceBuffer is the size of channel which save query service message from edge
var QueryServiceBuffer int

// QueryEndpointsBuffer is the size of channel which save query endpoints message from edge
var QueryEndpointsBuffer int

// PodEventBuffer is the size of channel which save pod event from k8s
var PodEventBuffer int

// ConfigMapEventBuffer is the size of channel which save configmap event from k8s
var ConfigMapEventBuffer int

// SecretEventBuffer is the size of channel which save secret event from k8s
var SecretEventBuffer int

// ServiceEventBuffer is the size of channel which save service event from k8s
var ServiceEventBuffer int

// EndpointsEventBuffer is the size of channel which save endpoints event from k8s
var EndpointsEventBuffer int

func init() {
	if psb, err := config.CONFIG.GetValue("controller.buffer.update-pod-status").ToInt(); err != nil {
		UpdatePodStatusBuffer = constants.DefaultUpdatePodStatusBuffer
	} else {
		UpdatePodStatusBuffer = psb
	}
	log.LOGGER.Infof("Update controller.buffer.update-pod-status: %d", UpdatePodStatusBuffer)

	if nsb, err := config.CONFIG.GetValue("controller.buffer.update-node-status").ToInt(); err != nil {
		UpdateNodeStatusBuffer = constants.DefaultUpdateNodeStatusBuffer
	} else {
		UpdateNodeStatusBuffer = nsb
	}
	log.LOGGER.Infof("Update controller.buffer.update-node-status: %d", UpdateNodeStatusBuffer)

	if qcb, err := config.CONFIG.GetValue("controller.buffer.query-configmap").ToInt(); err != nil {
		QueryConfigMapBuffer = constants.DefaultQueryConfigMapBuffer
	} else {
		QueryConfigMapBuffer = qcb
	}
	log.LOGGER.Infof("Update controller.buffer.query-configmap: %d", QueryConfigMapBuffer)

	if qsb, err := config.CONFIG.GetValue("controller.buffer.query-secret").ToInt(); err != nil {
		QuerySecretBuffer = constants.DefaultQuerySecretBuffer
	} else {
		QuerySecretBuffer = qsb
	}

	if qsb, err := config.CONFIG.GetValue("controller.buffer.query-service").ToInt(); err != nil {
		QueryServiceBuffer = constants.DefaultQueryServiceBuffer
	} else {
		QueryServiceBuffer = qsb
	}

	log.LOGGER.Infof("Update controller.buffer.query-service: %d", QueryServiceBuffer)

	if qeb, err := config.CONFIG.GetValue("controller.buffer.query-endpoints").ToInt(); err != nil {
		QueryEndpointsBuffer = constants.DefaultQueryEndpointsBuffer
	} else {
		QueryEndpointsBuffer = qeb
	}

	log.LOGGER.Infof("Update controller.buffer.query-endpoints: %d", QueryEndpointsBuffer)

	if peb, err := config.CONFIG.GetValue("controller.buffer.pod-event").ToInt(); err != nil {
		PodEventBuffer = constants.DefaultPodEventBuffer
	} else {
		PodEventBuffer = peb
	}
	log.LOGGER.Infof("Update controller.buffer.pod-event: %d", PodEventBuffer)

	if cmeb, err := config.CONFIG.GetValue("controller.buffer.configmap-event").ToInt(); err != nil {
		ConfigMapEventBuffer = constants.DefaultConfigMapEventBuffer
	} else {
		ConfigMapEventBuffer = cmeb
	}
	log.LOGGER.Infof("Update controller.buffer.configmap-event: %d", ConfigMapEventBuffer)

	if seb, err := config.CONFIG.GetValue("controller.buffer.secret-event").ToInt(); err != nil {
		SecretEventBuffer = constants.DefaultSecretEventBuffer
	} else {
		SecretEventBuffer = seb
	}
	log.LOGGER.Infof("Update controller.buffer.secret-event: %d", SecretEventBuffer)

	if seb, err := config.CONFIG.GetValue("controller.buffer.service-event").ToInt(); err != nil {
		ServiceEventBuffer = constants.DefaultServiceEventBuffer
	} else {
		ServiceEventBuffer = seb
	}
	log.LOGGER.Infof("Update controller.buffer.service-event: %d", ServiceEventBuffer)

	if epb, err := config.CONFIG.GetValue("controller.buffer.endpoints-event").ToInt(); err != nil {
		EndpointsEventBuffer = constants.DefaultEndpointsEventBuffer
	} else {
		EndpointsEventBuffer = epb
	}
	log.LOGGER.Infof("Update controller.buffer.endpoint-event: %d", EndpointsEventBuffer)
}
