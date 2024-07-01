package device

import (
	"log"

	dmiapi "github.com/kubeedge/kubeedge/pkg/apis/dmi/v1beta1"
	"github.com/kubeedge/mapper-framework/pkg/grpcclient"
	"k8s.io/klog/v2"

	"github.com/kubeedge/Template/driver"
)

// GetStates is the timer structure for getting device states.
type DeviceStates struct {
	Client          *driver.CustomizedClient
	DeviceName      string
	DeviceNamespace string
}

// Run timer function.
func (gs *DeviceStates) Run() {
	states, error := gs.Client.GetDeviceStates()
	if error != nil {
		klog.Errorf("GetDeviceStates failed: %v", error)
		return
	}

	statesRequest := &dmiapi.ReportDeviceStatesRequest{
		DeviceName:      gs.DeviceName,
		State:           states,
		DeviceNamespace: gs.DeviceNamespace,
	}

	log.Printf("send statesRequest", statesRequest.DeviceName, statesRequest.State)
	if err := grpcclient.ReportDeviceStates(statesRequest); err != nil {
		klog.Errorf("fail to report device states of %s with err: %+v", gs.DeviceName, err)
	}
}
