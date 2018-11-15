package config

import (
	"fmt"
	"net/http"
	"time"

	"github.com/kubeedge/kubeedge/beehive/pkg/common/config"
	"github.com/kubeedge/kubeedge/beehive/pkg/common/log"
)

const (
	handshakeTimeoutDefault = 60
	readDeadlineDefault     = 15
	writeDeadlineDefault    = 15

	authInfoFilesPathDefault = "/var/IEF/secret"

	heartbeatDefault = 15
	refreshInterval  = 10
)

type WebSocketConfig struct {
	Url              string
	CertFilePath     string
	KeyFilePath      string
	HandshakeTimeout time.Duration
	ReadDeadline     time.Duration
	WriteDeadline    time.Duration
	ExtendHeader     http.Header
}

type ControllerConfig struct {
	HeartbeatPeroid time.Duration
	RefreshInterval time.Duration
	CloudhubURL     string
	AuthInfosPath   string
	PlacementUrl    string
	ProjectID       string
	NodeId          string
}

type DISClientConfig struct {
	ApiGatewayUrl string
	ApiVersion    string
	RegionID      string
	ProjectID     string
}

type EdgeHubConfig struct {
	WSConfig  WebSocketConfig
	CtrConfig ControllerConfig
}

var edgeHubConfig EdgeHubConfig

func init() {
	getWebSocketConfig()
	getControllerConfig()
}

func GetConfig() *EdgeHubConfig {
	return &edgeHubConfig
}

func getWebSocketConfig() error {
	url, err := config.CONFIG.GetValue("edgehub.websocket.url").ToString()
	if err != nil {
		log.LOGGER.Errorf("failed to get url for web socket client: %v", err)
		//return fmt.Errorf("failed to get url for web socket client, error: %v", err)
	}
	edgeHubConfig.WSConfig.Url = url

	certFile, err := config.CONFIG.GetValue("edgehub.websocket.certfile").ToString()
	if err != nil {
		log.LOGGER.Errorf("failed to get cert file for web socket client: %v", err)
		return fmt.Errorf("failed to get cert file for web socket client, error: %v", err)
	}
	edgeHubConfig.WSConfig.CertFilePath = certFile

	keyFile, err := config.CONFIG.GetValue("edgehub.websocket.keyfile").ToString()
	if err != nil {
		log.LOGGER.Errorf("failed to get key file for web socket client: %v", err)
		return fmt.Errorf("failed to get key file for web socket client, error: %v", err)
	}
	edgeHubConfig.WSConfig.KeyFilePath = keyFile

	writeDeadline, err := config.CONFIG.GetValue("edgehub.websocket.write-deadline").ToInt()
	if err != nil {
		log.LOGGER.Warnf("failed to get key file for web socket client")
		writeDeadline = writeDeadlineDefault
	}
	edgeHubConfig.WSConfig.WriteDeadline = time.Duration(writeDeadline) * time.Second

	readDeadline, err := config.CONFIG.GetValue("edgehub.websocket.write-deadline").ToInt()
	if err != nil {
		log.LOGGER.Warnf("failed to get key file for web socket client")
		readDeadline = readDeadlineDefault
	}
	edgeHubConfig.WSConfig.ReadDeadline = time.Duration(readDeadline) * time.Second

	handshakeTimeout, err := config.CONFIG.GetValue("edgehub.websocket.handshake-timeout").ToInt()
	if err != nil {
		log.LOGGER.Warnf("failed to get key file for web socket client")
		handshakeTimeout = handshakeTimeoutDefault
	}
	edgeHubConfig.WSConfig.HandshakeTimeout = time.Duration(handshakeTimeout) * time.Second

	edgeHubConfig.WSConfig.ExtendHeader = getExtendHeader()

	return nil
}

func getControllerConfig() {
	heartbeat, err := config.CONFIG.GetValue("edgehub.controller.heartbeat").ToInt()
	if err != nil {
		log.LOGGER.Warnf("failed to get key file for web socket client")
		heartbeat = handshakeTimeoutDefault
	}
	edgeHubConfig.CtrConfig.HeartbeatPeroid = time.Duration(heartbeat) * time.Second

	interval, err := config.CONFIG.GetValue("edgehub.controller.refresh-ak-sk-interval").ToInt()
	if err != nil {
		log.LOGGER.Warnf("failed to get key file for web socket client")
		interval = refreshInterval
	}
	edgeHubConfig.CtrConfig.RefreshInterval = time.Duration(interval) * time.Minute

	projectId, err := config.CONFIG.GetValue("edgehub.controller.project-id").ToString()
	if err != nil {
		log.LOGGER.Warnf("failed to get project id  for web socket client")
	}
	edgeHubConfig.CtrConfig.ProjectID = projectId

	nodeId, err := config.CONFIG.GetValue("edgehub.controller.node-id").ToString()
	if err != nil {
		log.LOGGER.Warnf("failed to get node id  for web socket client")
	}
	edgeHubConfig.CtrConfig.NodeId = nodeId

	placementUrl, err := config.CONFIG.GetValue("edgehub.controller.placement-url").ToString()
	if err != nil {
		log.LOGGER.Warnf("failed to get placement url  for web socket client")
	}
	edgeHubConfig.CtrConfig.PlacementUrl = placementUrl

	authInfoPath, err := config.CONFIG.GetValue("edgehub.controller.auth-info-files-path").ToString()
	if err != nil {
		log.LOGGER.Warnf("failed to get auth info files path")
		authInfoPath = authInfoFilesPathDefault
	}
	edgeHubConfig.CtrConfig.AuthInfosPath = authInfoPath
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
