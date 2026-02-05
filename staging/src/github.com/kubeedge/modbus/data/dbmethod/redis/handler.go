/*
Copyright 2023 The KubeEdge Authors.

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

package redis

import (
	"context"
	"time"

	"k8s.io/klog/v2"

	"github.com/kubeedge/modbus/driver"
	"github.com/kubeedge/mapper-framework/pkg/common"
)

func DataHandler(ctx context.Context, twin *common.Twin, client *driver.CustomizedClient, visitorConfig *driver.VisitorConfig, dataModel *common.DataModel) {
	dbConfig, err := NewDataBaseClient(twin.Property.PushMethod.DBMethod.DBConfig.RedisClientConfig)
	if err != nil {
		klog.Errorf("new database client error: %v", err)
		return
	}
	err = dbConfig.InitDbClient()
	if err != nil {
		klog.Errorf("init redis database client err: %v", err)
		return
	}
	reportCycle := time.Millisecond * time.Duration(twin.Property.ReportCycle)
	if reportCycle == 0 {
		reportCycle = common.DefaultReportCycle
	}
	ticker := time.NewTicker(reportCycle)
	go func() {
		for {
			select {
			case <-ticker.C:
				deviceData, err := client.GetDeviceData(visitorConfig)
				if err != nil {
					klog.Errorf("publish error: %v", err)
					continue
				}
				sData, err := common.ConvertToString(deviceData)
				if err != nil {
					klog.Errorf("Failed to convert publish method data : %v", err)
					continue
				}
				dataModel.SetValue(sData)
				dataModel.SetTimeStamp()

				err = dbConfig.AddData(dataModel)
				if err != nil {
					klog.Errorf("redis database add data error: %v", err)
					return
				}
			case <-ctx.Done():
				dbConfig.CloseSession()
				return
			}
		}
	}()

}
