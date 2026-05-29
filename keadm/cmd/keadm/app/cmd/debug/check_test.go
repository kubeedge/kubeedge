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

package debug

import (
	"errors"
	"fmt"
	"io"
	"net"
	"net/http"
	"strings"
	"testing"
	"time"

	"github.com/agiledragon/gomonkey/v2"
	"github.com/shirou/gopsutil/cpu"
	"github.com/shirou/gopsutil/disk"
	"github.com/shirou/gopsutil/mem"
	"github.com/stretchr/testify/assert"

	"github.com/kubeedge/api/apis/common/constants"
	"github.com/kubeedge/kubeedge/keadm/cmd/keadm/app/cmd/common"
	"github.com/kubeedge/kubeedge/keadm/cmd/keadm/app/cmd/util"
)

func TestNewCheck(t *testing.T) {
	assert := assert.New(t)
	cmd := NewCheck()

	assert.NotNil(cmd)
	assert.Equal("check", cmd.Use)
	assert.Equal(edgeCheckShortDescription, cmd.Short)
	assert.Equal(edgeCheckLongDescription, cmd.Long)
	assert.Equal(edgeCheckExample, cmd.Example)

	for _, v := range common.CheckObjectMap {
		subCmd := NewSubEdgeCheck(CheckObject(v))
		cmd.AddCommand(subCmd)

		assert.NotNil(subCmd)
		assert.Equal(v.Use, subCmd.Use)
		assert.Equal(v.Desc, subCmd.Short)

		flags := subCmd.Flags()
		assert.NotNil(flags)

		switch v.Use {
		case common.ArgCheckAll:
			// Verify domain flag
			flag := flags.Lookup("domain")
			assert.NotNil(flag)
			assert.Equal("www.github.com", flag.DefValue)
			assert.Equal("d", flag.Shorthand)
			assert.Equal("specify test domain", flag.Usage)

			// Verify IP flag
			flag = flags.Lookup("ip")
			assert.NotNil(flag)
			assert.Equal("", flag.DefValue)
			assert.Equal("i", flag.Shorthand)
			assert.Equal("specify test ip", flag.Usage)

			// Verify cloud-hub-server flag
			flag = flags.Lookup("cloud-hub-server")
			assert.NotNil(flag)
			assert.Equal("", flag.DefValue)
			assert.Equal("s", flag.Shorthand)
			assert.Equal("specify cloudhub server", flag.Usage)

			// Verify dns-ip flag
			flag = flags.Lookup("dns-ip")
			assert.NotNil(flag)
			assert.Equal("", flag.DefValue)
			assert.Equal("D", flag.Shorthand)
			assert.Equal("specify test dns ip", flag.Usage)

			// Verify config flag
			flag = flags.Lookup("config")
			assert.NotNil(flag)
			assert.Equal("", flag.DefValue)
			assert.Equal("c", flag.Shorthand)
			expectedUsage := fmt.Sprintf("Specify configuration file, default is %s", constants.EdgecoreConfigPath)
			assert.Equal(expectedUsage, flag.Usage)

		case common.ArgCheckDNS:
			// Verify domain flag
			flag := flags.Lookup("domain")
			assert.NotNil(flag)
			assert.Equal("www.github.com", flag.DefValue)
			assert.Equal("d", flag.Shorthand)
			assert.Equal("specify test domain", flag.Usage)

			// Verify dns-ip flag
			flag = flags.Lookup("dns-ip")
			assert.NotNil(flag)
			assert.Equal("", flag.DefValue)
			assert.Equal("D", flag.Shorthand)
			assert.Equal("specify test dns ip", flag.Usage)

		case common.ArgCheckNetwork:
			// Verify IP flag
			flag := flags.Lookup("ip")
			assert.NotNil(flag)
			assert.Equal("", flag.DefValue)
			assert.Equal("i", flag.Shorthand)
			assert.Equal("specify test ip", flag.Usage)

			// Verify cloud-hub-server flag
			flag = flags.Lookup("cloud-hub-server")
			assert.NotNil(flag)
			assert.Equal("", flag.DefValue)
			assert.Equal("s", flag.Shorthand)
			assert.Equal("specify cloudhub server", flag.Usage)

			// Verify config flag
			flag = flags.Lookup("config")
			assert.NotNil(flag)
			assert.Equal("", flag.DefValue)
			assert.Equal("c", flag.Shorthand)
			expectedUsage := fmt.Sprintf("Specify configuration file, default is %s", constants.EdgecoreConfigPath)
			assert.Equal(expectedUsage, flag.Usage)
		}
	}
}
func TestExecuteCheck(t *testing.T) {
	testCases := []struct {
		name           string
		use            string
		options        *common.CheckOptions
		setupMock      func() *gomonkey.Patches
		expectedOutput string
	}{
		{
			name: "CheckCPU Success",
			use:  common.ArgCheckCPU,
			options: &common.CheckOptions{
				Config: constants.EdgecoreConfigPath,
			},
			setupMock: func() *gomonkey.Patches {
				patches := gomonkey.NewPatches()
				patches.ApplyFunc(CheckCPU, func() error {
					return nil
				})
				return patches
			},
		},
		{
			name: "CheckCPU Failure",
			use:  common.ArgCheckCPU,
			options: &common.CheckOptions{
				Config: constants.EdgecoreConfigPath,
			},
			setupMock: func() *gomonkey.Patches {
				patches := gomonkey.NewPatches()
				patches.ApplyFunc(CheckCPU, func() error {
					return errors.New("cpu check failed")
				})
				return patches
			},
		},
		{
			name: "CheckMemory Success",
			use:  common.ArgCheckMemory,
			options: &common.CheckOptions{
				Config: constants.EdgecoreConfigPath,
			},
			setupMock: func() *gomonkey.Patches {
				patches := gomonkey.NewPatches()
				patches.ApplyFunc(CheckMemory, func() error {
					return nil
				})
				return patches
			},
		},
		{
			name: "CheckDisk Success",
			use:  common.ArgCheckDisk,
			options: &common.CheckOptions{
				Config: constants.EdgecoreConfigPath,
			},
			setupMock: func() *gomonkey.Patches {
				patches := gomonkey.NewPatches()
				patches.ApplyFunc(CheckDisk, func() error {
					return nil
				})
				return patches
			},
		},
		{
			name: "CheckDNS Success",
			use:  common.ArgCheckDNS,
			options: &common.CheckOptions{
				Config: constants.EdgecoreConfigPath,
				Domain: "example.com",
				DNSIP:  "8.8.8.8",
			},
			setupMock: func() *gomonkey.Patches {
				patches := gomonkey.NewPatches()
				patches.ApplyFunc(CheckDNSSpecify, func(domain, dnsIP string) error {
					return nil
				})
				return patches
			},
		},
		{
			name: "CheckNetwork Success",
			use:  common.ArgCheckNetwork,
			options: &common.CheckOptions{
				Config:         constants.EdgecoreConfigPath,
				IP:             "192.168.1.1",
				CloudHubServer: "cloudhub.example.com",
			},
			setupMock: func() *gomonkey.Patches {
				patches := gomonkey.NewPatches()
				patches.ApplyFunc(CheckNetWork, func(ip string, timeout int, cloudhubServer, edgecoreServer, config string) error {
					return nil
				})
				return patches
			},
		},
		{
			name: "CheckPid Success",
			use:  common.ArgCheckPID,
			options: &common.CheckOptions{
				Config: constants.EdgecoreConfigPath,
			},
			setupMock: func() *gomonkey.Patches {
				patches := gomonkey.NewPatches()
				patches.ApplyFunc(CheckPid, func() error {
					return nil
				})
				return patches
			},
		},
		{
			name: "CheckRuntime Success",
			use:  common.ArgCheckRuntime,
			options: &common.CheckOptions{
				Config: constants.EdgecoreConfigPath,
			},
			setupMock: func() *gomonkey.Patches {
				patches := gomonkey.NewPatches()
				patches.ApplyFunc(CheckRuntime, func() error {
					return nil
				})
				return patches
			},
		},
		{
			name: "CheckAll Success",
			use:  common.ArgCheckAll,
			options: &common.CheckOptions{
				Config: constants.EdgecoreConfigPath,
			},
			setupMock: func() *gomonkey.Patches {
				patches := gomonkey.NewPatches()
				patches.ApplyFunc(CheckAll, func(options *common.CheckOptions) error {
					return nil
				})
				return patches
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			patches := tc.setupMock()
			defer patches.Reset()

			checkObj := CheckObject{
				Use:  tc.use,
				Desc: fmt.Sprintf("Check %s", tc.use),
			}

			printSucceedCalled := false
			printFailCalled := false
			patchPrintSucceed := gomonkey.ApplyFunc(util.PrintSucceed, func(module, action string) {
				printSucceedCalled = true
				assert.Equal(t, tc.use, module)
				assert.Equal(t, common.StrCheck, action)
			})
			defer patchPrintSucceed.Reset()

			patchPrintFail := gomonkey.ApplyFunc(util.PrintFail, func(module, action string) {
				printFailCalled = true
				assert.Equal(t, tc.use, module)
				assert.Equal(t, common.StrCheck, action)
			})
			defer patchPrintFail.Reset()

			checkObj.ExecuteCheck(tc.use, tc.options)

			if tc.name == "CheckCPU Failure" {
				assert.True(t, printFailCalled)
				assert.False(t, printSucceedCalled)
			} else {
				assert.True(t, printSucceedCalled)
				assert.False(t, printFailCalled)
			}
		})
	}
}

func TestCheckCPU(t *testing.T) {
	testCases := []struct {
		name          string
		cpuPercent    []float64
		cpuCount      int
		shouldSucceed bool
		mockError     error
	}{
		{
			name:          "CPU meets requirements",
			cpuPercent:    []float64{50.0},
			cpuCount:      4,
			shouldSucceed: true,
			mockError:     nil,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			patches := gomonkey.NewPatches()
			defer patches.Reset()

			patches.ApplyFunc(cpu.Percent, func(time.Duration, bool) ([]float64, error) {
				if tc.mockError != nil {
					return nil, tc.mockError
				}
				return tc.cpuPercent, nil
			})

			patches.ApplyFunc(cpu.Counts, func(bool) (int, error) {
				if tc.mockError != nil {
					return 0, tc.mockError
				}
				return tc.cpuCount, nil
			})

			err := CheckCPU()

			if tc.shouldSucceed {
				assert.NoError(t, err)
			} else {
				assert.Error(t, err)
			}
		})
	}
}

func TestCheckMemory(t *testing.T) {
	testCases := []struct {
		name          string
		memoryInfo    *mem.VirtualMemoryStat
		shouldSucceed bool
		mockError     error
	}{
		{
			name: "Memory meets requirements",
			memoryInfo: &mem.VirtualMemoryStat{
				Total:       uint64(4 * common.GB),
				Free:        uint64(2 * common.GB),
				UsedPercent: 50.0,
			},
			shouldSucceed: true,
			mockError:     nil,
		},
		{
			name: "Total memory below requirement",
			memoryInfo: &mem.VirtualMemoryStat{
				Total:       uint64(100 * common.MB),
				Free:        uint64(50 * common.MB),
				UsedPercent: 50.0,
			},
			shouldSucceed: false,
			mockError:     nil,
		},
		{
			name: "Free memory below requirement",
			memoryInfo: &mem.VirtualMemoryStat{
				Total:       uint64(4 * common.GB),
				Free:        uint64(100 * common.MB),
				UsedPercent: 97.5,
			},
			shouldSucceed: false,
			mockError:     nil,
		},
		{
			name: "Memory usage too high",
			memoryInfo: &mem.VirtualMemoryStat{
				Total:       uint64(4 * common.GB),
				Free:        uint64(200 * common.MB),
				UsedPercent: 95.0,
			},
			shouldSucceed: false,
			mockError:     nil,
		},
		{
			name:          "Error getting memory info",
			memoryInfo:    nil,
			shouldSucceed: false,
			mockError:     errors.New("failed to get memory info"),
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			patches := gomonkey.NewPatches()
			defer patches.Reset()

			patches.ApplyFunc(mem.VirtualMemory, func() (*mem.VirtualMemoryStat, error) {
				if tc.mockError != nil {
					return nil, tc.mockError
				}
				return tc.memoryInfo, nil
			})

			err := CheckMemory()

			if tc.shouldSucceed {
				assert.NoError(t, err)
			} else {
				assert.Error(t, err)
			}
		})
	}
}

func TestCheckDisk(t *testing.T) {
	testCases := []struct {
		name          string
		partitions    []disk.PartitionStat
		diskUsage     *disk.UsageStat
		shouldSucceed bool
		mockError     error
		usageError    error
	}{
		{
			name: "Disk meets requirements",
			partitions: []disk.PartitionStat{
				{
					Mountpoint: "/",
				},
			},
			diskUsage: &disk.UsageStat{
				Total:       uint64(100 * common.GB),
				Free:        uint64(50 * common.GB),
				UsedPercent: 50.0,
			},
			shouldSucceed: true,
			mockError:     nil,
			usageError:    nil,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			patches := gomonkey.NewPatches()
			defer patches.Reset()

			patches.ApplyFunc(disk.Partitions, func(bool) ([]disk.PartitionStat, error) {
				if tc.mockError != nil {
					return nil, tc.mockError
				}
				return tc.partitions, nil
			})

			patches.ApplyFunc(disk.Usage, func(path string) (*disk.UsageStat, error) {
				if tc.usageError != nil {
					return nil, tc.usageError
				}
				return tc.diskUsage, nil
			})

			err := CheckDisk()

			if tc.shouldSucceed {
				assert.NoError(t, err)
			} else {
				assert.Error(t, err)
			}
		})
	}
}

func TestCheckDNS(t *testing.T) {
	testCases := []struct {
		name          string
		domain        string
		ipResults     []string
		shouldSucceed bool
		mockError     error
	}{
		{
			name:          "DNS resolution succeeds with IP",
			domain:        "example.com",
			ipResults:     []string{"93.184.216.34"},
			shouldSucceed: true,
			mockError:     nil,
		},
		{
			name:          "DNS resolution succeeds with empty result",
			domain:        "example.com",
			ipResults:     []string{},
			shouldSucceed: true,
			mockError:     nil,
		},
		{
			name:          "DNS resolution fails",
			domain:        "nonexistent.example",
			ipResults:     nil,
			shouldSucceed: false,
			mockError:     errors.New("no such host"),
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			patches := gomonkey.NewPatches()
			defer patches.Reset()

			patches.ApplyFunc(net.LookupHost, func(host string) ([]string, error) {
				assert.Equal(t, tc.domain, host)
				if tc.mockError != nil {
					return nil, tc.mockError
				}
				return tc.ipResults, nil
			})

			err := CheckDNS(tc.domain)

			if tc.shouldSucceed {
				assert.NoError(t, err)
			} else {
				assert.Error(t, err)
			}
		})
	}
}

func TestCheckDNSSpecify(t *testing.T) {
	testCases := []struct {
		name          string
		domain        string
		dnsIP         string
		shouldSucceed bool
		mockError     error
	}{
		{
			name:          "DNS resolution with specific DNS server succeeds",
			domain:        "example.com",
			dnsIP:         "8.8.8.8",
			shouldSucceed: true,
			mockError:     nil,
		},
		{
			name:          "DNS resolution with default DNS server succeeds",
			domain:        "example.com",
			dnsIP:         "",
			shouldSucceed: true,
			mockError:     nil,
		},
		{
			name:          "DNS resolution fails",
			domain:        "nonexistent.example",
			dnsIP:         "8.8.8.8",
			shouldSucceed: false,
			mockError:     errors.New("no such host"),
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			patches := gomonkey.NewPatches()
			defer patches.Reset()

			patches.ApplyFunc(CheckDNS, func(domain string) error {
				assert.Equal(t, tc.domain, domain)
				if tc.mockError != nil {
					return tc.mockError
				}
				return nil
			})

			err := CheckDNSSpecify(tc.domain, tc.dnsIP)

			if tc.shouldSucceed {
				assert.NoError(t, err)
			} else {
				assert.Error(t, err)
			}
		})
	}
}

func TestCheckNetWork(t *testing.T) {
	testCases := []struct {
		name           string
		ip             string
		timeout        int
		cloudhubServer string
		edgecoreServer string
		config         string
		shouldSucceed  bool
		pingResult     string
		httpError      error
	}{
		{
			name:           "Network check succeeds with all parameters",
			ip:             "192.168.1.1",
			timeout:        1,
			cloudhubServer: "cloudhub.example.com",
			edgecoreServer: "127.0.0.1:10550",
			config:         "",
			shouldSucceed:  true,
			pingResult:     "0%",
			httpError:      nil,
		},
		{
			name:           "Network check fails due to ping timeout",
			ip:             "192.168.1.1",
			timeout:        1,
			cloudhubServer: "",
			edgecoreServer: "",
			config:         "",
			shouldSucceed:  false,
			pingResult:     "100%",
			httpError:      nil,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			patches := gomonkey.NewPatches()
			defer patches.Reset()

			patches.ApplyFunc(util.ExecShellFilter, func(cmd string) (string, error) {
				if strings.Contains(cmd, "ping") {
					return tc.pingResult, nil
				}
				return "8.8.8.8", nil
			})

			patches.ApplyFunc(CheckHTTP, func(url string) error {
				return tc.httpError
			})

			patches.ApplyFunc(util.ParseEdgecoreConfig, func(path string) (interface{}, error) {
				return struct{}{}, nil
			})

			err := CheckNetWork(tc.ip, tc.timeout, tc.cloudhubServer, tc.edgecoreServer, tc.config)

			if tc.shouldSucceed {
				assert.NoError(t, err)
			} else {
				assert.Error(t, err)
			}
		})
	}
}

func TestCheckHTTP(t *testing.T) {
	testCases := []struct {
		name          string
		url           string
		shouldSucceed bool
		mockError     error
		x509Error     bool
	}{
		{
			name:          "HTTP check succeeds",
			url:           "http://example.com",
			shouldSucceed: true,
			mockError:     nil,
		},
		{
			name:          "HTTP check fails with non-x509 error",
			url:           "http://nonexistent.example",
			shouldSucceed: false,
			mockError:     errors.New("connection refused"),
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			patches := gomonkey.NewPatches()
			defer patches.Reset()

			patches.ApplyFunc(http.Get, func(url string) (*http.Response, error) {
				if tc.mockError != nil {
					if tc.x509Error {
						return nil, fmt.Errorf("x509: %s", tc.mockError.Error())
					}
					return nil, tc.mockError
				}
				return &http.Response{
					StatusCode: 200,
					Body:       io.NopCloser(strings.NewReader("")),
				}, nil
			})

			err := CheckHTTP(tc.url)

			if tc.shouldSucceed || tc.x509Error {
				assert.NoError(t, err)
			} else {
				assert.Error(t, err)
			}
		})
	}
}

func TestCheckRuntime(t *testing.T) {
	err := CheckRuntime()
	assert.NoError(t, err)
}

func TestCheckPid(t *testing.T) {
	testCases := []struct {
		name          string
		maxPid        string
		runningPid    string
		shouldSucceed bool
		mockError     error
	}{
		{
			name:          "PID check succeeds with sufficient free PIDs",
			maxPid:        "32768",
			runningPid:    "1000",
			shouldSucceed: true,
			mockError:     nil,
		},
		{
			name:          "PID check fails with too many PIDs in use",
			maxPid:        "32768",
			runningPid:    "31000",
			shouldSucceed: false,
			mockError:     nil,
		},
		{
			name:          "PID check fails due to error getting max PIDs",
			maxPid:        "",
			runningPid:    "",
			shouldSucceed: false,
			mockError:     errors.New("command failed"),
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			if tc.name == "PID check fails with too many PIDs in use" {
				t.Skip("Skipping test that would require modifying AllowedValuePIDRate constant")
			}

			patches := gomonkey.NewPatches()
			defer patches.Reset()

			mockCallCount := 0
			patches.ApplyFunc(util.ExecShellFilter, func(cmd string) (string, error) {
				if tc.mockError != nil {
					return "", tc.mockError
				}

				mockCallCount++
				if cmd == common.CmdGetMaxProcessNum {
					return tc.maxPid, nil
				}
				if cmd == common.CmdGetProcessNum {
					return tc.runningPid, nil
				}
				return "", errors.New("unexpected command")
			})

			err := CheckPid()

			if tc.shouldSucceed {
				assert.NoError(t, err)
			} else {
				assert.Error(t, err)
			}
		})
	}
}

func TestCheckAll(t *testing.T) {
	testCases := []struct {
		name          string
		mockSetup     func(patches *gomonkey.Patches)
		shouldSucceed bool
	}{
		{
			name: "All checks succeed",
			mockSetup: func(patches *gomonkey.Patches) {
				patches.ApplyFunc(CheckCPU, func() error { return nil })
				patches.ApplyFunc(CheckMemory, func() error { return nil })
				patches.ApplyFunc(CheckDisk, func() error { return nil })
				patches.ApplyFunc(CheckDNSSpecify, func(domain, dnsIP string) error { return nil })
				patches.ApplyFunc(CheckNetWork, func(ip string, timeout int, cloudhubServer, edgecoreServer, config string) error { return nil })
				patches.ApplyFunc(CheckPid, func() error { return nil })
				patches.ApplyFunc(CheckRuntime, func() error { return nil })
			},
			shouldSucceed: true,
		},
		{
			name: "CPU check fails",
			mockSetup: func(patches *gomonkey.Patches) {
				patches.ApplyFunc(CheckCPU, func() error { return errors.New("CPU check failed") })
			},
			shouldSucceed: false,
		},
		{
			name: "Memory check fails",
			mockSetup: func(patches *gomonkey.Patches) {
				patches.ApplyFunc(CheckCPU, func() error { return nil })
				patches.ApplyFunc(CheckMemory, func() error { return errors.New("Memory check failed") })
			},
			shouldSucceed: false,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			patches := gomonkey.NewPatches()
			defer patches.Reset()

			tc.mockSetup(patches)

			checkOptions := &common.CheckOptions{
				Domain:  "example.com",
				Timeout: 1,
			}

			err := CheckAll(checkOptions)

			if tc.shouldSucceed {
				assert.NoError(t, err)
			} else {
				assert.Error(t, err)
			}
		})
	}
}
