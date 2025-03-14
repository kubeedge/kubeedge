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
	"bytes"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/agiledragon/gomonkey/v2"
	"github.com/stretchr/testify/assert"

	"github.com/kubeedge/api/apis/componentconfig/edgecore/v1alpha2"
	"github.com/kubeedge/kubeedge/keadm/cmd/keadm/app/cmd/common"
	"github.com/kubeedge/kubeedge/keadm/cmd/keadm/app/cmd/util"
	"github.com/kubeedge/kubeedge/pkg/util/execs"
	"github.com/kubeedge/kubeedge/pkg/util/files"
)

func setupMkdirPatch(t *testing.T, path string, shouldSucceed bool) *gomonkey.Patches {
	assert := assert.New(t)
	if shouldSucceed {
		return gomonkey.ApplyFunc(os.Mkdir, func(p string, perm os.FileMode) error {
			assert.Equal(path, p)
			assert.Equal(os.ModePerm, perm)
			return nil
		})
	}
	return gomonkey.ApplyFunc(os.Mkdir, func(p string, perm os.FileMode) error {
		return errors.New("directory creation failed")
	})
}

func setupExecShellPatch(shouldSucceed bool) *gomonkey.Patches {
	if shouldSucceed {
		return gomonkey.ApplyFunc(ExecuteShell, func(cmdStr string, tmpPath string) error {
			return nil
		})
	}
	return gomonkey.ApplyFunc(ExecuteShell, func(cmdStr string, tmpPath string) error {
		return errors.New("command execution failed")
	})
}

func setupCopyFilePatch(shouldSucceed bool) *gomonkey.Patches {
	if shouldSucceed {
		return gomonkey.ApplyFunc(CopyFile, func(pathSrc, tmpPath string) error {
			return nil
		})
	}
	return gomonkey.ApplyFunc(CopyFile, func(pathSrc, tmpPath string) error {
		return errors.New("file copy failed")
	})
}

func TestPrintDetail(t *testing.T) {
	assert := assert.New(t)
	oldStdout := os.Stdout
	r, w, err := os.Pipe()
	assert.NoError(err)
	os.Stdout = w

	printDeatilFlag = false
	printDetail("This should not be printed")

	printDeatilFlag = true
	printDetail("This should be printed")

	w.Close()
	os.Stdout = oldStdout

	var buf bytes.Buffer
	_, err = buf.ReadFrom(r)
	assert.NoError(err)

	assert.NotContains(buf.String(), "This should not be printed")
	assert.Contains(buf.String(), "This should be printed")
}

func TestExecuteShell(t *testing.T) {
	assert := assert.New(t)

	mockCmd := &execs.Command{}

	cmdPatch := gomonkey.ApplyFunc(execs.NewCommand, func(command string) *execs.Command {
		return mockCmd
	})
	defer cmdPatch.Reset()

	execPatch := gomonkey.ApplyMethod((*execs.Command)(nil), "Exec", func(_ *execs.Command) error {
		return nil
	})
	defer execPatch.Reset()

	err := ExecuteShell("ls -la", "/tmp")
	assert.NoError(err)

	execPatch.Reset()
	execErrorPatch := gomonkey.ApplyMethod((*execs.Command)(nil), "Exec", func(_ *execs.Command) error {
		return errors.New("command execution failed")
	})
	defer execErrorPatch.Reset()

	err = ExecuteShell("ls -la", "/tmp")
	assert.Error(err)
	assert.Equal("command execution failed", err.Error())
}

func TestCopyFile(t *testing.T) {
	assert := assert.New(t)

	mockCmd := &execs.Command{}

	cmdPatch := gomonkey.ApplyFunc(execs.NewCommand, func(command string) *execs.Command {
		return mockCmd
	})
	defer cmdPatch.Reset()

	execPatch := gomonkey.ApplyMethod((*execs.Command)(nil), "Exec", func(_ *execs.Command) error {
		return nil
	})
	defer execPatch.Reset()

	err := CopyFile("/source/path", "/dest/path")
	assert.NoError(err)

	execPatch.Reset()
	execErrorPatch := gomonkey.ApplyMethod((*execs.Command)(nil), "Exec", func(_ *execs.Command) error {
		return errors.New("file copy failed")
	})
	defer execErrorPatch.Reset()

	err = CopyFile("/source/path", "/dest/path")
	assert.Error(err)
	assert.Equal("file copy failed", err.Error())
}

func TestMakeDirTmp(t *testing.T) {
	assert := assert.New(t)
	fixedTime := time.Date(2024, 1, 1, 12, 0, 0, 0, time.UTC)
	const expectedTimeStr = "2024_0101_120000"
	expectedTmpName := fmt.Sprintf("/tmp/edge_%s", expectedTimeStr)

	timePatch := gomonkey.ApplyFunc(time.Now, func() time.Time {
		return fixedTime
	})
	defer timePatch.Reset()

	mkdirPatch := gomonkey.ApplyFunc(os.Mkdir, func(path string, perm os.FileMode) error {
		assert.Equal(expectedTmpName, path)
		assert.Equal(os.ModePerm, perm)
		return nil
	})
	defer mkdirPatch.Reset()

	tmpName, timeStr, err := makeDirTmp()
	assert.NoError(err)
	assert.Equal(expectedTmpName, tmpName)
	assert.Equal(expectedTimeStr, timeStr)

	mkdirPatch.Reset()
	mkdirErrorPatch := gomonkey.ApplyFunc(os.Mkdir, func(path string, perm os.FileMode) error {
		return errors.New("directory creation failed")
	})
	defer mkdirErrorPatch.Reset()

	_, _, err = makeDirTmp()
	assert.Error(err)
	assert.Equal("directory creation failed", err.Error())
}

func TestVerificationParameters(t *testing.T) {
	assert := assert.New(t)
	opts := &common.CollectOptions{
		Config:     "/path/to/config",
		OutputPath: "/path/to/output",
		Detail:     false,
	}

	fileExistsPatch := gomonkey.ApplyFunc(files.FileExists, func(path string) bool {
		return path != opts.Config
	})
	defer fileExistsPatch.Reset()

	err := VerificationParameters(opts)
	assert.Error(err)
	assert.Contains(err.Error(), "edgecore config")

	fileExistsPatch.Reset()
	fileExistsConfigPatch := gomonkey.ApplyFunc(files.FileExists, func(path string) bool {
		return path == opts.Config
	})
	defer fileExistsConfigPatch.Reset()

	absPatch := gomonkey.ApplyFunc(filepath.Abs, func(path string) (string, error) {
		return path, nil
	})
	defer absPatch.Reset()

	err = VerificationParameters(opts)
	assert.Error(err)
	assert.Contains(err.Error(), "output-path")

	fileExistsConfigPatch.Reset()
	fileExistsAllPatch := gomonkey.ApplyFunc(files.FileExists, func(path string) bool {
		return true
	})
	defer fileExistsAllPatch.Reset()

	err = VerificationParameters(opts)
	assert.NoError(err)
	assert.Equal(opts.OutputPath, opts.OutputPath)

	opts.Detail = true
	err = VerificationParameters(opts)
	assert.NoError(err)
	assert.True(printDeatilFlag)

	printDeatilFlag = false
}

func TestCollectSystemData(t *testing.T) {
	assert := assert.New(t)

	mkdirPatch := setupMkdirPatch(t, "/tmp/system", true)
	defer mkdirPatch.Reset()

	execShellPatch := setupExecShellPatch(true)
	defer execShellPatch.Reset()

	copyFilePatch := setupCopyFilePatch(true)
	defer copyFilePatch.Reset()

	err := collectSystemData("/tmp/system")
	assert.NoError(err)

	mkdirPatch.Reset()
	mkdirErrorPatch := setupMkdirPatch(t, "/tmp/system", false)
	defer mkdirErrorPatch.Reset()

	err = collectSystemData("/tmp/system")
	assert.Error(err)
	assert.Equal("directory creation failed", err.Error())

	mkdirErrorPatch.Reset()
	mkdirSuccessPatch := setupMkdirPatch(t, "/tmp/system", true)
	defer mkdirSuccessPatch.Reset()

	execShellPatch.Reset()
	execShellErrorPatch := gomonkey.ApplyFunc(ExecuteShell, func(cmdStr string, tmpPath string) error {
		if cmdStr == common.CmdArchInfo {
			return errors.New("command execution failed")
		}
		return nil
	})
	defer execShellErrorPatch.Reset()

	err = collectSystemData("/tmp/system")
	assert.Error(err)
	assert.Equal("command execution failed", err.Error())

	execShellErrorPatch.Reset()
	execShellSuccessPatch := setupExecShellPatch(true)
	defer execShellSuccessPatch.Reset()

	copyFilePatch.Reset()
	copyFileErrorPatch := gomonkey.ApplyFunc(CopyFile, func(pathSrc, tmpPath string) error {
		if pathSrc == common.PathCpuinfo {
			return errors.New("file copy failed")
		}
		return nil
	})
	defer copyFileErrorPatch.Reset()

	err = collectSystemData("/tmp/system")
	assert.Error(err)
	assert.Equal("file copy failed", err.Error())
}

func TestCollectEdgecoreData(t *testing.T) {
	assert := assert.New(t)

	db := &v1alpha2.DataBase{
		DataSource: "/path/to/datasource",
	}

	eh := &v1alpha2.EdgeHub{
		TLSCertFile:       "/path/to/cert",
		TLSPrivateKeyFile: "/path/to/key",
		TLSCAFile:         "/path/to/ca",
	}

	modules := &v1alpha2.Modules{
		EdgeHub: eh,
	}

	config := &v1alpha2.EdgeCoreConfig{}
	config.DataBase = db
	config.Modules = modules

	opts := &common.CollectOptions{
		LogPath: "/path/to/log",
	}

	mkdirPatch := setupMkdirPatch(t, "/tmp/edgecore", true)
	defer mkdirPatch.Reset()

	copyFilePatch := setupCopyFilePatch(true)
	defer copyFilePatch.Reset()

	execShellPatch := setupExecShellPatch(true)
	defer execShellPatch.Reset()

	err := collectEdgecoreData("/tmp/edgecore", config, opts)
	assert.NoError(err)

	mkdirPatch.Reset()
	mkdirErrorPatch := setupMkdirPatch(t, "/tmp/edgecore", false)
	defer mkdirErrorPatch.Reset()

	err = collectEdgecoreData("/tmp/edgecore", config, opts)
	assert.Error(err)
	assert.Equal("directory creation failed", err.Error())

	mkdirErrorPatch.Reset()
	mkdirSuccessPatch := setupMkdirPatch(t, "/tmp/edgecore", true)
	defer mkdirSuccessPatch.Reset()

	configNoDataSource := &v1alpha2.EdgeCoreConfig{}
	noSourceDb := &v1alpha2.DataBase{
		DataSource: "",
	}
	configNoDataSource.DataBase = noSourceDb
	configNoDataSource.Modules = config.Modules

	err = collectEdgecoreData("/tmp/edgecore", configNoDataSource, opts)
	assert.NoError(err)

	optsNoLogPath := &common.CollectOptions{
		LogPath: "",
	}

	err = collectEdgecoreData("/tmp/edgecore", config, optsNoLogPath)
	assert.NoError(err)

	configNoTLS := &v1alpha2.EdgeCoreConfig{}
	configNoTLS.DataBase = config.DataBase

	noTlsHub := &v1alpha2.EdgeHub{
		TLSCertFile:       "",
		TLSPrivateKeyFile: "",
		TLSCAFile:         "",
	}

	noTlsModules := &v1alpha2.Modules{
		EdgeHub: noTlsHub,
	}

	configNoTLS.Modules = noTlsModules

	err = collectEdgecoreData("/tmp/edgecore", configNoTLS, opts)
	assert.NoError(err)

	copyFilePatch.Reset()
	copyFileErrorPatch := gomonkey.ApplyFunc(CopyFile, func(pathSrc, tmpPath string) error {
		if pathSrc == config.DataBase.DataSource {
			return errors.New("file copy failed")
		}
		return nil
	})
	defer copyFileErrorPatch.Reset()
	err = collectEdgecoreData("/tmp/edgecore", config, opts)
	assert.Error(err)
	assert.Equal("file copy failed", err.Error())
}

func TestCollectRuntimeData(t *testing.T) {
	assert := assert.New(t)

	mkdirPatch := setupMkdirPatch(t, "/tmp/runtime", true)
	defer mkdirPatch.Reset()

	copyFilePatch := gomonkey.ApplyFunc(CopyFile, func(pathSrc, tmpPath string) error {
		assert.Equal(common.PathDockerService, pathSrc)
		return nil
	})
	defer copyFilePatch.Reset()

	execShellPatch := setupExecShellPatch(true)
	defer execShellPatch.Reset()

	err := collectRuntimeData("/tmp/runtime")
	assert.NoError(err)

	mkdirPatch.Reset()
	mkdirErrorPatch := setupMkdirPatch(t, "/tmp/runtime", false)
	defer mkdirErrorPatch.Reset()

	err = collectRuntimeData("/tmp/runtime")
	assert.Error(err)
	assert.Equal("directory creation failed", err.Error())

	mkdirErrorPatch.Reset()
	mkdirSuccessPatch := setupMkdirPatch(t, "/tmp/runtime", true)
	defer mkdirSuccessPatch.Reset()

	copyFilePatch.Reset()
	copyFileErrorPatch := setupCopyFilePatch(false)
	defer copyFileErrorPatch.Reset()

	err = collectRuntimeData("/tmp/runtime")
	assert.Error(err)
	assert.Equal("file copy failed", err.Error())
}

func TestExecuteCollect(t *testing.T) {
	assert := assert.New(t)

	opts := &common.CollectOptions{
		Config:     "/path/to/config",
		OutputPath: "/path/to/output",
		Detail:     false,
	}

	verifyPatch := gomonkey.ApplyFunc(VerificationParameters, func(collectOptions *common.CollectOptions) error {
		return nil
	})
	defer verifyPatch.Reset()

	const timeStr = "2024_0101_120000"
	tmpDir := "/tmp/edge_" + timeStr
	makeDirPatch := gomonkey.ApplyFunc(makeDirTmp, func() (string, string, error) {
		return tmpDir, timeStr, nil
	})
	defer makeDirPatch.Reset()

	collectSystemPatch := gomonkey.ApplyFunc(collectSystemData, func(tmpPath string) error {
		assert.Equal(tmpDir+"/system", tmpPath)
		return nil
	})
	defer collectSystemPatch.Reset()

	config := &v1alpha2.EdgeCoreConfig{}
	parseConfigPatch := gomonkey.ApplyFunc(util.ParseEdgecoreConfig, func(configPath string) (*v1alpha2.EdgeCoreConfig, error) {
		assert.Equal(opts.Config, configPath)
		return config, nil
	})
	defer parseConfigPatch.Reset()

	collectEdgePatch := gomonkey.ApplyFunc(collectEdgecoreData, func(tmpPath string, config *v1alpha2.EdgeCoreConfig, ops *common.CollectOptions) error {
		assert.Equal(tmpDir+"/edgecore", tmpPath)
		return nil
	})
	defer collectEdgePatch.Reset()

	compressPatch := gomonkey.ApplyFunc(util.Compress, func(zipName string, sources []string) error {
		assert.Equal("/path/to/output/edge_"+timeStr+".tar.gz", zipName)
		assert.Equal([]string{tmpDir}, sources)
		return nil
	})
	defer compressPatch.Reset()

	removeAllPatch := gomonkey.ApplyFunc(os.RemoveAll, func(path string) error {
		assert.Equal(tmpDir, path)
		return nil
	})
	defer removeAllPatch.Reset()

	oldStdout := os.Stdout
	r, w, err := os.Pipe()
	assert.NoError(err)
	os.Stdout = w

	err = ExecuteCollect(opts)
	assert.NoError(err)

	w.Close()
	os.Stdout = oldStdout

	var buf bytes.Buffer
	_, err = buf.ReadFrom(r)
	assert.NoError(err)

	assert.Contains(buf.String(), "Start collecting data")
	assert.Contains(buf.String(), "Data collected successfully")

	verifyPatch.Reset()
	verifyErrorPatch := gomonkey.ApplyFunc(VerificationParameters, func(collectOptions *common.CollectOptions) error {
		return errors.New("parameter verification failed")
	})
	defer verifyErrorPatch.Reset()

	err = ExecuteCollect(opts)
	assert.Error(err)
	assert.Equal("parameter verification failed", err.Error())

	verifyErrorPatch.Reset()
	verifySuccessPatch := gomonkey.ApplyFunc(VerificationParameters, func(collectOptions *common.CollectOptions) error {
		return nil
	})
	defer verifySuccessPatch.Reset()

	makeDirPatch.Reset()
	makeDirErrorPatch := gomonkey.ApplyFunc(makeDirTmp, func() (string, string, error) {
		return "", "", errors.New("directory creation failed")
	})
	defer makeDirErrorPatch.Reset()

	err = ExecuteCollect(opts)
	assert.Error(err)
	assert.Equal("directory creation failed", err.Error())

	makeDirErrorPatch.Reset()
	makeDirSuccessPatch := gomonkey.ApplyFunc(makeDirTmp, func() (string, string, error) {
		return tmpDir, timeStr, nil
	})
	defer makeDirSuccessPatch.Reset()

	collectSystemPatch.Reset()
	collectSystemErrorPatch := gomonkey.ApplyFunc(collectSystemData, func(tmpPath string) error {
		return errors.New("system data collection failed")
	})
	defer collectSystemErrorPatch.Reset()

	r, w, err = os.Pipe()
	assert.NoError(err)
	os.Stdout = w

	err = ExecuteCollect(opts)
	assert.NoError(err)

	w.Close()
	os.Stdout = oldStdout

	buf.Reset()
	_, err = buf.ReadFrom(r)
	assert.NoError(err)

	assert.Contains(buf.String(), "collect System data failed")

	collectSystemErrorPatch.Reset()
	collectSystemSuccessPatch := gomonkey.ApplyFunc(collectSystemData, func(tmpPath string) error {
		return nil
	})
	defer collectSystemSuccessPatch.Reset()

	compressPatch.Reset()
	compressErrorPatch := gomonkey.ApplyFunc(util.Compress, func(zipName string, sources []string) error {
		return errors.New("compression failed")
	})
	defer compressErrorPatch.Reset()

	err = ExecuteCollect(opts)
	assert.Error(err)
	assert.Equal("compression failed", err.Error())
}
