/*
Copyright 2026 The KubeEdge Authors.

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

package metamanager

import (
	"errors"
	"testing"
	"time"

	"github.com/agiledragon/gomonkey/v2"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	beehiveContext "github.com/kubeedge/beehive/pkg/core/context"
	"github.com/kubeedge/beehive/pkg/core/model"
	cloudmodules "github.com/kubeedge/kubeedge/cloud/pkg/common/modules"
	"github.com/kubeedge/kubeedge/edge/pkg/metamanager/metaserver/kubernetes/storage/sqlite/imitator"
	fakeclient "github.com/kubeedge/kubeedge/edge/pkg/metamanager/metaserver/kubernetes/storage/sqlite/imitator/fake"
)

func newCloudMessage(operation, resource string) model.Message {
	msg := model.NewMessage("")
	msg.BuildRouter(cloudmodules.EdgeControllerModuleName, "resource", resource, operation)
	return *msg
}

// injectFails replaces the imitator with a fake whose Inject always fails, and
// records whether feedbackError and sendToCloud are called.
func injectFails(patches *gomonkey.Patches, gotErr *error, acked *bool) {
	patches.ApplyGlobalVar(&imitator.DefaultV2Client, fakeclient.Client{
		InjectF: func(_ model.Message) error {
			return errors.New("cache write failed")
		},
	})
	patches.ApplyFunc(feedbackError, func(err error, _ model.Message) {
		*gotErr = err
	})
	patches.ApplyFunc(sendToCloud, func(_ *model.Message) {
		*acked = true
	})
}

func TestProcessInsertFeedsBackErrorWhenInjectFails(t *testing.T) {
	m := &metaManager{}
	patches := gomonkey.NewPatches()
	defer patches.Reset()

	var gotErr error
	var acked bool
	injectFails(patches, &gotErr, &acked)

	m.processInsert(newCloudMessage(model.InsertOperation, "default/configmap/cm1"))
	require.Error(t, gotErr)
	assert.Contains(t, gotErr.Error(), "cache write failed")
	assert.False(t, acked, "the cloud must not be acked OK when the local cache write fails")
}

func TestProcessUpdateFeedsBackErrorWhenInjectFails(t *testing.T) {
	m := &metaManager{}
	patches := gomonkey.NewPatches()
	defer patches.Reset()

	var gotErr error
	var acked bool
	injectFails(patches, &gotErr, &acked)

	m.processUpdate(newCloudMessage(model.UpdateOperation, "default/configmap/cm1"))
	require.Error(t, gotErr)
	assert.Contains(t, gotErr.Error(), "cache write failed")
	assert.False(t, acked, "the cloud must not be acked OK when the local cache write fails")
}

func TestProcessDeleteFeedsBackErrorWhenInjectFails(t *testing.T) {
	m := &metaManager{}
	patches := gomonkey.NewPatches()
	defer patches.Reset()

	var gotErr error
	var acked bool
	injectFails(patches, &gotErr, &acked)

	m.processDelete(newCloudMessage(model.DeleteOperation, "default/configmap/cm1"))
	require.Error(t, gotErr)
	assert.Contains(t, gotErr.Error(), "cache write failed")
	assert.False(t, acked, "the cloud must not be acked OK when the local cache write fails")
}

func TestProcessVolumeFeedsBackErrorOnTimeout(t *testing.T) {
	m := &metaManager{}
	patches := gomonkey.NewPatches()
	defer patches.Reset()

	patches.ApplyFunc(beehiveContext.SendSync,
		func(_ string, _ model.Message, _ time.Duration) (model.Message, error) {
			return model.Message{}, errors.New("sync timeout")
		})
	var gotErr error
	patches.ApplyFunc(feedbackError, func(err error, _ model.Message) {
		gotErr = err
	})
	var acked bool
	patches.ApplyFunc(sendToCloud, func(_ *model.Message) {
		acked = true
	})

	m.processVolume(newCloudMessage(model.InsertOperation, "default/volume/v1"))
	require.Error(t, gotErr)
	assert.Contains(t, gotErr.Error(), "sync timeout")
	assert.False(t, acked, "the cloud must not be acked when the volume operation times out")
}
