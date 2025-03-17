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
	"reflect"
	"testing"

	"github.com/agiledragon/gomonkey/v2"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	v1 "k8s.io/api/core/v1"

	"github.com/kubeedge/api/apis/common/constants"
	cfgv1alpha2 "github.com/kubeedge/api/apis/componentconfig/edgecore/v1alpha2"
	"github.com/kubeedge/kubeedge/keadm/cmd/keadm/app/cmd/common"
	"github.com/kubeedge/kubeedge/keadm/cmd/keadm/app/cmd/util"
	"github.com/kubeedge/kubeedge/pkg/util/files"
)

func TestNewDiagnose(t *testing.T) {
	assert := assert.New(t)
	cmd := NewDiagnose()

	assert.NotNil(cmd)
	assert.Equal("diagnose", cmd.Use)
	assert.Equal(edgeDiagnoseShortDescription, cmd.Short)
	assert.Equal(edgeDiagnoseLongDescription, cmd.Long)
	assert.Equal(edgeDiagnoseExample, cmd.Example)

	subcommands := cmd.Commands()
	assert.NotNil(subcommands)
}

func TestNewSubDiagnose(t *testing.T) {
	assert := assert.New(t)

	cases := []struct {
		use               string
		expectedDefValue  map[string]string
		expectedShorthand map[string]string
		expectedUsage     map[string]string
	}{
		{
			use: common.ArgDiagnoseNode,
			expectedDefValue: map[string]string{
				common.EdgecoreConfig: constants.EdgecoreConfigPath,
			},
			expectedShorthand: map[string]string{
				common.EdgecoreConfig: "c",
			},
			expectedUsage: map[string]string{
				common.EdgecoreConfig: fmt.Sprintf("Specify configuration file, default is %s", constants.EdgecoreConfigPath),
			},
		},
		{
			use: common.ArgDiagnosePod,
			expectedDefValue: map[string]string{
				"namespace": "default",
			},
			expectedShorthand: map[string]string{
				"namespace": "n",
			},
			expectedUsage: map[string]string{
				"namespace": "specify namespace",
			},
		},
		{
			use: common.ArgDiagnoseInstall,
			expectedDefValue: map[string]string{
				"dns-ip":           "",
				"domain":           "",
				"ip":               "",
				"cloud-hub-server": "",
			},
			expectedShorthand: map[string]string{
				"dns-ip":           "D",
				"domain":           "d",
				"ip":               "i",
				"cloud-hub-server": "s",
			},
			expectedUsage: map[string]string{
				"dns-ip":           "specify test dns server ip",
				"domain":           "specify test domain",
				"ip":               "specify test ip",
				"cloud-hub-server": "specify cloudhub server",
			},
		},
	}

	for _, test := range cases {
		t.Run(test.use, func(t *testing.T) {
			diagnoseObj := Diagnose{
				Use:  test.use,
				Desc: fmt.Sprintf("Diagnose %s", test.use),
			}
			cmd := NewSubDiagnose(diagnoseObj)

			assert.NotNil(cmd)
			assert.Equal(diagnoseObj.Use, cmd.Use)
			assert.Equal(diagnoseObj.Desc, cmd.Short)

			flags := cmd.Flags()
			assert.NotNil(flags)

			for flagName, expectedDefValue := range test.expectedDefValue {
				t.Run(flagName, func(t *testing.T) {
					flag := flags.Lookup(flagName)
					assert.NotNil(flag)

					assert.Equal(expectedDefValue, flag.DefValue)
					assert.Equal(test.expectedShorthand[flagName], flag.Shorthand)
					assert.Equal(test.expectedUsage[flagName], flag.Usage)
				})
			}
		})
	}
}

func TestNewDiagnoseOptions(t *testing.T) {
	assert := assert.New(t)

	do := NewDiagnoseOptions()
	assert.NotNil(do)

	assert.Equal("default", do.Namespace)
	assert.Equal(constants.EdgecoreConfigPath, do.Config)
	assert.Equal("", do.CheckOptions.IP)
	assert.Equal(3, do.CheckOptions.Timeout)
}

func TestExecuteDiagnose(t *testing.T) {
	opts := &common.DiagnoseOptions{
		Config:    constants.EdgecoreConfigPath,
		Namespace: "default",
		CheckOptions: &common.CheckOptions{
			IP:      "1.1.1.1",
			Timeout: 3,
		},
	}

	t.Run("using the diagnose node", func(t *testing.T) {
		var mustCallPrintSuccessed bool

		patches := gomonkey.NewPatches()
		defer patches.Reset()

		patches.ApplyFunc(DiagnoseNode, func(_ops *common.DiagnoseOptions) error {
			return nil
		})
		patches.ApplyFunc(util.PrintSucceed, func(cmd, s string) {
			mustCallPrintSuccessed = true
			assert.Equal(t, common.ArgDiagnoseNode, cmd)
			assert.Equal(t, common.StrDiagnose, s)
		})

		var da Diagnose
		da.ExecuteDiagnose(common.ArgDiagnoseNode, opts, nil)
		assert.True(t, mustCallPrintSuccessed)
	})

	t.Run("using the diagnose node successful", func(t *testing.T) {
		var mustCallDiagnosePod, mustCallPrintSuccessed bool

		patches := gomonkey.NewPatches()
		defer patches.Reset()

		patches.ApplyFunc(DiagnoseNode, func(_ops *common.DiagnoseOptions) error {
			return nil
		})
		patches.ApplyFunc(DiagnosePod, func(_ops *common.DiagnoseOptions, _podName string) error {
			mustCallDiagnosePod = true
			return nil
		})
		patches.ApplyFunc(util.PrintSucceed, func(cmd, s string) {
			mustCallPrintSuccessed = true
			assert.Equal(t, common.ArgDiagnosePod, cmd)
			assert.Equal(t, common.StrDiagnose, s)
		})

		var da Diagnose
		da.ExecuteDiagnose(common.ArgDiagnosePod, opts, []string{"test-pod"})
		assert.True(t, mustCallPrintSuccessed)
		assert.True(t, mustCallDiagnosePod)
	})

	t.Run("using the diagnose node failed", func(t *testing.T) {
		var mustCallDiagnosePod, mustCallPrintFail bool

		patches := gomonkey.NewPatches()
		defer patches.Reset()

		patches.ApplyFunc(DiagnoseNode, func(_ops *common.DiagnoseOptions) error {
			return errors.New("test error")
		})
		patches.ApplyFunc(DiagnosePod, func(_ops *common.DiagnoseOptions, _podName string) error {
			mustCallDiagnosePod = true
			return nil
		})
		patches.ApplyFunc(util.PrintFail, func(cmd, s string) {
			mustCallPrintFail = true
			assert.Equal(t, common.ArgDiagnosePod, cmd)
			assert.Equal(t, common.StrDiagnose, s)
		})

		var da Diagnose
		da.ExecuteDiagnose(common.ArgDiagnosePod, opts, []string{"test-pod"})
		assert.True(t, mustCallPrintFail)
		assert.False(t, mustCallDiagnosePod)
	})

	t.Run("using the diagnose node", func(t *testing.T) {
		var mustCallPrintSuccessed bool

		patches := gomonkey.NewPatches()
		defer patches.Reset()

		patches.ApplyFunc(DiagnoseInstall, func(_ob *common.CheckOptions) error {
			return nil
		})
		patches.ApplyFunc(util.PrintSucceed, func(cmd, s string) {
			mustCallPrintSuccessed = true
			assert.Equal(t, common.ArgDiagnoseInstall, cmd)
			assert.Equal(t, common.StrDiagnose, s)
		})

		var da Diagnose
		da.ExecuteDiagnose(common.ArgDiagnoseInstall, opts, nil)
		assert.True(t, mustCallPrintSuccessed)
	})
}

func TestDiagnoseNode(t *testing.T) {
	globpatches := gomonkey.NewPatches()
	defer globpatches.Reset()

	globpatches.ApplyFunc(util.GetOSInterface, func() common.OSTypeInstaller {
		return &util.DebOS{}
	})
	globpatches.ApplyMethodFunc(reflect.TypeOf(&util.DebOS{}), "IsKubeEdgeProcessRunning",
		func(string) (bool, error) {
			return true, nil
		})
	globpatches.ApplyFunc(files.FileExists, func(path string) bool {
		switch path {
		case constants.EdgecoreConfigPath:
			return true
		case cfgv1alpha2.DataBaseDataSource:
			return true
		}
		return false
	})
	globpatches.ApplyFunc(util.ParseEdgecoreConfig, func(_edgecorePath string) (*cfgv1alpha2.EdgeCoreConfig, error) {
		return cfgv1alpha2.NewDefaultEdgeCoreConfig(), nil
	})
	globpatches.ApplyFunc(CheckHTTP, func(_url string) error {
		return nil
	})

	opts := &common.DiagnoseOptions{
		Config: constants.EdgecoreConfigPath,
	}

	t.Run("get edgecore status fail", func(t *testing.T) {
		patches := gomonkey.NewPatches()
		defer patches.Reset()

		patches.ApplyMethodFunc(reflect.TypeOf(&util.DebOS{}), "IsKubeEdgeProcessRunning",
			func(string) (bool, error) {
				return false, errors.New("test error")
			})

		err := DiagnoseNode(opts)
		require.ErrorContains(t, err, "get edgecore status fail")
	})

	t.Run("edgecore is not running", func(t *testing.T) {
		patches := gomonkey.NewPatches()
		defer patches.Reset()

		patches.ApplyMethodFunc(reflect.TypeOf(&util.DebOS{}), "IsKubeEdgeProcessRunning",
			func(string) (bool, error) {
				return false, nil
			})
		err := DiagnoseNode(opts)
		require.ErrorContains(t, err, "edgecore is not running")
	})

	t.Run("edge config is not exists", func(t *testing.T) {
		err := DiagnoseNode(&common.DiagnoseOptions{
			Config: "config/edgecore.yaml",
		})
		require.ErrorContains(t, err, "edge config is not exists")
	})

	t.Run("parse edgecore config failed", func(t *testing.T) {
		patches := gomonkey.NewPatches()
		defer patches.Reset()

		patches.ApplyFunc(util.ParseEdgecoreConfig, func(_edgecorePath string) (*cfgv1alpha2.EdgeCoreConfig, error) {
			return nil, errors.New("test error")
		})
		err := DiagnoseNode(opts)
		require.ErrorContains(t, err, "parse edgecore config failed")
	})

	t.Run("dataSource is not exists", func(t *testing.T) {
		patches := gomonkey.NewPatches()
		defer patches.Reset()

		patches.ApplyFunc(util.ParseEdgecoreConfig, func(_edgecorePath string) (*cfgv1alpha2.EdgeCoreConfig, error) {
			cfg := cfgv1alpha2.NewDefaultEdgeCoreConfig()
			cfg.DataBase.DataSource = "database/edgecore.db"
			return cfg, nil
		})

		err := DiagnoseNode(opts)
		require.ErrorContains(t, err, "dataSource is not exists")
	})

	t.Run("edgehub is not enable", func(t *testing.T) {
		patches := gomonkey.NewPatches()
		defer patches.Reset()

		patches.ApplyFunc(util.ParseEdgecoreConfig, func(_edgecorePath string) (*cfgv1alpha2.EdgeCoreConfig, error) {
			cfg := cfgv1alpha2.NewDefaultEdgeCoreConfig()
			cfg.Modules.EdgeHub.WebSocket.Enable = false
			return cfg, nil
		})

		err := DiagnoseNode(opts)
		require.ErrorContains(t, err, "edgehub is not enable")
	})

	t.Run("cloudcore websocket connection failed", func(t *testing.T) {
		patches := gomonkey.NewPatches()
		defer patches.Reset()

		patches.ApplyFunc(CheckHTTP, func(_url string) error {
			return errors.New("test error")
		})

		err := DiagnoseNode(opts)
		require.ErrorContains(t, err, "cloudcore websocket connection failed")
	})

	t.Run("diagnose node successful", func(t *testing.T) {
		err := DiagnoseNode(opts)
		require.NoError(t, err)
	})
}

func TestDiagnosePod(t *testing.T) {
	globpatches := gomonkey.NewPatches()
	defer globpatches.Reset()

	globpatches.ApplyFunc(InitDB, func(_driverName, _dbName, _dataSource string) error {
		return nil
	})

	ops := &common.DiagnoseOptions{
		Namespace: "default",
		DBPath:    "/var/lib/kubeedge/edgecore.db",
	}

	t.Run("failed to initialize database", func(t *testing.T) {
		patches := gomonkey.NewPatches()
		defer patches.Reset()

		patches.ApplyFunc(InitDB, func(_driverName, _dbName, _dataSource string) error {
			return errors.New("test error")
		})

		err := DiagnosePod(ops, "test-pod")
		require.ErrorContains(t, err, "failed to initialize database")
	})

	t.Run("pod status query failed", func(t *testing.T) {
		patches := gomonkey.NewPatches()
		defer patches.Reset()

		patches.ApplyFunc(QueryPodFromDatabase, func(_namespace, _podName string) (*v1.PodStatus, error) {
			return nil, errors.New("pod status query failed")
		})

		err := DiagnosePod(ops, "test-pod")
		require.ErrorContains(t, err, "pod status query failed")
	})

	t.Run("pod status not ready 1", func(t *testing.T) {
		cases := []v1.PodStatus{
			{
				Phase: "Pending",
				Conditions: []v1.PodCondition{
					{
						Type:    "Ready",
						Status:  "False",
						Reason:  "ContainersNotReady",
						Message: "containers with unready status",
					},
				},
				ContainerStatuses: []v1.ContainerStatus{
					{
						Name:  "container1",
						Ready: false,
						State: v1.ContainerState{
							Waiting: &v1.ContainerStateWaiting{
								Reason:  "ImagePullBackOff",
								Message: "Back-off pulling image",
							},
						},
						RestartCount: 2,
					},
				},
			},
			{
				Phase: "Failed",
				Conditions: []v1.PodCondition{
					{
						Type:   "Ready",
						Status: "False",
					},
				},
				ContainerStatuses: []v1.ContainerStatus{
					{
						Name:  "container1",
						Ready: false,
						State: v1.ContainerState{
							Terminated: &v1.ContainerStateTerminated{
								Reason:   "Error",
								Message:  "Container terminated",
								ExitCode: 1,
							},
						},
						RestartCount: 3,
					},
				},
			},
			{
				Phase: "Running",
				Conditions: []v1.PodCondition{
					{
						Type:   "Ready",
						Status: "False",
					},
				},
				ContainerStatuses: []v1.ContainerStatus{
					{
						Name:  "container1",
						Ready: false,
						State: v1.ContainerState{
							Terminated: &v1.ContainerStateTerminated{
								Reason:   "Error",
								Message:  "Container terminated",
								ExitCode: 1,
							},
						},
						RestartCount: 3,
					},
				},
			},
		}
		for i := range cases {
			t.Run(fmt.Sprintf("#%d", i+1), func(t *testing.T) {
				patches := gomonkey.NewPatches()
				defer patches.Reset()

				patches.ApplyFunc(QueryPodFromDatabase, func(_namespace, _podName string) (*v1.PodStatus, error) {
					return &cases[i], nil
				})

				err := DiagnosePod(ops, "test-pod")
				require.ErrorContains(t, err, "pod test-pod is not Ready")
			})
		}
	})

	t.Run("diagnose pod successful", func(t *testing.T) {
		patches := gomonkey.NewPatches()
		defer patches.Reset()

		patches.ApplyFunc(QueryPodFromDatabase, func(_namespace, _podName string) (*v1.PodStatus, error) {
			return &v1.PodStatus{
				Phase: "Running",
				Conditions: []v1.PodCondition{
					{
						Type:   "Ready",
						Status: "True",
					},
				},
				ContainerStatuses: []v1.ContainerStatus{
					{
						Name:  "container1",
						Ready: true,
						State: v1.ContainerState{},
					},
				},
			}, nil
		})

		err := DiagnosePod(ops, "test-pod")
		require.NoError(t, err)
	})
}

func TestDiagnoseInstall(t *testing.T) {
	patches := gomonkey.NewPatches()
	defer patches.Reset()

	const (
		cpuError     = "cpu check failed"
		memoryError  = "memory check failed"
		diskError    = "disk check failed"
		dnsError     = "dns specify check failed"
		networkError = "network check failed"
		pidError     = "pid check failed"
	)

	funcsFake := &struct {
		checkCPUError     bool
		checkMemoryError  bool
		checkDiskError    bool
		checkDNSError     bool
		checkNetWorkError bool
		checkPidError     bool
	}{}

	patches.ApplyFunc(CheckCPU, func() error {
		if funcsFake.checkCPUError {
			return errors.New(cpuError)
		}
		return nil
	})
	patches.ApplyFunc(CheckMemory, func() error {
		if funcsFake.checkMemoryError {
			return errors.New(memoryError)
		}
		return nil
	})
	patches.ApplyFunc(CheckDisk, func() error {
		if funcsFake.checkDiskError {
			return errors.New(diskError)
		}
		return nil
	})
	patches.ApplyFunc(CheckDNSSpecify, func(_domain, _dnsIP string) error {
		if funcsFake.checkDNSError {
			return errors.New(dnsError)
		}
		return nil
	})
	patches.ApplyFunc(CheckNetWork, func(_ip string, _timeout int, _cloudHub, _edgeCore, _config string) error {
		if funcsFake.checkNetWorkError {
			return errors.New(networkError)
		}
		return nil
	})
	patches.ApplyFunc(CheckPid, func() error {
		if funcsFake.checkPidError {
			return errors.New(pidError)
		}
		return nil
	})

	opts := &common.CheckOptions{
		IP:      "127.0.0.1",
		Timeout: 3,
		Domain:  "example.com",
		DNSIP:   "8.8.8.8",
	}

	t.Run(cpuError, func(t *testing.T) {
		funcsFake.checkCPUError = true
		defer func() {
			funcsFake.checkCPUError = false
		}()
		err := DiagnoseInstall(opts)
		require.ErrorContains(t, err, cpuError)
	})

	t.Run(memoryError, func(t *testing.T) {
		funcsFake.checkMemoryError = true
		defer func() {
			funcsFake.checkMemoryError = false
		}()

		err := DiagnoseInstall(opts)
		require.ErrorContains(t, err, memoryError)
	})

	t.Run(diskError, func(t *testing.T) {
		funcsFake.checkDiskError = true
		defer func() {
			funcsFake.checkDiskError = false
		}()

		err := DiagnoseInstall(opts)
		require.ErrorContains(t, err, diskError)
	})

	t.Run(dnsError, func(t *testing.T) {
		funcsFake.checkDNSError = true
		defer func() {
			funcsFake.checkDNSError = false
		}()

		err := DiagnoseInstall(opts)
		require.ErrorContains(t, err, dnsError)
	})

	t.Run(networkError, func(t *testing.T) {
		funcsFake.checkNetWorkError = true
		defer func() {
			funcsFake.checkNetWorkError = false
		}()

		err := DiagnoseInstall(opts)
		require.ErrorContains(t, err, networkError)
	})

	t.Run(pidError, func(t *testing.T) {
		funcsFake.checkPidError = true
		defer func() {
			funcsFake.checkPidError = false
		}()

		err := DiagnoseInstall(opts)
		require.ErrorContains(t, err, pidError)
	})

	t.Run("diagnose install successful", func(t *testing.T) {
		err := DiagnoseInstall(opts)
		require.NoError(t, err)
	})
}
