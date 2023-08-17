package http

import (
	"encoding/json"
	"fmt"

	"github.com/kubeedge/mapper-generator/pkg/common"
	"github.com/kubeedge/mapper-generator/pkg/global"
)

type PushMethod struct {
	HTTP *HTTPConfig `json:"http"`
}

type HTTPConfig struct {
	HostName    string `json:"hostName,omitempty"`
	Port        int    `json:"port,omitempty"`
	RequestPath string `json:"requestPath,omitempty"`
	Timeout     int    `json:"timeout,omitempty"`
}

func NewDataPanel(config json.RawMessage) (global.DataPanel, error) {
	httpConfig := new(HTTPConfig)
	err := json.Unmarshal(config, httpConfig)
	if err != nil {
		return nil, err
	}
	return &PushMethod{
		HTTP: httpConfig,
	}, nil
}

func (pm *PushMethod) InitPushMethod() error {
	// TODO add init code
	fmt.Println("Init Http")
	return nil
}

func (pm *PushMethod) Push(data *common.DataModel) {
	// TODO add push code
	url := fmt.Sprintf("%s%d/%s", pm.HTTP.HostName, pm.HTTP.Port, pm.HTTP.RequestPath)
	fmt.Printf("publish %v to: %s, timeout: %d", data.Value, url, pm.HTTP.Timeout)
}
