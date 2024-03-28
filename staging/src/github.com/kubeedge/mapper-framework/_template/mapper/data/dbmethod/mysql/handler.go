package mysql

import (
	"context"
	"time"

	"k8s.io/klog/v2"

	"github.com/kubeedge/Template/driver"
	"github.com/kubeedge/mapper-framework/pkg/common"
)

func DataHandler(ctx context.Context, twin *common.Twin, client *driver.CustomizedClient, visitorConfig *driver.VisitorConfig, dataModel *common.DataModel) {
	dbConfig, err := NewDataBaseClient(twin.Property.PushMethod.DBMethod.DBConfig.MySQLClientConfig)
	if err != nil {
		klog.Errorf("new database client error: %v", err)
		return
	}
	err = dbConfig.InitDbClient()
	if err != nil {
		klog.Errorf("init redis database client err: %v", err)
		return
	}
	reportCycle := time.Duration(twin.Property.ReportCycle)
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
					klog.Errorf("mysql database add data error: %v", err)
					return
				}
			case <-ctx.Done():
				dbConfig.CloseSession()
				return
			}
		}
	}()
}
