/*
Copyright 2025 The KubeEdge Authors.

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

package actions

import (
	"fmt"
	"time"

	"github.com/shirou/gopsutil/cpu"
	"github.com/shirou/gopsutil/disk"
	"github.com/shirou/gopsutil/v3/mem"

	operationsv1alpha2 "github.com/kubeedge/api/apis/operations/v1alpha2"
)

const (
	MaxCPUUsage  float64 = 80
	MaxMemUsage  float64 = 80
	MaxDiskUsage float64 = 80
)

var checkMapper = map[string]func() error{
	operationsv1alpha2.CheckItemCPU:  checkCPU,
	operationsv1alpha2.CheckItemMem:  checkMem,
	operationsv1alpha2.CheckItemDisk: checkDisk,
}

// PreCheck a general pre-check function used to execute node check items.
func PreCheck(checkItems []string) error {
	for _, item := range checkItems {
		fn, ok := checkMapper[item]
		if !ok {
			return fmt.Errorf("check item %s not support", item)
		}
		err := fn()
		if err != nil {
			return err
		}
	}
	return nil
}

func checkCPU() error {
	cpuUsage, err := cpu.Percent(100*time.Millisecond, false)
	if err != nil {
		return fmt.Errorf("failed to get cpu usage, err: %v", err)
	}
	if len(cpuUsage) != 1 {
		return fmt.Errorf("unexpected cpu usage length %d", len(cpuUsage))
	}
	if cpuUsage[0] > MaxCPUUsage {
		return fmt.Errorf("current cpu usage is %.2f, which exceeds the maximum allowed usage %.2f",
			cpuUsage[0], MaxCPUUsage)
	}
	return nil
}

func checkMem() error {
	memInfo, err := mem.VirtualMemory()
	if err != nil {
		return fmt.Errorf("failed to get virtual memory stat, err: %v", err)
	}
	if memInfo.UsedPercent > MaxMemUsage {
		return fmt.Errorf("current mem usage is %.2f, which exceeds the maximum allowed usage %.2f",
			memInfo.UsedPercent, MaxMemUsage)
	}
	return nil
}

func checkDisk() error {
	// TODO: First determine the disk usage of the root directory,
	// and then decide how to handle other more appropriate disks based on actual needs.
	usage, err := disk.Usage("/")
	if err != nil {
		return fmt.Errorf("failed to get disk usage, err: %v", err)
	}
	if usage.UsedPercent > MaxDiskUsage {
		return fmt.Errorf("current disk usage is %.2f, which exceeds the maximum allowed usage %.2f",
			usage.UsedPercent, MaxDiskUsage)
	}
	return nil
}
