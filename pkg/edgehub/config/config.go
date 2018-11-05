package config

import (
	"net/http"
	"time"

	"kubeedge/beehive/pkg/common/config"
	"kubeedge/beehive/pkg/common/log"
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
	HandshakeTimeout time.Duration
	ReadDeadline     time.Duration
	WriteDeadline    time.Duration
	ExtendHeader     http.Header
}

type ControllerConfig struct {
	HeartbeatPeroid time.Duration
	CloudhubURL     string
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

	cloudhubURL, err := config.CONFIG.GetValue("edgehub.controller.cloudhub-url").ToString()
	if err != nil {
		log.LOGGER.Warnf("failed to get cloudhub url  for web socket client")
	}
	edgeHubConfig.CtrConfig.CloudhubURL = cloudhubURL
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
