package config

import (
	"fmt"
	"os"
	"sync"
	"time"

	"k8s.io/klog"

	"github.com/kubeedge/beehive/pkg/common/config"
)

const (
	handshakeTimeoutDefault = 60
	readDeadlineDefault     = 15
	writeDeadlineDefault    = 15

	heartbeatDefault = 15

	protocolDefault   = protocolWebsocket
	protocolWebsocket = "websocket"
	protocolQuic      = "quic"
)

var c Configure
var once sync.Once

//WebSocketConfig defines web socket configuration object type
type WebSocketConfig struct {
	URL              string
	CertFilePath     string
	KeyFilePath      string
	HandshakeTimeout time.Duration
	ReadDeadline     time.Duration
	WriteDeadline    time.Duration
}

//ControllerConfig defines controller configuration object type
type ControllerConfig struct {
	Protocol        string
	HeartbeatPeriod time.Duration
	ProjectID       string
	NodeID          string
}

//QuicConfig defines quic configuration object type
type QuicConfig struct {
	URL              string
	CaFilePath       string
	CertFilePath     string
	KeyFilePath      string
	HandshakeTimeout time.Duration
	ReadDeadline     time.Duration
	WriteDeadline    time.Duration
}

type Configure struct {
	WSConfig  WebSocketConfig
	CtrConfig ControllerConfig
	QcConfig  QuicConfig
}

func InitConfigure() {
	once.Do(func() {
		var errs []error
		protocol, err := config.CONFIG.GetValue("edgehub.controller.protocol").ToString()
		if err != nil {
			// Guaranteed forward compatibility @kadisi
			protocol = protocolDefault
			klog.Infof("can not get edgehub.controller.protocol key, use default %v", protocol)
		}
		heartbeat, err := config.CONFIG.GetValue("edgehub.controller.heartbeat").ToInt()
		if err != nil {
			// Guaranteed forward compatibility @kadisi
			heartbeat = heartbeatDefault
			klog.Infof("can not get edgehub.controller.heartbeat key, use default %v", heartbeat)
		}
		projectID, err := config.CONFIG.GetValue("edgehub.controller.project-id").ToString()
		if err != nil {
			// Guaranteed forward compatibility @kadisi
			errs = append(errs, fmt.Errorf("get edgehub.controller.project-id key error %v", err))
		}
		nodeID, err := config.CONFIG.GetValue("edgehub.controller.node-id").ToString()
		if err != nil {
			// Guaranteed forward compatibility @kadisi
			errs = append(errs, fmt.Errorf("get edgehub.controller.node-id key error %v", err))
		}
		websocketURL, err := config.CONFIG.GetValue("edgehub.websocket.url").ToString()
		if err != nil {
			// Guaranteed forward compatibility @kadisi
			errs = append(errs, fmt.Errorf("get edgehub.websocket.url key error %v", err))
		}
		websocketCertFile, err := config.CONFIG.GetValue("edgehub.websocket.certfile").ToString()
		if err != nil {
			// Guaranteed forward compatibility @kadisi
			errs = append(errs, fmt.Errorf("get edgehub.websocket.certfile key error %v", err))
		}
		websocketKeyFile, err := config.CONFIG.GetValue("edgehub.websocket.keyfile").ToString()
		if err != nil {
			// Guaranteed forward compatibility @kadisi
			errs = append(errs, fmt.Errorf("get edgehub.websocket.keyfile key error %v", err))
		}
		websocketWriteDeadline, err := config.CONFIG.GetValue("edgehub.websocket.write-deadline").ToInt()
		if err != nil {
			// Guaranteed forward compatibility @kadisi
			websocketWriteDeadline = writeDeadlineDefault
			klog.Infof("can not get edgehub.websocket.write-deadline key, use default %v", websocketWriteDeadline)
		}
		websocketReadDeadline, err := config.CONFIG.GetValue("edgehub.websocket.read-deadline").ToInt()
		if err != nil {
			// Guaranteed forward compatibility @kadisi
			websocketReadDeadline = readDeadlineDefault
			klog.Infof("can not get edgehub.websocket.read-deadline key, use default %v", websocketReadDeadline)
		}
		websocketHandshakeTimeout, err := config.CONFIG.GetValue("edgehub.websocket.handshake-timeout").ToInt()
		if err != nil {
			// Guaranteed forward compatibility @kadisi
			websocketHandshakeTimeout = handshakeTimeoutDefault
			klog.Infof("can not get edgehub.websocket.handshake-timeout key, use default %v", websocketHandshakeTimeout)
		}
		quicURL, err := config.CONFIG.GetValue("edgehub.quic.url").ToString()
		if err != nil {
			// Guaranteed forward compatibility @kadisi
			errs = append(errs, fmt.Errorf("get edgehub.quic.url key error %v", err))
		}
		quickCAFile, err := config.CONFIG.GetValue("edgehub.quic.cafile").ToString()
		if err != nil {
			// Guaranteed forward compatibility @kadisi
			errs = append(errs, fmt.Errorf("get edgehub.quic.cafile key error %v", err))
		}
		quicCertFile, err := config.CONFIG.GetValue("edgehub.quic.certfile").ToString()
		if err != nil {
			// Guaranteed forward compatibility @kadisi
			errs = append(errs, fmt.Errorf("get edgehub.quic.certfile key error %v", err))
		}
		quickKeyFile, err := config.CONFIG.GetValue("edgehub.quic.keyfile").ToString()
		if err != nil {
			// Guaranteed forward compatibility @kadisi
			errs = append(errs, fmt.Errorf("get edgehub.quic.keyfile key error %v", err))
		}
		quickWriteDeadline, err := config.CONFIG.GetValue("edgehub.quic.write-deadline").ToInt()
		if err != nil {
			// Guaranteed forward compatibility @kadisi
			quickWriteDeadline = writeDeadlineDefault
			klog.Infof("can not get edgehub.quic.write-deadline key, use default %v", quickWriteDeadline)
		}
		quicReadDeadline, err := config.CONFIG.GetValue("edgehub.quic.read-deadline").ToInt()
		if err != nil {
			// Guaranteed forward compatibility @kadisi
			quicReadDeadline = readDeadlineDefault
			klog.Infof("can not  get edgehub.quic.read-deadline key, use default %v", quicReadDeadline)
		}
		quicHandshakeTimeout, err := config.CONFIG.GetValue("edgehub.quic.handshake-timeout").ToInt()
		if err != nil {
			// Guaranteed forward compatibility @kadisi
			quicHandshakeTimeout = handshakeTimeoutDefault
			klog.Infof("can to get edgehub.quic.handshake-timeout key, use default %v", quicHandshakeTimeout)
		}

		if len(errs) != 0 {
			for _, e := range errs {
				klog.Errorf("%v", e)
			}
			klog.Error("init edgehub config error")
			os.Exit(1)
		} else {
			klog.Infof("init edgehub config successfully，config info %++v", c)
		}

		c = Configure{
			WSConfig: WebSocketConfig{
				URL:              websocketURL,
				CertFilePath:     websocketCertFile,
				KeyFilePath:      websocketKeyFile,
				WriteDeadline:    time.Duration(websocketWriteDeadline) * time.Second,
				ReadDeadline:     time.Duration(websocketReadDeadline) * time.Second,
				HandshakeTimeout: time.Duration(websocketHandshakeTimeout) * time.Second,
			},
			CtrConfig: ControllerConfig{
				Protocol:        protocol,
				HeartbeatPeriod: time.Duration(heartbeat) * time.Second,
				ProjectID:       projectID,
				NodeID:          nodeID,
			},
			QcConfig: QuicConfig{
				URL:              quicURL,
				CaFilePath:       quickCAFile,
				CertFilePath:     quicCertFile,
				KeyFilePath:      quickKeyFile,
				WriteDeadline:    time.Duration(quickWriteDeadline) * time.Second,
				ReadDeadline:     time.Duration(quicReadDeadline) * time.Second,
				HandshakeTimeout: time.Duration(quicHandshakeTimeout) * time.Second,
			},
		}
	})
}

func Get() *Configure {
	return &c
}
