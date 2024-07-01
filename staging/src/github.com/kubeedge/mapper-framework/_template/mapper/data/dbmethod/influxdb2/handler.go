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

package influxdb2

import (
	"context"
	"time"

	"k8s.io/klog/v2"

	"github.com/kubeedge/Template/data/dbmethod"
	"github.com/kubeedge/Template/driver"
	"github.com/kubeedge/mapper-framework/pkg/common"
)

func DataHandler(ctx context.Context, twin *common.Twin, client *driver.CustomizedClient, visitorConfig *driver.VisitorConfig, dataModel *common.DataModel) {
	dbConfig, err := NewDataBaseClient(twin.Property.PushMethod.DBMethod.DBConfig.Influxdb2ClientConfig, twin.Property.PushMethod.DBMethod.DBConfig.Influxdb2DataConfig)
	if err != nil {
		klog.Errorf("new database client error: %v", err)
		return
	}
	dbClient := dbConfig.InitDbClient()
	if err != nil {
		klog.Errorf("init database client err: %v", err)
		return
	}
	ticker := time.NewTicker(dbmethod.GetReportCycle(twin))
	go func() {
		for {
			select {
			case <-ticker.C:
				dataModel, err = dbmethod.GetDataModel(client, visitorConfig, dataModel)
				if err != nil {
					klog.Errorf("get and convert device data error: %v", err)
					continue
				}
				err = dbConfig.AddData(dataModel, dbClient)
				if err != nil {
					klog.Errorf("influx database add data error: %v", err)
					return
				}
			case <-ctx.Done():
				dbConfig.CloseSession(dbClient)
				return
			}
		}
	}()
}
