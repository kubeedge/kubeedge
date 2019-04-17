package config

import (
	"fmt"
	"net/http"
	"time"

	"github.com/kubeedge/beehive/pkg/common/config"
	"github.com/kubeedge/beehive/pkg/common/log"
)

const (
	handshakeTimeoutDefault = 60
	readDeadlineDefault     = 15
	writeDeadlineDefault    = 15

	authInfoFilesPathDefault = "/var/IEF/secret"

	heartbeatDefault = 15
	refreshInterval  = 10

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
	ExtendHeader     http.Header
}

//ControllerConfig defines controller configuration object type
type ControllerConfig struct {
	Protocol        string
	HeartbeatPeriod time.Duration
	RefreshInterval time.Duration
	CloudhubURL     string
	AuthInfosPath   string
	PlacementURL    string
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

func init() {
	err := getControllerConfig()
	if err != nil {
		log.LOGGER.Errorf("Error in loading Controller configurations in edge hub:  %v", err)
		panic(err)
	}
	if edgeHubConfig.CtrConfig.Protocol == protocolWebsocket {
		err = getWebSocketConfig()
		if err != nil {
			log.LOGGER.Errorf("Error in loading Web Socket configurations in edgehub:  %v", err)
			panic(err)
		}
	} else if edgeHubConfig.CtrConfig.Protocol == protocolQuic {
		err = getQuicConfig()
		if err != nil {
			log.LOGGER.Errorf("Error in loading Quic configurations in edge hub:  %v", err)
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
		log.LOGGER.Errorf("Failed to get url for web socket client: %v", err)
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
		log.LOGGER.Warnf("Failed to get write-deadline for web socket client: %v", err)
		writeDeadline = writeDeadlineDefault
	}
	edgeHubConfig.WSConfig.WriteDeadline = time.Duration(writeDeadline) * time.Second

	readDeadline, err := config.CONFIG.GetValue("edgehub.websocket.read-deadline").ToInt()
	if err != nil {
		log.LOGGER.Warnf("Failed to get read-deadline for web socket client: %v", err)
		readDeadline = readDeadlineDefault
	}
	edgeHubConfig.WSConfig.ReadDeadline = time.Duration(readDeadline) * time.Second

	handshakeTimeout, err := config.CONFIG.GetValue("edgehub.websocket.handshake-timeout").ToInt()
	if err != nil {
		log.LOGGER.Warnf("Failed to get handshake-timeout for web socket client: %v", err)
		handshakeTimeout = handshakeTimeoutDefault
	}
	edgeHubConfig.WSConfig.HandshakeTimeout = time.Duration(handshakeTimeout) * time.Second

	edgeHubConfig.WSConfig.ExtendHeader = getExtendHeader()

	return nil
}

func getControllerConfig() error {
	protocol, err := config.CONFIG.GetValue("edgehub.controller.protocol").ToString()
	if err != nil {
		log.LOGGER.Warnf("Failed to get protocol for controller client: %v", err)
		protocol = protocolDefault
	}
	edgeHubConfig.CtrConfig.Protocol = protocol

	heartbeat, err := config.CONFIG.GetValue("edgehub.controller.heartbeat").ToInt()
	if err != nil {
		log.LOGGER.Warnf("Failed to get heartbeat for controller client: %v", err)
		heartbeat = heartbeatDefault
	}
	edgeHubConfig.CtrConfig.HeartbeatPeriod = time.Duration(heartbeat) * time.Second

	interval, err := config.CONFIG.GetValue("edgehub.controller.refresh-ak-sk-interval").ToInt()
	if err != nil {
		log.LOGGER.Warnf("Failed to get refresh-ak-sk-interval for controller client: %v", err)
		interval = refreshInterval
	}
	edgeHubConfig.CtrConfig.RefreshInterval = time.Duration(interval) * time.Minute

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

	placementURL, err := config.CONFIG.GetValue("edgehub.controller.placement-url").ToString()
	if err != nil {
		return fmt.Errorf("failed to get placement url for controller client: %v", err)
	}
	edgeHubConfig.CtrConfig.PlacementURL = placementURL

	authInfoPath, err := config.CONFIG.GetValue("edgehub.controller.auth-info-files-path").ToString()
	if err != nil {
		log.LOGGER.Warnf("Failed to get auth info : %v", err)
		authInfoPath = authInfoFilesPathDefault
	}
	edgeHubConfig.CtrConfig.AuthInfosPath = authInfoPath

	return nil
}

func getExtendHeader() http.Header {
	header := http.Header{}
	if arch, err := config.CONFIG.GetValue("systeminfo.architecture").ToString(); err == nil {
		header.Add("Arch", arch)
	}
	if dockerRoot, err := config.CONFIG.GetValue("systeminfo.docker_root_dir").ToString(); err == nil {
		header.Add("DockerRootDir", dockerRoot)
	}
	log.LOGGER.Infof("websocket connection header is %v", header)

	return header
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
		log.LOGGER.Warnf("Failed to get write-deadline for quic client: %v", err)
		writeDeadline = writeDeadlineDefault
	}
	edgeHubConfig.QcConfig.WriteDeadline = time.Duration(writeDeadline) * time.Second

	readDeadline, err := config.CONFIG.GetValue("edgehub.quic.read-deadline").ToInt()
	if err != nil {
		log.LOGGER.Warnf("Failed to get read-deadline for quic client: %v", err)
		readDeadline = readDeadlineDefault
	}
	edgeHubConfig.QcConfig.ReadDeadline = time.Duration(readDeadline) * time.Second

	handshakeTimeout, err := config.CONFIG.GetValue("edgehub.quic.handshake-timeout").ToInt()
	if err != nil {
		log.LOGGER.Warnf("Failed to get handshake-timeout for quic client: %v", err)
		handshakeTimeout = handshakeTimeoutDefault
	}
	edgeHubConfig.QcConfig.HandshakeTimeout = time.Duration(handshakeTimeout) * time.Second
	return nil
}
