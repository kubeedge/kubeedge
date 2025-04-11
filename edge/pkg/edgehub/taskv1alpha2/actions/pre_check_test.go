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
	"errors"
	"fmt"
	"testing"
	"time"

	"github.com/agiledragon/gomonkey/v2"
	"github.com/shirou/gopsutil/cpu"
	"github.com/shirou/gopsutil/disk"
	"github.com/shirou/gopsutil/v3/mem"
	"github.com/stretchr/testify/assert"
)

func TestPreCheck(t *testing.T) {
	patches := gomonkey.NewPatches()
	defer patches.Reset()

	patches.ApplyFunc(checkCPU, func() error {
		return nil
	})
	patches.ApplyFunc(checkMem, func() error {
		return nil
	})
	patches.ApplyFunc(checkDisk, func() error {
		return nil
	})

	err := PreCheck([]string{"cpu", "mem", "disk"})
	assert.NoError(t, err)

	err = PreCheck([]string{"unsupport"})
	assert.Error(t, err)
}

func TestCheckCPU(t *testing.T) {
	t.Run("get cpu usage failed", func(t *testing.T) {
		patches := gomonkey.NewPatches()
		defer patches.Reset()

		patches.ApplyFunc(cpu.Percent, func(_interval time.Duration, _percpu bool) ([]float64, error) {
			return nil, errors.New("test error")
		})

		err := checkCPU()
		assert.ErrorContains(t, err, "failed to get cpu usage")
	})

	t.Run("unexpected cpu usage length", func(t *testing.T) {
		patches := gomonkey.NewPatches()
		defer patches.Reset()

		patches.ApplyFunc(cpu.Percent, func(_interval time.Duration, _percpu bool) ([]float64, error) {
			return []float64{}, nil
		})

		err := checkCPU()
		assert.EqualError(t, err, "unexpected cpu usage length 0")
	})

	t.Run("cpu usage too high", func(t *testing.T) {
		patches := gomonkey.NewPatches()
		defer patches.Reset()

		patches.ApplyFunc(cpu.Percent, func(_interval time.Duration, _percpu bool) ([]float64, error) {
			return []float64{100}, nil
		})

		err := checkCPU()
		errmsg := fmt.Sprintf("current cpu usage is %.2f, which exceeds the maximum allowed usage %.2f", 100.0, MaxCPUUsage)
		assert.EqualError(t, err, errmsg)
	})

	t.Run("cpu check successful", func(t *testing.T) {
		patches := gomonkey.NewPatches()
		defer patches.Reset()

		patches.ApplyFunc(cpu.Percent, func(_interval time.Duration, _percpu bool) ([]float64, error) {
			return []float64{20}, nil
		})
		err := checkCPU()
		assert.NoError(t, err)
	})
}

func TestCheckMem(t *testing.T) {
	t.Run("get mem usage failed", func(t *testing.T) {
		patches := gomonkey.NewPatches()
		defer patches.Reset()

		patches.ApplyFunc(mem.VirtualMemory, func() (*mem.VirtualMemoryStat, error) {
			return nil, errors.New("test error")
		})

		err := checkMem()
		assert.ErrorContains(t, err, "failed to get virtual memory stat")
	})

	t.Run("mem usage too high", func(t *testing.T) {
		patches := gomonkey.NewPatches()
		defer patches.Reset()

		patches.ApplyFunc(mem.VirtualMemory, func() (*mem.VirtualMemoryStat, error) {
			return &mem.VirtualMemoryStat{UsedPercent: 100}, nil
		})

		err := checkMem()
		errmsg := fmt.Sprintf("current mem usage is %.2f, which exceeds the maximum allowed usage %.2f", 100.0, MaxMemUsage)
		assert.EqualError(t, err, errmsg)
	})

	t.Run("mem check successful", func(t *testing.T) {
		patches := gomonkey.NewPatches()
		defer patches.Reset()

		patches.ApplyFunc(mem.VirtualMemory, func() (*mem.VirtualMemoryStat, error) {
			return &mem.VirtualMemoryStat{UsedPercent: 20}, nil
		})

		err := checkMem()
		assert.NoError(t, err)
	})
}

func TestCheckDisk(t *testing.T) {
	t.Run("get disk usage failed", func(t *testing.T) {
		patches := gomonkey.NewPatches()
		defer patches.Reset()

		patches.ApplyFunc(disk.Usage, func(_path string) (*disk.UsageStat, error) {
			return nil, errors.New("test error")
		})
		err := checkDisk()
		assert.ErrorContains(t, err, "failed to get disk usage")
	})

	t.Run("disk usage too high", func(t *testing.T) {
		patches := gomonkey.NewPatches()
		defer patches.Reset()

		patches.ApplyFunc(disk.Usage, func(_path string) (*disk.UsageStat, error) {
			return &disk.UsageStat{UsedPercent: 100}, nil
		})

		err := checkDisk()
		errmsg := fmt.Sprintf("current disk usage is %.2f, which exceeds the maximum allowed usage %.2f", 100.0, MaxDiskUsage)
		assert.EqualError(t, err, errmsg)
	})

	t.Run("disk check successful", func(t *testing.T) {
		patches := gomonkey.NewPatches()
		defer patches.Reset()
		patches.ApplyFunc(disk.Usage, func(_path string) (*disk.UsageStat, error) {
			return &disk.UsageStat{UsedPercent: 20}, nil
		})
		err := checkDisk()
		assert.NoError(t, err)
	})
}
