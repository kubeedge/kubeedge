/*
Copyright 2024 The KubeEdge Authors.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package stream

import (
	"encoding/json"
	"fmt"

	"k8s.io/klog/v2"

	"github.com/kubeedge/Template/driver"
	"github.com/kubeedge/mapper-framework/pkg/common"
)

type StreamConfig struct {
	Format        string `json:"format"`
	OutputDir     string `json:"outputDir"`
	FrameCount    int    `json:"frameCount"`
	FrameInterval int    `json:"frameInterval"`
	VideoNum      int    `json:"videoNum"`
}

func StreamHandler(twin *common.Twin, client *driver.CustomizedClient, visitorConfig *driver.VisitorConfig) error {
	// Get RTSP URI from camera device
	streamURI, err := client.GetDeviceData(visitorConfig)
	if err != nil {
		return err
	}

	// parse streamConfig data from device visitorConfig
	var streamConfig StreamConfig
	visitorConfigData, err := json.Marshal(visitorConfig.VisitorConfigData)
	err = json.Unmarshal(visitorConfigData, &streamConfig)
	if err != nil {
		return fmt.Errorf("Unmarshal streamConfigs error: %v", err)
	}

	switch twin.PropertyName {
	// Currently, the function of saving frames and saving videos is built-in according to the configuration.
	// Other functions can be expanded here.
	case common.SaveFrame:
		err = SaveFrame(streamURI.(string), streamConfig.OutputDir, streamConfig.Format, streamConfig.FrameCount, streamConfig.FrameInterval)
	case common.SaveVideo:
		err = SaveVideo(streamURI.(string), streamConfig.OutputDir, streamConfig.Format, streamConfig.FrameCount, streamConfig.VideoNum)
	default:
		err = fmt.Errorf("cannot find the processing method for the corresponding Property %s of the stream data", twin.PropertyName)
	}
	if err != nil {
		return err
	}
	klog.V(2).Infof("Successfully processed streaming data by %s", twin.PropertyName)
	return nil
}
