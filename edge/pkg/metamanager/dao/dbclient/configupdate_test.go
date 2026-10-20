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

package dbclient

import (
	"errors"
	"os"
	"path/filepath"
	"testing"

	"github.com/agiledragon/gomonkey/v2"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"

	edgecorev1alpha2 "github.com/kubeedge/api/apis/componentconfig/edgecore/v1alpha2"
	operationv1alpha2 "github.com/kubeedge/api/apis/operations/v1alpha2"
	"github.com/kubeedge/kubeedge/edge/pkg/metamanager/dao"
	"github.com/kubeedge/kubeedge/edge/pkg/metamanager/dao/models"
)

// TestMain initializes the database once for the package, because dao.Init
// only opens the database on the first call.
func TestMain(m *testing.M) {
	dir, err := os.MkdirTemp("", "dbclient-test")
	if err != nil {
		panic(err)
	}
	dao.Init(filepath.Join(dir, "edgecore.db"), &edgecorev1alpha2.MetaManager{Enable: true})
	code := m.Run()
	_ = os.RemoveAll(dir)
	os.Exit(code)
}

func TestConfigUpdate(t *testing.T) {
	configUpdateDao := NewConfigUpdate()

	insertRecord := func(t *testing.T, jobname, nodename, value string) {
		require.NoError(t, NewMetaV2Service().InsertOrReplaceMetaV2(&models.MetaV2{
			Key:                  configUpdateDao.key(jobname, nodename),
			Name:                 jobname,
			Namespace:            models.NullNamespace,
			GroupVersionResource: MetaGVRConfigUpdate.String(),
			Value:                value,
		}))
	}

	t.Run("save, get and delete", func(t *testing.T) {
		jobname, nodename, spec, err := configUpdateDao.Get()
		require.NoError(t, err)
		assert.Empty(t, jobname)
		assert.Empty(t, nodename)
		assert.Nil(t, spec)

		firstSpec := &operationv1alpha2.ConfigUpdateJobSpec{
			UpdateFields: map[string]string{"modules.edged.tailoredKubeletConfig.imageGCHighThresholdPercent": "90"},
		}
		require.NoError(t, configUpdateDao.Save("job-1", "node-1", firstSpec))
		jobname, nodename, spec, err = configUpdateDao.Get()
		require.NoError(t, err)
		assert.Equal(t, "job-1", jobname)
		assert.Equal(t, "node-1", nodename)
		assert.Equal(t, firstSpec, spec)

		// A node only retains one config update record.
		secondSpec := &operationv1alpha2.ConfigUpdateJobSpec{
			UpdateFields: map[string]string{"modules.edgehub.heartbeat": "20"},
		}
		require.NoError(t, configUpdateDao.Save("job-2", "node-1", secondSpec))
		jobname, nodename, spec, err = configUpdateDao.Get()
		require.NoError(t, err)
		assert.Equal(t, "job-2", jobname)
		assert.Equal(t, "node-1", nodename)
		assert.Equal(t, secondSpec, spec)

		// Deleting the config update record does not touch the node upgrade record.
		upgradeDao := NewUpgrade()
		require.NoError(t, upgradeDao.Save("upgrade-job", "node-1", &operationv1alpha2.NodeUpgradeJobSpec{Version: "v1.21.0"}))
		require.NoError(t, configUpdateDao.Delete())

		jobname, nodename, spec, err = configUpdateDao.Get()
		require.NoError(t, err)
		assert.Empty(t, jobname)
		assert.Empty(t, nodename)
		assert.Nil(t, spec)

		upgradeJobname, upgradeNodename, upgradeSpec, err := upgradeDao.Get()
		require.NoError(t, err)
		assert.Equal(t, "upgrade-job", upgradeJobname)
		assert.Equal(t, "node-1", upgradeNodename)
		assert.Equal(t, "v1.21.0", upgradeSpec.Version)
		require.NoError(t, upgradeDao.Delete())
	})

	t.Run("record without spec", func(t *testing.T) {
		insertRecord(t, "job-3", "node-1", "")
		defer func() { require.NoError(t, configUpdateDao.Delete()) }()

		jobname, nodename, spec, err := configUpdateDao.Get()
		require.NoError(t, err)
		assert.Equal(t, "job-3", jobname)
		assert.Equal(t, "node-1", nodename)
		assert.Nil(t, spec)
	})

	t.Run("record with invalid spec", func(t *testing.T) {
		insertRecord(t, "job-4", "node-1", "{")
		defer func() { require.NoError(t, configUpdateDao.Delete()) }()

		jobname, nodename, spec, err := configUpdateDao.Get()
		require.ErrorContains(t, err, "failed to unmarshal metav2 value to ConfigUpdateJobSpec")
		assert.Equal(t, "job-4", jobname)
		assert.Equal(t, "node-1", nodename)
		assert.Nil(t, spec)
	})

	t.Run("failed to query record", func(t *testing.T) {
		patches := gomonkey.ApplyFunc(onlyOneUpgradeRowByGVR, func(*gorm.DB, string) (*models.MetaV2, error) {
			return nil, errors.New("test error")
		})
		defer patches.Reset()

		_, _, _, err := configUpdateDao.Get()
		require.ErrorContains(t, err, "test error")
		// Save cleans up the historical record first, so it fails as well.
		err = configUpdateDao.Save("job-5", "node-1", &operationv1alpha2.ConfigUpdateJobSpec{})
		require.ErrorContains(t, err, "test error")
		require.ErrorContains(t, configUpdateDao.Delete(), "test error")
	})
}
