package http

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strconv"
	"strings"
	"time"

	"k8s.io/klog/v2"

	"github.com/kubeedge/mapper-framework/pkg/common"
	"github.com/kubeedge/mapper-framework/pkg/global"
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
	klog.V(1).Info("Init HTTP")
	return nil
}

func (pm *PushMethod) Push(data *common.DataModel) {
	klog.V(2).Info("Publish device data by HTTP")

	targetUrl := pm.HTTP.HostName + ":" + strconv.Itoa(pm.HTTP.Port) + pm.HTTP.RequestPath
	payload := data.PropertyName + "=" + data.Value
	formatTimeStr := time.Unix(data.TimeStamp/1e3, 0).Format("2006-01-02 15:04:05")
	currentTime := "&time" + "=" + formatTimeStr
	payload += currentTime

	klog.V(3).Infof("Publish %v to %s", payload, targetUrl)

	resp, err := http.Post(targetUrl,
		"application/x-www-form-urlencoded",
		strings.NewReader(payload))

	if err != nil {
		fmt.Println(err)
	}
	defer resp.Body.Close()
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		// handle error
		klog.Errorf("Publish device data by HTTP failed, err = %v", err)
		return
	}
	klog.V(1).Info("###############  Message published.  ###############")
	klog.V(3).Infof("HTTP reviced %s", string(body))

}
