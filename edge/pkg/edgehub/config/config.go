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
	CloudhubURL     string
	ProjectID       string
	NodeID          string
}

//EdgeHubConfig edge hub configuration object containing web socket and controller configuration
type EdgeHubConfig struct {
	WSConfig  WebSocketConfig
	CtrConfig ControllerConfig
	QcConfig  QuicConfig
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

var edgeHubConfig EdgeHubConfig

// InitEdgehubConfig init edgehub config
func InitEdgehubConfig() {
	err := getControllerConfig()
	if err != nil {
		klog.Errorf("Error in loading Controller configurations in edge hub:  %v", err)
		panic(err)
	}
	if edgeHubConfig.CtrConfig.Protocol == protocolWebsocket {
		err = getWebSocketConfig()
		if err != nil {
			klog.Errorf("Error in loading Web Socket configurations in edgehub:  %v", err)
			panic(err)
		}
	} else if edgeHubConfig.CtrConfig.Protocol == protocolQuic {
		err = getQuicConfig()
		if err != nil {
			klog.Errorf("Error in loading Quic configurations in edge hub:  %v", err)
			panic(err)
		}
	} else {
		panic(fmt.Errorf("error in loading Controller configurations, protocol %s is invalid", edgeHubConfig.CtrConfig.Protocol))
	}
}

//GetConfig returns the EdgeHub configuration object
func GetConfig() *EdgeHubConfig {
	return &edgeHubConfig
}

func getWebSocketConfig() error {
	url, err := config.CONFIG.GetValue("edgehub.websocket.url").ToString()
	if err != nil {
		klog.Errorf("Failed to get url for web socket client: %v", err)
	}
	edgeHubConfig.WSConfig.URL = url

	certFile, err := config.CONFIG.GetValue("edgehub.websocket.certfile").ToString()
	if err != nil {
		return fmt.Errorf("failed to get cert file for web socket client, error: %v", err)
	}
	edgeHubConfig.WSConfig.CertFilePath = certFile

	keyFile, err := config.CONFIG.GetValue("edgehub.websocket.keyfile").ToString()
	if err != nil {
		return fmt.Errorf("failed to get key file for web socket client, error: %v", err)
	}
	edgeHubConfig.WSConfig.KeyFilePath = keyFile

	writeDeadline, err := config.CONFIG.GetValue("edgehub.websocket.write-deadline").ToInt()
	if err != nil {
		klog.Warningf("Failed to get write-deadline for web socket client: %v", err)
		writeDeadline = writeDeadlineDefault
	}
	edgeHubConfig.WSConfig.WriteDeadline = time.Duration(writeDeadline) * time.Second

	readDeadline, err := config.CONFIG.GetValue("edgehub.websocket.read-deadline").ToInt()
	if err != nil {
		klog.Warningf("Failed to get read-deadline for web socket client: %v", err)
		readDeadline = readDeadlineDefault
	}
	edgeHubConfig.WSConfig.ReadDeadline = time.Duration(readDeadline) * time.Second

	handshakeTimeout, err := config.CONFIG.GetValue("edgehub.websocket.handshake-timeout").ToInt()
	if err != nil {
		klog.Warningf("Failed to get handshake-timeout for web socket client: %v", err)
		handshakeTimeout = handshakeTimeoutDefault
	}
	edgeHubConfig.WSConfig.HandshakeTimeout = time.Duration(handshakeTimeout) * time.Second

	return nil
}

func getControllerConfig() error {
	protocol, err := config.CONFIG.GetValue("edgehub.controller.protocol").ToString()
	if err != nil {
		klog.Warningf("Failed to get protocol for controller client: %v", err)
		protocol = protocolDefault
	}
	edgeHubConfig.CtrConfig.Protocol = protocol

	heartbeat, err := config.CONFIG.GetValue("edgehub.controller.heartbeat").ToInt()
	if err != nil {
		klog.Warningf("Failed to get heartbeat for controller client: %v", err)
		heartbeat = heartbeatDefault
	}
	edgeHubConfig.CtrConfig.HeartbeatPeriod = time.Duration(heartbeat) * time.Second

	projectID, err := config.CONFIG.GetValue("edgehub.controller.project-id").ToString()
	if err != nil {
		return fmt.Errorf("failed to get project id for controller client: %v", err)
	}
	edgeHubConfig.CtrConfig.ProjectID = projectID

	nodeID, err := config.CONFIG.GetValue("edgehub.controller.node-id").ToString()
	if err != nil {
		return fmt.Errorf("failed to get node id for controller client: %v", err)
	}
	edgeHubConfig.CtrConfig.NodeID = nodeID

	return nil
}

func getQuicConfig() error {
	url, err := config.CONFIG.GetValue("edgehub.quic.url").ToString()
	if err != nil {
		return fmt.Errorf("Failed to get url for quic client: %v", err)
	}
	edgeHubConfig.QcConfig.URL = url

	caFile, err := config.CONFIG.GetValue("edgehub.quic.cafile").ToString()
	if err != nil {
		return fmt.Errorf("failed to get cert file for quic client, error: %v", err)
	}
	edgeHubConfig.QcConfig.CaFilePath = caFile

	certFile, err := config.CONFIG.GetValue("edgehub.quic.certfile").ToString()
	if err != nil {
		return fmt.Errorf("failed to get cert file for quic client, error: %v", err)
	}
	edgeHubConfig.QcConfig.CertFilePath = certFile

	keyFile, err := config.CONFIG.GetValue("edgehub.quic.keyfile").ToString()
	if err != nil {
		return fmt.Errorf("failed to get key file for quic client, error: %v", err)
	}
	edgeHubConfig.QcConfig.KeyFilePath = keyFile

	writeDeadline, err := config.CONFIG.GetValue("edgehub.quic.write-deadline").ToInt()
	if err != nil {
		klog.Warningf("Failed to get write-deadline for quic client: %v", err)
		writeDeadline = writeDeadlineDefault
	}
	edgeHubConfig.QcConfig.WriteDeadline = time.Duration(writeDeadline) * time.Second

	readDeadline, err := config.CONFIG.GetValue("edgehub.quic.read-deadline").ToInt()
	if err != nil {
		klog.Warningf("Failed to get read-deadline for quic client: %v", err)
		readDeadline = readDeadlineDefault
	}
	edgeHubConfig.QcConfig.ReadDeadline = time.Duration(readDeadline) * time.Second

	handshakeTimeout, err := config.CONFIG.GetValue("edgehub.quic.handshake-timeout").ToInt()
	if err != nil {
		klog.Warningf("Failed to get handshake-timeout for quic client: %v", err)
		handshakeTimeout = handshakeTimeoutDefault
	}
	edgeHubConfig.QcConfig.HandshakeTimeout = time.Duration(handshakeTimeout) * time.Second
	return nil
}

//////////////////////////

var c Configure
var once sync.Once

type Configure struct {
}

func InitConfigure() {
	once.Do(func() {
		var errs []error
		defer func() {
			if len(errs) != 0 {
				for _, e := range errs {
					klog.Errorf("%v", e)
				}
				klog.Error("init edgehub config error")
				os.Exit(1)
			} else {
				klog.Infof("init edgehub config successfully，config info %++v", c)
			}
		}()
	})
}

func Get() Configure {
	return c
}
