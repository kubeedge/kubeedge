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

package handlerfactory

import "testing"

// FIXME: ...
func TestConfirmUpgrade(t *testing.T) {
	// patch := gomonkey.NewPatches()
	// defer patch.Reset()

	// f := NewFactory()

	// patch.ApplyFunc(options.GetEdgeCoreOptions, func() *options.EdgeCoreOptions {
	// 	return &options.EdgeCoreOptions{
	// 		ConfigFile: "/etc/kubeedge/config/edgecore.yaml",
	// 	}
	// })

	// patch.ApplyFunc(upgradedb.QueryNodeTaskRequestFromMetaV2, func() (types.NodeTaskRequest, error) {
	// 	return types.NodeTaskRequest{
	// 		TaskID: "task-123",
	// 		Type:   "upgrade",
	// 	}, nil
	// })

	// patch.ApplyFunc(upgradedb.QueryNodeUpgradeJobRequestFromMetaV2, func() (types.NodeUpgradeJobRequest, error) {
	// 	return types.NodeUpgradeJobRequest{
	// 		UpgradeID: "upgrade-123",
	// 		HistoryID: "history-123",
	// 		Version:   "v1.12.0",
	// 		Image:     "kubeedge/installation-package:v1.12.0",
	// 	}, nil
	// })

	// executorMock := &mockExecutor{}
	// patch.ApplyFunc(taskexecutor.GetExecutor, func(taskType string) (taskexecutor.Executor, error) {
	// 	return executorMock, nil
	// })

	// patch.ApplyFunc(klog.Errorf, func(format string, args ...interface{}) {})
	// patch.ApplyFunc(klog.Info, func(args ...interface{}) {})
	// patch.ApplyFunc(klog.Infof, func(format string, args ...interface{}) {})

	// patch.ApplyFunc(exec.Command, func(name string, args ...string) *exec.Cmd {
	// 	return &exec.Cmd{}
	// })

	// patch.ApplyMethod((*exec.Cmd)(nil), "CombinedOutput", func(_ *exec.Cmd) ([]byte, error) {
	// 	return []byte("upgrade successful"), nil
	// })

	// patch.ApplyFunc(upgradedb.DeleteNodeTaskRequestFromMetaV2, func() error {
	// 	return nil
	// })

	// patch.ApplyFunc(upgradedb.DeleteNodeUpgradeJobRequestFromMetaV2, func() error {
	// 	return nil
	// })

	// t.Run("ConfirmUpgrade success", func(t *testing.T) {
	// 	req := httptest.NewRequest("POST", "/confirm-upgrade", nil)
	// 	w := httptest.NewRecorder()

	// 	handler := f.ConfirmUpgrade()
	// 	handler.ServeHTTP(w, req)

	// 	resp := w.Result()
	// 	assert.Equal(t, http.StatusOK, resp.StatusCode)
	// })

	// t.Run("ConfirmUpgrade command error", func(t *testing.T) {
	// 	cmdErrorPatch := gomonkey.ApplyMethod((*exec.Cmd)(nil), "CombinedOutput",
	// 		func(_ *exec.Cmd) ([]byte, error) {
	// 			return []byte("command failed"), errors.New("command failed")
	// 		})
	// 	defer cmdErrorPatch.Reset()

	// 	req := httptest.NewRequest("POST", "/confirm-upgrade", nil)
	// 	w := httptest.NewRecorder()

	// 	handler := f.ConfirmUpgrade()
	// 	handler.ServeHTTP(w, req)

	// 	resp := w.Result()
	// 	body, err := io.ReadAll(resp.Body)
	// 	if err != nil {
	// 		t.Fatalf("Failed to read response body: %v", err)
	// 	}

	// 	assert.Equal(t, http.StatusInternalServerError, resp.StatusCode)
	// 	assert.Contains(t, string(body), "command failed")
	// })

	// t.Run("ConfirmUpgrade with db error handling", func(t *testing.T) {
	// 	dbErrorPatch := gomonkey.ApplyFunc(upgradedb.DeleteNodeTaskRequestFromMetaV2, func() error {
	// 		return errors.New("db delete error")
	// 	})
	// 	defer dbErrorPatch.Reset()

	// 	req := httptest.NewRequest("POST", "/confirm-upgrade", nil)
	// 	w := httptest.NewRecorder()

	// 	handler := f.ConfirmUpgrade()
	// 	handler.ServeHTTP(w, req)

	// 	resp := w.Result()
	// 	assert.Equal(t, http.StatusOK, resp.StatusCode)
	// })
}
