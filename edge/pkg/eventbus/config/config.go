package config

import (
	"fmt"
	"os"
	"sync"

	"github.com/kubeedge/beehive/pkg/common/config"

	"k8s.io/klog"
)

const (
	defaultInternalMqttURL  = "tcp://127.0.0.1:1884"
	defaultExternalMqttURL  = "tcp://127.0.0.1:1883"
	defaultQos              = 0
	defaultRetain           = false
	defaultSessionQueueSize = 100
)

const (
	InternalMqttMode = iota // 0: launch an internal mqtt broker.
	BothMqttMode            // 1: launch an internal and external mqtt broker.
	ExternalMqttMode        // 2: launch an external mqtt broker.
)

var c Configure
var once sync.Once

type Configure struct {
	ExternalMqttURL  string
	InternalMqttURL  string
	QOS              int
	Retain           bool
	SessionQueueSize int
	Mode             int
	NodeID           string
}

func InitConfigure() {
	once.Do(func() {
		var errs []error
		defer func() {
			if len(errs) != 0 {
				for _, e := range errs {
					klog.Errorf("%v", e)
				}
				klog.Error("init eventbus config error")
				os.Exit(1)
			} else {
				klog.Infof("init eventbus config successfully，config info %++v", c)
			}
		}()
		internalMqttURL, err := config.CONFIG.GetValue("mqtt.internal-server").ToString()
		if err != nil {
			// Guaranteed forward compatibility @kadisi
			internalMqttURL = defaultInternalMqttURL
			klog.Infof("can not get mqtt.internal-server key, use default %v", internalMqttURL)
		}

		qos, err := config.CONFIG.GetValue("mqtt.qos").ToInt()
		if err != nil {
			// Guaranteed forward compatibility @kadisi
			qos = defaultQos
			klog.Infof("can not get mqtt.qos key, use default %v", qos)
		}
		retain, err := config.CONFIG.GetValue("mqtt.retain").ToBool()
		if err != nil {
			// Guaranteed forward compatibility @kadisi
			retain = defaultRetain
			klog.Infof("can not get mqtt.retain key, use default %v", retain)
		}
		sessionQueueSize, err := config.CONFIG.GetValue("mqtt.session-queue-size").ToInt()
		if err != nil {
			// Guaranteed forward compatibility @kadisi
			sessionQueueSize = defaultSessionQueueSize
			klog.Infof("can not get mqtt.session-queue-size key, use default %v", sessionQueueSize)
		}

		mode, err := config.CONFIG.GetValue("mqtt.mode").ToInt()
		if err != nil || mode > ExternalMqttMode || mode < InternalMqttMode {
			// Guaranteed forward compatibility @kadisi
			mode = InternalMqttMode
			klog.Infof("can not get mqtt.mode key, use default %v", mode)
		}
		externalMqttURL, err := config.CONFIG.GetValue("mqtt.server").ToString()
		if err != nil {
			// Guaranteed forward compatibility @kadisi
			externalMqttURL = defaultExternalMqttURL
			klog.Infof("can not get mqtt.server key, use default %v", externalMqttURL)
		}
		nodeID, err := config.CONFIG.GetValue("edgehub.controller.node-id").ToString()
		if err != nil {
			errs = append(errs, fmt.Errorf("get edgehub.controller.node-id key error %v", err))
		}

		c = Configure{
			ExternalMqttURL:  externalMqttURL,
			InternalMqttURL:  internalMqttURL,
			QOS:              qos,
			Retain:           retain,
			SessionQueueSize: sessionQueueSize,
			Mode:             mode,
			NodeID:           nodeID,
		}
	})

}
func Get() Configure {
	return c
}
