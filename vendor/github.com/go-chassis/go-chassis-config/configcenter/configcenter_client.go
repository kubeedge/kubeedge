/*
 * Copyright 2017 Huawei Technologies Co., Ltd
 *
 * Licensed under the Apache License, Version 2.0 (the "License");
 * you may not use this file except in compliance with the License.
 * You may obtain a copy of the License at
 *
 *    http://www.apache.org/licenses/LICENSE-2.0
 *
 * Unless required by applicable law or agreed to in writing, software
 * distributed under the License is distributed on an "AS IS" BASIS,
 * WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 * See the License for the specific language governing permissions and
 * limitations under the License.
 */

// Package memberdiscovery created on 2017/6/20.
package configcenter

import (
	"github.com/go-chassis/foundation/httpclient"
	"github.com/go-mesh/openlogging"

	"errors"
	"fmt"
	"github.com/go-chassis/go-chassis-config"
	"github.com/go-chassis/go-chassis-config/serializers"
	"github.com/gorilla/websocket"
	"net/http"
	"net/url"
	"strings"
	"time"
)

var (
	//HeaderTenantName is a variable of type string
	HeaderTenantName = "X-Tenant-Name"
	//ConfigMembersPath is a variable of type string
	ConfigMembersPath = ""
	//ConfigPath is a variable of type string
	ConfigPath = ""
	//ConfigRefreshPath is a variable of type string
	ConfigRefreshPath = ""
	//MemberDiscoveryService is a variable
	MemberDiscoveryService MemberDiscovery
	autoDiscoverable       = false
	apiVersionConfig       = ""
	environmentConfig      = ""
)

const (
	defaultTimeout = 10 * time.Second
	//StatusUP is a variable of type string
	StatusUP = "UP"
	//HeaderContentType is a variable of type string
	HeaderContentType = "Content-Type"
	//HeaderUserAgent is a variable of type string
	HeaderUserAgent = "User-Agent"
	//HeaderEnvironment specifies the environment of a service
	HeaderEnvironment        = "X-Environment"
	members                  = "/configuration/members"
	dimensionsInfo           = "dimensionsInfo"
	dynamicConfigAPI         = `/configuration/refresh/items`
	getConfigAPI             = `/configuration/items`
	defaultContentType       = "application/json"
	envProjectID             = "CSE_PROJECT_ID"
	packageInitError         = "package not initialize successfully"
	emptyConfigServerMembers = "empty config server member"
	emptyConfigServerConfig  = "empty config server passed"
	// Name of the Plugin
	Name = "config_center"
)

//Client is Client Implementation of Client
type Client struct {
	memDiscovery         *MemDiscovery
	refreshPort          string
	defaultDimensionInfo string
	wsDialer             *websocket.Dialer
	wsConnection         *websocket.Conn
}

//NewConfigCenter is a function
func NewConfigCenter(options config.Options) config.Client {
	memDiscovery := new(MemDiscovery)
	//memDiscovery.Logger = logger
	memDiscovery.TLSConfig = options.TLSConfig
	memDiscovery.TenantName = options.TenantName
	memDiscovery.EnableSSL = options.EnableSSL
	var apiVersion string
	apiVersionConfig = options.APIVersion
	autoDiscoverable = options.AutoDiscovery
	environmentConfig = options.Env

	switch apiVersionConfig {
	case "v2":
		apiVersion = "v2"
	case "V2":
		apiVersion = "v2"
	case "v3":
		apiVersion = "v3"
	case "V3":
		apiVersion = "v3"
	default:
		apiVersion = "v3"
	}
	//Update the API Base Path based on the Version
	updateAPIPath(apiVersion)

	//Initiate RestClient from http-client package
	opts := &httpclient.URLClientOption{
		SSLEnabled: options.EnableSSL,
		TLSConfig:  options.TLSConfig,
		Compressed: false,
	}
	memDiscovery.client, _ = httpclient.GetURLClient(opts)

	ccclient := &Client{
		memDiscovery: memDiscovery,
		refreshPort:  options.RefreshPort,
		wsDialer: &websocket.Dialer{
			TLSClientConfig:  options.TLSConfig,
			HandshakeTimeout: defaultTimeout,
		},
		defaultDimensionInfo: options.DimensionInfo,
	}

	configCenters := strings.Split(options.ServerURI, ",")
	cCenters := make([]string, 0)
	for _, value := range configCenters {
		value = strings.Replace(value, " ", "", -1)
		cCenters = append(cCenters, value)
	}
	memDiscovery.ConfigurationInit(cCenters)
	return ccclient
}

// PullConfigs is the implementation of Client to pull all the configurations from Config-Server
func (cclient *Client) PullConfigs(serviceName, version, app, env string) (map[string]interface{}, error) {
	// serviceName is the defaultDimensionInfo passed from Client (small hack)
	configurations, error := cclient.memDiscovery.pullConfigurationsFromServer(serviceName)
	if error != nil {
		return nil, error
	}
	return configurations, nil
}

// PullConfig is the implementation of Client to pull specific configurations from Config-Server
func (cclient *Client) PullConfig(serviceName, version, app, env, key, contentType string) (interface{}, error) {
	// serviceName is the defaultDimensionInfo passed from Client (small hack)
	// TODO use the contentType to return the configurations
	configurations, error := cclient.memDiscovery.pullConfigurationsFromServer(serviceName)
	if error != nil {
		return nil, error
	}
	configurationsValue, ok := configurations[key]
	if !ok {
		openlogging.GetLogger().Error("Error in fetching the configurations for particular value,No Key found : " + key)
	}

	return configurationsValue, nil
}

// PullConfigsByDI pulls the configuration for custom DimensionInfo
func (cclient *Client) PullConfigsByDI(dimensionInfo, diInfo string) (map[string]map[string]interface{}, error) {
	// update defaultDimensionInfo value
	type GetConfigAPI map[string]map[string]interface{}
	configAPIRes := make(GetConfigAPI)
	parsedDimensionInfo := strings.Replace(diInfo, "#", "%23", -1)
	restApi := ConfigPath + "?" + dimensionsInfo + "=" + parsedDimensionInfo
	err := cclient.memDiscovery.call(http.MethodGet, restApi, nil, nil, &configAPIRes)
	if err != nil {
		openlogging.GetLogger().Error("Pull config by DI failed:" + err.Error())
		return nil, err

	}
	return configAPIRes, nil
}

// PushConfigs push configs to ConfigSource cc , success will return { "Result": "Success" }
func (cclient *Client) PushConfigs(items map[string]interface{}, dimensionInfo string) (map[string]interface{}, error) {
	if len(items) == 0 {
		em := "data is empty , which data need to send cc"
		openlogging.GetLogger().Error(em)
		return nil, errors.New(em)
	}

	configApi := CreateConfigApi{
		DimensionInfo: dimensionInfo,
		Items:         items,
	}

	return cclient.addDeleteConfig(configApi, http.MethodPost)
}

// DeleteConfigsByKeys
func (cclient *Client) DeleteConfigsByKeys(keys []string, dimensionInfo string) (map[string]interface{}, error) {
	if len(keys) == 0 {
		em := "not key need to delete for cc, please check keys"
		openlogging.GetLogger().Error(em)
		return nil, errors.New(em)
	}

	configApi := DeleteConfigApi{
		DimensionInfo: dimensionInfo,
		Keys:          keys,
	}

	return cclient.addDeleteConfig(configApi, http.MethodDelete)
}

func (cclient *Client) addDeleteConfig(data interface{}, method string) (map[string]interface{}, error) {
	type ConfigAPI map[string]interface{}
	configAPIS := make(ConfigAPI)
	body, err := serializers.Encode(serializers.JsonEncoder, data)
	if err != nil {
		openlogging.GetLogger().Errorf("serializer data failed , err :", err.Error())
		return nil, err
	}
	err = cclient.memDiscovery.call(method, ConfigPath, nil, body, &configAPIS)
	if err != nil {
		return nil, err
	}
	return configAPIS, nil
}
func (cclient *Client) Watch(f func(map[string]interface{}), errHandler func(err error)) error {
	parsedDimensionInfo := strings.Replace(cclient.defaultDimensionInfo, "#", "%23", -1)
	refreshConfigPath := ConfigRefreshPath + `?` + dimensionsInfo + `=` + parsedDimensionInfo
	if cclient.wsDialer != nil {
		/*-----------------
		1. Decide on the URL
		2. Create WebSocket Connection
		3. Call KeepAlive in seperate thread
		3. Generate events on Recieve Data
		*/
		baseURL, err := cclient.getWebSocketURL()
		if err != nil {
			error := errors.New("error in getting default server info")
			return error
		}
		url := baseURL.String() + refreshConfigPath
		cclient.wsConnection, _, err = cclient.wsDialer.Dial(url, nil)
		if err != nil {
			return fmt.Errorf("watching config-center dial catch an exception error:%s", err.Error())
		}
		keepAlive(cclient.wsConnection, 15*time.Second)
		go func() error {
			for {
				messageType, message, err := cclient.wsConnection.ReadMessage()
				if err != nil {
					break
				}
				if messageType == websocket.TextMessage {
					m, err := GetConfigs(message)
					if err != nil {
						errHandler(err)
						continue
					}
					f(m)
				}
			}
			err = cclient.wsConnection.Close()
			if err != nil {
				openlogging.Error(err.Error())
				return fmt.Errorf("CC watch Conn close failed error:%s", err.Error())
			}
			return nil
		}()
	}
	return nil
}

func (cclient *Client) getWebSocketURL() (*url.URL, error) {
	var defaultTLS bool
	var parsedEndPoint []string
	var host string

	configCenterEntryPointList, err := cclient.memDiscovery.GetConfigServer()
	if err != nil {
		openlogging.GetLogger().Error("error in member discovery:" + err.Error())
		return nil, err
	}
	activeEndPointList, err := cclient.memDiscovery.GetWorkingConfigCenterIP(configCenterEntryPointList)
	if err != nil {
		openlogging.GetLogger().Error("failed to get ip list:" + err.Error())
	}
	for _, server := range activeEndPointList {
		parsedEndPoint = strings.Split(server, `://`)
		hostArr := strings.Split(parsedEndPoint[1], `:`)
		port := cclient.refreshPort
		if port == "" {
			port = "30104"
		}
		host = hostArr[0] + ":" + port
		if host == "" {
			host = "localhost"
		}
	}

	if cclient.wsDialer.TLSClientConfig != nil {
		defaultTLS = true
	}
	if host == "" {
		err := errors.New("host must be a URL or a host:port pair")
		openlogging.GetLogger().Error("empty host for watch action:" + err.Error())
		return nil, err
	}
	hostURL, err := url.Parse(host)
	if err != nil || hostURL.Scheme == "" || hostURL.Host == "" {
		scheme := "ws://"
		if defaultTLS {
			scheme = "wss://"
		}
		hostURL, err = url.Parse(scheme + host)
		if err != nil {
			return nil, err
		}
		if hostURL.Path != "" && hostURL.Path != "/" {
			return nil, fmt.Errorf("host must be a URL or a host:port pair: %q", host)
		}
	}
	return hostURL, nil
}
func init() {
	config.InstallConfigClientPlugin(Name, NewConfigCenter)
}
