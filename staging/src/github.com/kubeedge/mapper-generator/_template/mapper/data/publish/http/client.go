package http

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"k8s.io/klog/v2"
	"net/http"
	"strconv"
	"strings"
	"time"

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

	targetUrl := pm.HTTP.HostName + ":" + strconv.Itoa(pm.HTTP.Port) + pm.HTTP.RequestPath
	klog.V(1).Infof("targetUrl = %s", targetUrl)
	payload := data.PropertyName + "=" + data.Value
	formatTimeStr := time.Unix(data.TimeStamp/1e3, 0).Format("2006-01-02 15:04:05")
	currentTime := "&time" + "=" + formatTimeStr
	payload += currentTime
	resp, err := http.Post(targetUrl,
		"application/x-www-form-urlencoded",
		strings.NewReader(payload))

	if err != nil {
		fmt.Println(err)
	}
	defer resp.Body.Close()
	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		// handle error
	}
	klog.V(1).Info(string(body))

}
