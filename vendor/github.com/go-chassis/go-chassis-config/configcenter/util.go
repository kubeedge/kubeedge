package configcenter

import (
	"fmt"

	"github.com/go-chassis/go-chassis-config/serializers"
	"github.com/go-mesh/openlogging"
	"github.com/gorilla/websocket"
	"net/http"
	"os"
	"time"
)

//GetConfigs get KV from a event
func GetConfigs(actionData []byte) (map[string]interface{}, error) {
	configCenterEvent := new(Event)
	err := serializers.Decode(serializers.JsonEncoder, actionData, &configCenterEvent)
	if err != nil {
		openlogging.GetLogger().Errorf(fmt.Sprintf("error in unmarshalling data on event receive with error %s", err.Error()))
		return nil, err
	}
	sourceConfig := make(map[string]interface{})
	err = serializers.Decode(serializers.JsonEncoder, []byte(configCenterEvent.Value), &sourceConfig)
	if err != nil {
		openlogging.GetLogger().Errorf(fmt.Sprintf("error in unmarshalling config values %s", err.Error()))
		return nil, err
	}
	return sourceConfig, nil
}
func keepAlive(c *websocket.Conn, timeout time.Duration) {
	lastResponse := time.Now()
	c.SetPongHandler(func(msg string) error {
		lastResponse = time.Now()
		return nil
	})
	go func() {
		for {
			err := c.WriteMessage(websocket.PingMessage, []byte("keepalive"))
			if err != nil {
				return
			}
			time.Sleep(timeout / 2)
			if time.Now().Sub(lastResponse) > timeout {
				c.Close()
				return
			}
		}
	}()
}

//GetDefaultHeaders gets default headers
func GetDefaultHeaders(tenantName string) http.Header {
	headers := http.Header{
		HeaderContentType: []string{"application/json"},
		HeaderUserAgent:   []string{"cse-configcenter-client/1.0.0"},
		HeaderTenantName:  []string{tenantName},
	}
	if environmentConfig != "" {
		headers.Set(HeaderEnvironment, environmentConfig)
	}

	return headers
}

//Update the Base PATH and HEADERS Based on the version of ConfigCenter used.
func updateAPIPath(apiVersion string) {

	//Check for the env Name in Container to get Domain Name
	//Default value is  "default"
	projectID, isExsist := os.LookupEnv(envProjectID)
	if !isExsist {
		projectID = "default"
	}
	switch apiVersion {
	case "v3":
		ConfigMembersPath = "/v3/" + projectID + members
		ConfigPath = "/v3/" + projectID + getConfigAPI
		ConfigRefreshPath = "/v3/" + projectID + dynamicConfigAPI
	case "v2":
		ConfigMembersPath = "/members"
		ConfigPath = "/configuration/v2/items"
		ConfigRefreshPath = "/configuration/v2/refresh/items"
	default:
		ConfigMembersPath = "/v3/" + projectID + members
		ConfigPath = "/v3/" + projectID + getConfigAPI
		ConfigRefreshPath = "/v3/" + projectID + dynamicConfigAPI
	}
}
func isStatusSuccess(i int) bool {
	return i >= http.StatusOK && i < http.StatusBadRequest
}
