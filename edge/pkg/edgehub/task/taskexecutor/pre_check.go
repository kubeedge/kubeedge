package taskexecutor

import (
	"encoding/json"
	"fmt"
	"time"

	"github.com/shirou/gopsutil/cpu"
	"github.com/shirou/gopsutil/disk"
	"github.com/shirou/gopsutil/v3/mem"

	"github.com/kubeedge/kubeedge/common/types"
	api "github.com/kubeedge/kubeedge/pkg/apis/fsm/v1alpha1"
	"github.com/kubeedge/kubeedge/pkg/util/fsm"
)

const (
	MaxCPUUsage  float64 = 80
	MaxMemUsage  float64 = 80
	MaxDiskUsage float64 = 80
)

func preCheck(taskReq types.NodeTaskRequest) fsm.Event {
	event := fsm.Event{
		Type:     "Check",
		Action:   api.ActionSuccess,
		ErrorMsg: "",
	}

	data, err := json.Marshal(taskReq.Item)
	if err != nil {
		event.Action = api.ActionFailure
		event.ErrorMsg = err.Error()
		return event
	}
	var checkItems types.NodePreCheckRequest
	err = json.Unmarshal(data, &checkItems)
	if err != nil {
		event.Action = api.ActionFailure
		event.ErrorMsg = err.Error()
		return event
	}

	var failed bool
	var checkResult = map[string]string{}
	var checkFunc = map[string]func() error{
		"cpu":  checkCPU,
		"mem":  checkMem,
		"disk": checkDisk,
	}
	for _, item := range checkItems.CheckItem {
		f, ok := checkFunc[item]
		if !ok {
			checkResult[item] = "check item not support"
			continue
		}
		err = f()
		if err != nil {
			failed = true
			checkResult[item] = err.Error()
			continue
		}
		checkResult[item] = "ok"
	}
	if !failed {
		return event
	}
	event.Action = api.ActionFailure
	result, err := json.Marshal(checkResult)
	if err != nil {
		event.ErrorMsg = err.Error()
		return event
	}
	event.ErrorMsg = string(result)
	return event
}

func checkCPU() error {
	cpuUsage, err := cpu.Percent(100*time.Millisecond, false)
	if err != nil {
		return err
	}
	var usage float64
	for _, percpu := range cpuUsage {
		usage += percpu / float64(len(cpuUsage))
	}
	if usage > MaxCPUUsage {
		return fmt.Errorf("current cpu usage is %f, which exceeds the maximum allowed usage %f", usage, MaxCPUUsage)
	}
	return nil
}

func checkMem() error {
	memInfo, err := mem.VirtualMemory()
	if err != nil {
		return err
	}
	memUsedPercent := memInfo.UsedPercent
	if memUsedPercent > MaxMemUsage {
		return fmt.Errorf("current mem usage is %f, which exceeds the maximum allowed usage %f", memUsedPercent, MaxMemUsage)
	}
	return nil
}

func checkDisk() error {
	partitions, err := disk.Partitions(true)
	if err != nil {
		return err
	}
	var failed bool
	var diskUsages = map[string]string{}
	for _, part := range partitions {
		usage, err := disk.Usage(part.Mountpoint)
		if err != nil {
			failed = true
			diskUsages[part.Device] = err.Error()
			continue
		}
		if usage.UsedPercent > MaxDiskUsage {
			failed = true
			diskUsages[part.Device] = fmt.Sprintf("current disk usage is %f, which exceeds the maximum allowed usage %f", usage.UsedPercent, MaxMemUsage)
			continue
		}
		diskUsages[part.Device] = fmt.Sprintf("%f", usage.UsedPercent)
	}
	if !failed {
		return nil
	}
	result, err := json.Marshal(diskUsages)
	if err != nil {
		return err
	}
	return fmt.Errorf(string(result))
}

func normalInit(types.NodeTaskRequest) fsm.Event {
	return fsm.Event{
		Type:     "Init",
		Action:   api.ActionSuccess,
		ErrorMsg: "",
	}
}
