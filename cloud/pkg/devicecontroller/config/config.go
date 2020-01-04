package config

import (
	"fmt"
	"os"
	"sync"

	"github.com/kubeedge/beehive/pkg/common/config"
	deviceconstants "github.com/kubeedge/kubeedge/cloud/pkg/devicecontroller/constants"
	"github.com/kubeedge/kubeedge/common/constants"
	"k8s.io/klog"
)

var c Configure
var once sync.Once

type Configure struct {
	// UpdateDeviceStatusBuffer is the size of channel which save update device status message from edge
	UpdateDeviceStatusBuffer int
	// DeviceEventBuffer is the size of channel which save device event from k8s
	DeviceEventBuffer int
	// DeviceModelEventBuffer is the size of channel which save devicemodel event from k8s
	DeviceModelEventBuffer int
	// ContextSendModule is the name send message to
	ContextSendModule string
	// ContextReceiveModule is the name receive message from
	ContextReceiveModule string
	// ContextResponseModule is the name response message from
	ContextResponseModule string
	// KubeMaster is the url of edge master(kube api server)
	KubeMaster string
	// KubeConfig is the config used connect to edge master
	KubeConfig string
	// KubeContentType is the content type communicate with edge master(default is "application/vnd.kubernetes.protobuf")
	KubeContentType string
	// KubeQPS is the QPS communicate with edge master(default is 1024)
	KubeQPS float32
	// KubeBurst default is 10
	KubeBurst int
	// UpdateDeviceStatusWorkers is the count of goroutines of update device status
	UpdateDeviceStatusWorkers int
}

func InitConfigure() {
	once.Do(func() {
		var errs []error

		km, err := config.CONFIG.GetValue("devicecontroller.kube.master").ToString()
		if err != nil {
			errs = append(errs, fmt.Errorf("get devicecontroller.kube.master configuration key error %v", err))
		}
		kc, err := config.CONFIG.GetValue("devicecontroller.kube.kubeconfig").ToString()
		if err != nil {
			errs = append(errs, fmt.Errorf("get devicecontroller.kube.kubeconfig configuration key error %v", err))
		}
		kct, err := config.CONFIG.GetValue("devicecontroller.kube.content_type").ToString()
		if err != nil {
			errs = append(errs, fmt.Errorf("get devicecontroller.kube.content_type configuration key error %v", err))
		}
		kqps, err := config.CONFIG.GetValue("devicecontroller.kube.qps").ToFloat64()
		if err != nil {
			errs = append(errs, fmt.Errorf("get devicecontroller.kube.qps configuration key error %v", err))
		}
		kb, err := config.CONFIG.GetValue("controller.kube.burst").ToInt()
		if err != nil {
			errs = append(errs, fmt.Errorf("get devicecontroller.kube.burst configuration key error %v", err))
		}
		smn, err := config.CONFIG.GetValue("devicecontroller.context.send-module").ToString()
		if err != nil {
			// Guaranteed forward compatibility @kadisi
			smn = deviceconstants.DefaultContextSendModuleName
			klog.Infof("can not get devicecontroller.context.send-module key, use default value %v", smn)
		}
		rmn, err := config.CONFIG.GetValue("devicecontroller.context.receive-module").ToString()
		if err != nil {
			// Guaranteed forward compatibility @kadisi
			rmn = deviceconstants.DefaultContextReceiveModuleName
			klog.Infof("can not get devicecontroller.context.receive-module key, use default value %v", rmn)
		}
		res, err := config.CONFIG.GetValue("devicecontroller.context.response-module").ToString()
		if err != nil {
			// Guaranteed forward compatibility @kadisi
			res = deviceconstants.DefaultContextResponseModuleName
			klog.Infof("can not get devicecontroller.context.response-module key, use default value %v", res)
		}
		uds, err := config.CONFIG.GetValue("devicecontroller.buffer.update-device-status").ToInt()
		if err != nil {
			// Guaranteed forward compatibility @kadisi
			uds = constants.DefaultUpdateDeviceStatusBuffer
			klog.Infof("can not get devicecontroller.buffer.update-device-status key, use default value %v", uds)
		}
		dbde, err := config.CONFIG.GetValue("devicecontroller.buffer.device-event").ToInt()
		if err != nil {
			// Guaranteed forward compatibility @kadisi
			dbde = constants.DefaultDeviceEventBuffer
			klog.Infof("can not get devicecontroller.buffer.device-event key, use default value %v", dbde)
		}
		dmeb, err := config.CONFIG.GetValue("devicecontroller.buffer.device-model-event").ToInt()
		if err != nil {
			// Guaranteed forward compatibility @kadisi
			dmeb = constants.DefaultDeviceModelEventBuffer
			klog.Infof("can not get devicecontroller.buffer.device-model-event key, use default value %v", dmeb)
		}
		psw, err := config.CONFIG.GetValue("devicecontroller.load.update-device-status-workers").ToInt()
		if err != nil {
			// Guaranteed forward compatibility @kadisi
			psw = constants.DefaultUpdateDeviceStatusWorkers
			klog.Infof("can not get devicecontroller.load.update-device-status-workers key, use default value %v", psw)
		}
		if len(errs) != 0 {
			for _, e := range errs {
				klog.Errorf("%v", e)
			}
			klog.Error("init devicecontroller config error")
			os.Exit(1)
		}
		c = Configure{
			KubeMaster:                km,
			KubeConfig:                kc,
			KubeContentType:           kct,
			KubeQPS:                   float32(kqps),
			KubeBurst:                 kb,
			ContextSendModule:         smn,
			ContextReceiveModule:      rmn,
			ContextResponseModule:     res,
			UpdateDeviceStatusBuffer:  uds,
			DeviceEventBuffer:         dbde,
			DeviceModelEventBuffer:    dmeb,
			UpdateDeviceStatusWorkers: psw,
		}
		klog.Infof("init devicecontroller config successfully, config info %++v", c)
	})
}

func Get() *Configure {
	return &c
}
