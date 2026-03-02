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
	"errors"

	"github.com/kubeedge/modbus/driver"
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
	return errors.New("need to add the stream flag when make generate if you want to enable stream data processing.")
}
