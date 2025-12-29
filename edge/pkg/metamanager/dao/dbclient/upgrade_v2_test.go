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

package dbclient

import (
	"encoding/json"
	"fmt"
	"reflect"
	"testing"

	"github.com/agiledragon/gomonkey/v2"
	"github.com/beego/beego/v2/client/orm"
	"github.com/golang/mock/gomock"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	operationv1alpha1 "github.com/kubeedge/api/apis/operations/v1alpha1"
	operationv1alpha2 "github.com/kubeedge/api/apis/operations/v1alpha2"
	"github.com/kubeedge/kubeedge/common/types"
	"github.com/kubeedge/kubeedge/edge/mocks/beego"
)

func TestUpgradeV1alpha1Key(t *testing.T) {
	dao := NewUpgradeV1alpha1()
	key := dao.key("test-job")
	assert.Equal(t, "/"+operationv1alpha1.GroupName+"/"+operationv1alpha1.Version+"/nodeupgradejobtaskrequests/test-job", key)
}

func TestUpgradeV1alpha1Get(t *testing.T) {
	t.Run("found one", func(t *testing.T) {
		patches := gomonkey.NewPatches()
		defer patches.Reset()

		patches.ApplyFunc(onlyOneUpgradeRowByGVR, func(_ormer orm.Ormer, _gvr string) (*MetaV2, error) {
			return fakeUpgradeV1alpha1MetaV2()
		})

		dao := NewUpgradeV1alpha1()
		req, err := dao.Get()
		require.NoError(t, err)
		require.NotNil(t, req)
		assert.Equal(t, "test-job", req.TaskID)
		assert.Equal(t, "Upgrade", req.Type)
		assert.Equal(t, "Upgrading", req.State)
		assert.NotNil(t, req.Item)
	})

	t.Run("not found", func(t *testing.T) {
		patches := gomonkey.NewPatches()
		defer patches.Reset()

		patches.ApplyFunc(onlyOneUpgradeRowByGVR, func(_ormer orm.Ormer, _gvr string) (*MetaV2, error) {
			return nil, nil
		})

		dao := NewUpgradeV1alpha1()
		req, err := dao.Get()
		require.NoError(t, err)
		assert.Nil(t, req)

		patches.ApplyFunc(onlyOneUpgradeRowByGVR, func(_ormer orm.Ormer, _gvr string) (*MetaV2, error) {
			return &MetaV2{Value: ""}, nil
		})

		dao = NewUpgradeV1alpha1()
		req, err = dao.Get()
		require.NoError(t, err)
		assert.Nil(t, req)
	})
}

func TestUpgradeV1alpha1Save(t *testing.T) {
	mockCtrl := gomock.NewController(t)
	defer mockCtrl.Finish()

	t.Run("save successful", func(t *testing.T) {
		ormerMock := beego.NewMockOrmer(mockCtrl)
		dao := &UpgradeV1alpha1{
			db: ormerMock,
		}

		patches := gomonkey.NewPatches()
		defer patches.Reset()

		patches.ApplyMethodFunc(reflect.TypeOf(dao), "Delete",
			func() error {
				return nil
			})

		ormerMock.EXPECT().Delete(gomock.Any(), gomock.Any()).DoAndReturn(func(md interface{}, cols ...string) (int64, error) {
			return 1, nil
		}).Times(0)

		ormerMock.EXPECT().Insert(gomock.Any()).DoAndReturn(func(md interface{}) (int64, error) {
			row, ok := md.(*MetaV2)
			require.True(t, ok)
			assert.Equal(t, "/"+operationv1alpha1.GroupName+"/"+operationv1alpha1.Version+"/nodeupgradejobtaskrequests/test-job", row.Key)
			assert.Equal(t, "test-job", row.Name)
			assert.Equal(t, MetaGVRUpgradeV1alpha1.String(), row.GroupVersionResource)
			assert.NotEmpty(t, row.Value)
			return 1, nil
		}).Times(1)

		err := dao.Save(&types.NodeTaskRequest{
			TaskID: "test-job",
			Type:   "Upgrade",
			State:  "Upgrading",
			Item: &types.NodeUpgradeJobRequest{
				UpgradeID:           "test-job",
				HistoryID:           "history",
				Version:             "v1.20.0",
				UpgradeTool:         "keadm",
				Image:               "kubeedge/installation-package:v1.20.0",
				ImageDigest:         "sha256:1234567890",
				RequireConfirmation: false,
			},
		})
		require.NoError(t, err)
	})
}

func TestUpgradeV1alpha1Delete(t *testing.T) {
	mockCtrl := gomock.NewController(t)
	defer mockCtrl.Finish()

	t.Run("not found row", func(t *testing.T) {
		patches := gomonkey.NewPatches()
		defer patches.Reset()

		patches.ApplyFunc(onlyOneUpgradeRowByGVR, func(_ormer orm.Ormer, _gvr string) (*MetaV2, error) {
			return nil, nil
		})

		ormerMock := beego.NewMockOrmer(mockCtrl)
		ormerMock.EXPECT().Delete(gomock.Any(), gomock.Any()).
			DoAndReturn(func(_md interface{}, _cols ...string) (int64, error) {
				return 1, nil
			}).Times(0)

		dao := &UpgradeV1alpha1{
			db: ormerMock,
		}
		err := dao.Delete()
		require.NoError(t, err)
	})

	t.Run("found row and delete it", func(t *testing.T) {
		patches := gomonkey.NewPatches()
		defer patches.Reset()

		patches.ApplyFunc(onlyOneUpgradeRowByGVR, func(_ormer orm.Ormer, _gvr string) (*MetaV2, error) {
			return fakeUpgradeV1alpha1MetaV2()
		})

		ormerMock := beego.NewMockOrmer(mockCtrl)
		ormerMock.EXPECT().Delete(gomock.Any(), gomock.Any()).
			DoAndReturn(func(_md interface{}, _cols ...string) (int64, error) {
				return 1, nil
			}).Times(1)

		dao := &UpgradeV1alpha1{
			db: ormerMock,
		}
		err := dao.Delete()
		require.NoError(t, err)
	})
}

func fakeUpgradeV1alpha1MetaV2() (*MetaV2, error) {
	req := &types.NodeTaskRequest{
		TaskID: "test-job",
		Type:   "Upgrade",
		State:  "Upgrading",
		Item: &types.NodeUpgradeJobRequest{
			UpgradeID:           "test-job",
			HistoryID:           "history",
			Version:             "v1.20.0",
			UpgradeTool:         "keadm",
			Image:               "kubeedge/installation-package:v1.20.0",
			ImageDigest:         "sha256:1234567890",
			RequireConfirmation: false,
		},
	}
	data, err := json.Marshal(req)
	if err != nil {
		return nil, err
	}
	return &MetaV2{
		Value: string(data),
	}, nil
}

func TestUpgradeMetaV2Get(t *testing.T) {
	t.Run("found one", func(t *testing.T) {
		dao := NewUpgrade()
		patches := gomonkey.NewPatches()
		defer patches.Reset()

		patches.ApplyFunc(onlyOneUpgradeRowByGVR, func(_ormer orm.Ormer, _gvr string) (*MetaV2, error) {
			return fakeUpgradeMetaV2(dao)
		})

		jobname, nodename, spec, err := dao.Get()
		require.NoError(t, err)
		assert.Equal(t, "test-job", jobname)
		assert.Equal(t, "test-node", nodename)
		require.NotNil(t, spec)
	})

	t.Run("not found", func(t *testing.T) {
		dao := NewUpgrade()
		patches := gomonkey.NewPatches()
		defer patches.Reset()

		patches.ApplyFunc(onlyOneUpgradeRowByGVR, func(_ormer orm.Ormer, _gvr string) (*MetaV2, error) {
			return nil, nil
		})

		jobname, nodename, spec, err := dao.Get()
		require.NoError(t, err)
		assert.Empty(t, jobname)
		assert.Empty(t, nodename)
		assert.Nil(t, spec)

		patches.ApplyFunc(onlyOneUpgradeRowByGVR, func(_ormer orm.Ormer, _gvr string) (*MetaV2, error) {
			return &MetaV2{
				Key:       dao.key("test-job", "test-node"),
				Name:      "test-job",
				Namespace: NullNamespace,
			}, nil
		})

		jobname, nodename, spec, err = dao.Get()
		require.NoError(t, err)
		assert.Equal(t, "test-job", jobname)
		assert.Equal(t, "test-node", nodename)
		assert.Nil(t, spec)
	})
}

func TestUpgradeMetaV2Save(t *testing.T) {
	mockCtrl := gomock.NewController(t)
	defer mockCtrl.Finish()

	t.Run("save successful", func(t *testing.T) {
		ormerMock := beego.NewMockOrmer(mockCtrl)
		dao := &Upgrade{
			db: ormerMock,
		}

		patches := gomonkey.NewPatches()
		defer patches.Reset()

		patches.ApplyMethodFunc(reflect.TypeOf(dao), "Delete",
			func() error {
				return nil
			})

		ormerMock.EXPECT().Delete(gomock.Any(), gomock.Any()).DoAndReturn(func(md interface{}, cols ...string) (int64, error) {
			return 1, nil
		}).Times(0)

		ormerMock.EXPECT().Insert(gomock.Any()).DoAndReturn(func(md interface{}) (int64, error) {
			row, ok := md.(*MetaV2)
			require.True(t, ok)
			expectKey := fmt.Sprintf("/%s/%s/nodeupgradejobspecs/test-job/nodes/test-node",
				operationv1alpha2.GroupName, operationv1alpha2.Version)
			assert.Equal(t, expectKey, row.Key)
			assert.Equal(t, "test-job", row.Name)
			assert.Equal(t, MetaGVRUpgrade.String(), row.GroupVersionResource)
			assert.NotEmpty(t, row.Value)
			return 1, nil
		}).Times(1)

		err := dao.Save("test-job", "test-node", &operationv1alpha2.NodeUpgradeJobSpec{
			Version:             "v1.20.0",
			Image:               "kubeedge/installation-package:v1.20.0",
			CheckItems:          []string{"cpu", "memory", "disk"},
			RequireConfirmation: true,
		})
		require.NoError(t, err)
	})
}

func TestUpgradeMetaV2Delete(t *testing.T) {
	mockCtrl := gomock.NewController(t)
	defer mockCtrl.Finish()

	t.Run("not found row", func(t *testing.T) {
		patches := gomonkey.NewPatches()
		defer patches.Reset()

		patches.ApplyFunc(onlyOneUpgradeRowByGVR, func(_ormer orm.Ormer, _gvr string) (*MetaV2, error) {
			return nil, nil
		})

		ormerMock := beego.NewMockOrmer(mockCtrl)
		ormerMock.EXPECT().Delete(gomock.Any(), gomock.Any()).
			DoAndReturn(func(_md interface{}, _cols ...string) (int64, error) {
				return 1, nil
			}).Times(0)

		dao := &Upgrade{
			db: ormerMock,
		}
		err := dao.Delete()
		require.NoError(t, err)
	})

	t.Run("found row and delete it", func(t *testing.T) {
		ormerMock := beego.NewMockOrmer(mockCtrl)
		dao := &Upgrade{
			db: ormerMock,
		}

		patches := gomonkey.NewPatches()
		defer patches.Reset()

		patches.ApplyFunc(onlyOneUpgradeRowByGVR, func(_ormer orm.Ormer, _gvr string) (*MetaV2, error) {
			return fakeUpgradeMetaV2(dao)
		})

		ormerMock.EXPECT().Delete(gomock.Any(), gomock.Any()).
			DoAndReturn(func(_md interface{}, _cols ...string) (int64, error) {
				return 1, nil
			}).Times(1)

		err := dao.Delete()
		require.NoError(t, err)
	})
}

func fakeUpgradeMetaV2(dao *Upgrade) (*MetaV2, error) {
	timeout := uint32(300)
	spec := &operationv1alpha2.NodeUpgradeJobSpec{
		Version:             "v1.20.0",
		TimeoutSeconds:      &timeout,
		Image:               "kubeedge/installation-package:v1.20.0",
		CheckItems:          []string{"cpu", "memory", "disk"},
		RequireConfirmation: true,
	}
	data, err := json.Marshal(spec)
	if err != nil {
		return nil, err
	}
	return &MetaV2{
		Key:                  dao.key("test-job", "test-node"),
		Name:                 "test-job",
		Namespace:            NullNamespace,
		Value:                string(data),
		GroupVersionResource: MetaGVRUpgrade.String(),
	}, nil
}

func TestOnlyOneUpgradeRowByGVR(t *testing.T) {
	mockCtrl := gomock.NewController(t)
	defer mockCtrl.Finish()
	ormerMock := beego.NewMockOrmer(mockCtrl)

	t.Run("found one", func(t *testing.T) {
		querySeterMock := beego.NewMockQuerySeter(mockCtrl)
		ormerMock.EXPECT().QueryTable(gomock.Any()).Return(querySeterMock)
		querySeterMock.EXPECT().Filter(gomock.Any(), gomock.Any()).Return(querySeterMock).Times(1)
		querySeterMock.EXPECT().All(gomock.Any()).SetArg(0, []MetaV2{{}}).Return(int64(1), nil).Times(1)
		row, err := onlyOneUpgradeRowByGVR(ormerMock, MetaGVRUpgradeV1alpha1.String())
		require.NoError(t, err)
		assert.NotNil(t, row)
	})

	t.Run("not found", func(t *testing.T) {
		querySeterMock := beego.NewMockQuerySeter(mockCtrl)
		ormerMock.EXPECT().QueryTable(gomock.Any()).Return(querySeterMock)
		querySeterMock.EXPECT().Filter(gomock.Any(), gomock.Any()).Return(querySeterMock).Times(1)
		querySeterMock.EXPECT().All(gomock.Any()).Return(int64(1), nil).Times(1)
		row, err := onlyOneUpgradeRowByGVR(ormerMock, MetaGVRUpgradeV1alpha1.String())
		require.NoError(t, err)
		assert.Nil(t, row)
	})
}
