package dbmethod

import (
	"time"

	"github.com/kubeedge/Template/driver"
	"github.com/kubeedge/mapper-framework/pkg/common"
)

func GetReportCycle(twin *common.Twin) time.Duration {
	// get ReportCycle parameter
	reportCycle := time.Duration(twin.Property.ReportCycle)
	if reportCycle == 0 {
		reportCycle = common.DefaultReportCycle
	}
	return reportCycle
}

func GetDataModel(client *driver.CustomizedClient, visitorConfig *driver.VisitorConfig, dataModel *common.DataModel) (*common.DataModel, error) {
	// get and convert device data
	deviceData, err := client.GetDeviceData(visitorConfig)
	if err != nil {
		return nil, err
	}
	sData, err := common.ConvertToString(deviceData)
	if err != nil {
		return nil, err
	}
	dataModel.SetValue(sData)
	dataModel.SetTimeStamp()
	return dataModel, nil
}
