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
	"testing"

	"github.com/agiledragon/gomonkey/v2"
	"github.com/stretchr/testify/assert"
	v1 "k8s.io/api/core/v1"

	"github.com/kubeedge/kubeedge/keadm/cmd/keadm/app/cmd/common"
	"github.com/kubeedge/kubeedge/keadm/cmd/keadm/app/cmd/util"
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
				common.EdgecoreConfig: common.EdgecoreConfigPath,
			},
			expectedShorthand: map[string]string{
				common.EdgecoreConfig: "c",
			},
			expectedUsage: map[string]string{
				common.EdgecoreConfig: fmt.Sprintf("Specify configuration file, default is %s", common.EdgecoreConfigPath),
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
	assert.Equal(common.EdgecoreConfigPath, do.Config)
	assert.Equal("", do.CheckOptions.IP)
	assert.Equal(3, do.CheckOptions.Timeout)
}

func TestExecuteDiagnose(t *testing.T) {
	assert := assert.New(t)

	do := &common.DiagnoseOptions{
		Config:    common.EdgecoreConfigPath,
		Namespace: "default",
		CheckOptions: &common.CheckOptions{
			IP:      "",
			Timeout: 3,
		},
	}

	diagObj := Diagnose{
		Use:  common.ArgDiagnoseNode,
		Desc: "Diagnose node",
	}

	patchDiagnoseNode := gomonkey.ApplyFunc(DiagnoseNode, func(ops *common.DiagnoseOptions) error {
		return nil
	})
	defer patchDiagnoseNode.Reset()

	patchPrintSucceed := gomonkey.ApplyFunc(util.PrintSucceed, func(objectType, action string) {})
	defer patchPrintSucceed.Reset()

	diagObj.ExecuteDiagnose(common.ArgDiagnoseNode, do, []string{})

	patchDiagnoseNode.Reset()
	patchDiagnoseNodeFail := gomonkey.ApplyFunc(DiagnoseNode, func(ops *common.DiagnoseOptions) error {
		return errors.New("node diagnosis failed")
	})
	defer patchDiagnoseNodeFail.Reset()

	patchPrintFail := gomonkey.ApplyFunc(util.PrintFail, func(objectType, action string) {})
	defer patchPrintFail.Reset()

	diagObj.ExecuteDiagnose(common.ArgDiagnoseNode, do, []string{})

	patchDiagnoseNodeFail.Reset()
	patchDiagnoseNode = gomonkey.ApplyFunc(DiagnoseNode, func(ops *common.DiagnoseOptions) error {
		return nil
	})
	defer patchDiagnoseNode.Reset()

	patchDiagnosePod := gomonkey.ApplyFunc(DiagnosePod, func(ops *common.DiagnoseOptions, podName string) error {
		assert.Equal("test-pod", podName)
		return nil
	})
	defer patchDiagnosePod.Reset()

	diagObj.Use = common.ArgDiagnosePod
	diagObj.ExecuteDiagnose(common.ArgDiagnosePod, do, []string{"test-pod"})

	diagObj.ExecuteDiagnose(common.ArgDiagnosePod, do, []string{})

	diagObj.Use = common.ArgDiagnoseInstall

	patchDiagnoseInstall := gomonkey.ApplyFunc(DiagnoseInstall, func(ob *common.CheckOptions) error {
		return nil
	})
	defer patchDiagnoseInstall.Reset()

	diagObj.ExecuteDiagnose(common.ArgDiagnoseInstall, do, []string{})

	patchDiagnoseInstall.Reset()
	patchDiagnoseInstallFail := gomonkey.ApplyFunc(DiagnoseInstall, func(ob *common.CheckOptions) error {
		return errors.New("install diagnosis failed")
	})
	defer patchDiagnoseInstallFail.Reset()

	diagObj.ExecuteDiagnose(common.ArgDiagnoseInstall, do, []string{})
}

func TestDiagnosePod(t *testing.T) {
	assert := assert.New(t)

	ops := &common.DiagnoseOptions{
		Namespace: "default",
		DBPath:    "/var/lib/kubeedge/edgecore.db",
	}

	podStatus := &v1.PodStatus{
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
			},
		},
	}

	patchInitDB := gomonkey.ApplyFunc(InitDB, func(driverName, aliasName, dbPath string) error {
		return nil
	})
	defer patchInitDB.Reset()

	patchQueryPod := gomonkey.ApplyFunc(QueryPodFromDatabase, func(namespace, podName string) (*v1.PodStatus, error) {
		return podStatus, nil
	})
	defer patchQueryPod.Reset()

	err := DiagnosePod(ops, "test-pod")
	assert.NoError(err)

	patchInitDB.Reset()
	patchInitDB = gomonkey.ApplyFunc(InitDB, func(driverName, aliasName, dbPath string) error {
		return errors.New("db init failed")
	})

	err = DiagnosePod(ops, "test-pod")
	assert.Error(err)
	assert.Contains(err.Error(), "failed to initialize database")

	patchInitDB.Reset()
	patchInitDB = gomonkey.ApplyFunc(InitDB, func(driverName, aliasName, dbPath string) error {
		return nil
	})

	patchQueryPod.Reset()
	patchQueryPod = gomonkey.ApplyFunc(QueryPodFromDatabase, func(namespace, podName string) (*v1.PodStatus, error) {
		return nil, errors.New("pod query failed")
	})

	err = DiagnosePod(ops, "test-pod")
	assert.Error(err)
	assert.Contains(err.Error(), "pod query failed")

	patchQueryPod.Reset()
	notReadyStatus := &v1.PodStatus{
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
	}
	patchQueryPod = gomonkey.ApplyFunc(QueryPodFromDatabase, func(namespace, podName string) (*v1.PodStatus, error) {
		return notReadyStatus, nil
	})

	err = DiagnosePod(ops, "test-pod")
	assert.Error(err)
	assert.Contains(err.Error(), "not Ready")

	patchQueryPod.Reset()
	terminatedStatus := &v1.PodStatus{
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
	}
	patchQueryPod = gomonkey.ApplyFunc(QueryPodFromDatabase, func(namespace, podName string) (*v1.PodStatus, error) {
		return terminatedStatus, nil
	})

	err = DiagnosePod(ops, "test-pod")
	assert.Error(err)
	assert.Contains(err.Error(), "not Ready")

	patchQueryPod.Reset()
	genericNotReadyStatus := &v1.PodStatus{
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
				State: v1.ContainerState{},
			},
		},
	}
	patchQueryPod = gomonkey.ApplyFunc(QueryPodFromDatabase, func(namespace, podName string) (*v1.PodStatus, error) {
		return genericNotReadyStatus, nil
	})

	err = DiagnosePod(ops, "test-pod")
	assert.Error(err)
	assert.Contains(err.Error(), "not Ready")
}

func TestDiagnoseInstall(t *testing.T) {
	assert := assert.New(t)

	opts := &common.CheckOptions{
		IP:      "127.0.0.1",
		Timeout: 3,
		Domain:  "example.com",
		DNSIP:   "8.8.8.8",
	}

	patchCheckCPU := gomonkey.ApplyFunc(CheckCPU, func() error {
		return errors.New("cpu check failed")
	})
	defer patchCheckCPU.Reset()

	err := DiagnoseInstall(opts)
	assert.Error(err)
	assert.Contains(err.Error(), "cpu check failed")

	patchCheckCPU.Reset()
	patchCheckCPU = gomonkey.ApplyFunc(CheckCPU, func() error {
		return nil
	})

	patchCheckMemory := gomonkey.ApplyFunc(CheckMemory, func() error {
		return errors.New("memory check failed")
	})
	defer patchCheckMemory.Reset()

	err = DiagnoseInstall(opts)
	assert.Error(err)
	assert.Contains(err.Error(), "memory check failed")

	patchCheckMemory.Reset()
	patchCheckMemory = gomonkey.ApplyFunc(CheckMemory, func() error {
		return nil
	})

	patchCheckDisk := gomonkey.ApplyFunc(CheckDisk, func() error {
		return errors.New("disk check failed")
	})
	defer patchCheckDisk.Reset()

	err = DiagnoseInstall(opts)
	assert.Error(err)
	assert.Contains(err.Error(), "disk check failed")

	patchCheckDisk.Reset()
	patchCheckDisk = gomonkey.ApplyFunc(CheckDisk, func() error {
		return nil
	})

	patchCheckDNS := gomonkey.ApplyFunc(CheckDNSSpecify, func(domain, dnsIP string) error {
		return errors.New("dns check failed")
	})
	defer patchCheckDNS.Reset()

	err = DiagnoseInstall(opts)
	assert.Error(err)
	assert.Contains(err.Error(), "dns check failed")

	patchCheckDNS.Reset()
	patchCheckDNS = gomonkey.ApplyFunc(CheckDNSSpecify, func(domain, dnsIP string) error {
		return nil
	})

	patchCheckNetwork := gomonkey.ApplyFunc(CheckNetWork, func(ip string, timeout int, cloudHub, edgeCore, config string) error {
		return errors.New("network check failed")
	})
	defer patchCheckNetwork.Reset()

	err = DiagnoseInstall(opts)
	assert.Error(err)
	assert.Contains(err.Error(), "network check failed")

	patchCheckNetwork.Reset()
	patchCheckNetwork = gomonkey.ApplyFunc(CheckNetWork, func(ip string, timeout int, cloudHub, edgeCore, config string) error {
		return nil
	})

	patchCheckPid := gomonkey.ApplyFunc(CheckPid, func() error {
		return errors.New("pid check failed")
	})
	defer patchCheckPid.Reset()

	err = DiagnoseInstall(opts)
	assert.Error(err)
	assert.Contains(err.Error(), "pid check failed")

	patchCheckPid.Reset()
	patchCheckPid = gomonkey.ApplyFunc(CheckPid, func() error {
		return nil
	})

	err = DiagnoseInstall(opts)
	assert.NoError(err)

	opts.Domain = ""
	err = DiagnoseInstall(opts)
	assert.NoError(err)
}

func TestDiagnoseNode(t *testing.T) {
	assert := assert.New(t)

	ops := &common.DiagnoseOptions{
		Config: common.EdgecoreConfigPath,
	}

	err := DiagnoseNode(ops)

	assert.Error(err)
}
