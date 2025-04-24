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
	"errors"
	"fmt"
	"strings"

	"gorm.io/gorm"
	"k8s.io/apimachinery/pkg/runtime/schema"

	operationv1alpha1 "github.com/kubeedge/api/apis/operations/v1alpha1"
	operationv1alpha2 "github.com/kubeedge/api/apis/operations/v1alpha2"
	"github.com/kubeedge/kubeedge/common/types"
	"github.com/kubeedge/kubeedge/edge/pkg/metamanager/dao"
	"github.com/kubeedge/kubeedge/edge/pkg/metamanager/dao/models"
)

var (
	MetaGVRUpgrade = schema.GroupVersionResource{
		Group:    operationv1alpha2.GroupName,
		Version:  operationv1alpha2.Version,
		Resource: "nodeupgradejobspecs",
	}

	// Deprecated: For compatibility with v1alpha1 version, It will be removed in v1.23
	MetaGVRUpgradeV1alpha1 = schema.GroupVersionResource{
		Group:    operationv1alpha1.GroupName,
		Version:  operationv1alpha1.Version,
		Resource: "nodeupgradejobtaskrequests",
	}
)

// Upgrade is a dao wrapper for node upgrade job record.
// It's used to store information needed to continue upgrading after the upgrade is confirmed.
// Whether complete information about node jobs needs to be stored is another matter.
type Upgrade struct {
	db *gorm.DB
}

func NewUpgrade() *Upgrade {
	return &Upgrade{db: dao.GetDB()}
}

// Generates the key for meta_v2 table. The format is:
// /{group}/{version}/{resource}/{jobname}/nodes/{nodename}
func (dao *Upgrade) key(jobname, nodename string) string {
	items := []string{
		"/" + MetaGVRUpgrade.Group, // Start with '/'
		MetaGVRUpgrade.Version,
		MetaGVRUpgrade.Resource,
		jobname,
		"nodes",
		nodename,
	}
	return strings.Join(items, "/")
}

// Get returns jobname, nodename and NodeUpgradeJobSpec that query by GVR.
// The return values jobname and nodename are parsed from 'meta_v2.key'.
// The NodeUpgradeJobSpec is parsed from 'meta_v2.value'.
// Usually, only 1 or 0 rows of data should be queried by GVR.
func (dao *Upgrade) Get() (string, string, *operationv1alpha2.NodeUpgradeJobSpec, error) {
	row, err := onlyOneUpgradeRowByGVR(dao.db, MetaGVRUpgrade.String())
	if err != nil {
		return "", "", nil, err
	}
	if row == nil {
		return "", "", nil, nil
	}
	arrs := strings.Split(row.Key, "/")
	jobname, nodename := arrs[len(arrs)-3], arrs[len(arrs)-1]
	if row.Value == "" {
		return jobname, nodename, nil, nil
	}
	var spec operationv1alpha2.NodeUpgradeJobSpec
	if err := json.Unmarshal([]byte(row.Value), &spec); err != nil {
		return jobname, nodename, nil, fmt.Errorf("failed to unmarshal metav2 value to NodeTaskRequest, err: %v", err)
	}
	return jobname, nodename, &spec, nil
}

// Save saves jobname, nodename and NodeUpgradeJobSpec to metav2 table.
// A node will only retain one piece of upgrade data.
func (dao *Upgrade) Save(jobname, nodename string, spec *operationv1alpha2.NodeUpgradeJobSpec) error {
	// Cleaning up historical nodeupgradejobtaskrequests data
	if err := dao.Delete(); err != nil {
		return err
	}

	buff, err := json.Marshal(spec)
	if err != nil {
		return fmt.Errorf("failed to marshal NodeTaskRequest to json, err: %v", err)
	}
	meta := models.MetaV2{
		Key:                  dao.key(jobname, nodename),
		Name:                 jobname,
		Namespace:            models.NullNamespace,
		GroupVersionResource: MetaGVRUpgrade.String(),
		Value:                string(buff),
	}
	return NewMetaV2Service().InsertOrReplaceMetaV2(&meta)
}

// Delete deletes nodeupgradejobspecs resources from metav2 table.
func (dao *Upgrade) Delete() error {
	row, err := onlyOneUpgradeRowByGVR(dao.db, MetaGVRUpgrade.String())
	if err != nil {
		return err
	}
	if row == nil {
		return nil
	}

	return NewMetaV2Service().DeleteByKey(row.Key)
}

// onlyOneUpgradeRowByGVR returns the first row of data that query by GVR.
func onlyOneUpgradeRowByGVR(db *gorm.DB, gvr string) (*models.MetaV2, error) {
	var row models.MetaV2
	err := db.Where(models.GVR+" = ?", gvr).Limit(1).Find(&row).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("failed to query metav2 by %s GVR, err: %v", gvr, err)
	}
	if row.Key == "" {
		return nil, nil
	}
	return &row, nil
}

// UpgradeV1alpha1 is a dao wrapper for v1alpha1 node upgrade job record.
// It's used to store information needed to continue upgrading after the upgrade is confirmed.
// Deprecated: For compatibility with v1alpha1 version, It will be removed in v1.23
type UpgradeV1alpha1 struct {
	db *gorm.DB
}

func NewUpgradeV1alpha1() *UpgradeV1alpha1 {
	return &UpgradeV1alpha1{
		db: dao.GetDB(),
	}
}

// Generates the key for meta_v2 table. The format is:
// /{group}/{version}/{resource}/{taskID}
func (dao *UpgradeV1alpha1) key(taskID string) string {
	items := []string{
		"/" + MetaGVRUpgradeV1alpha1.Group, // Start with '/'
		MetaGVRUpgradeV1alpha1.Version,
		MetaGVRUpgradeV1alpha1.Resource,
		taskID,
	}
	return strings.Join(items, "/")
}

// Get returns the NodeTaskRequest that query by GVR and parsed from 'meta_v2.value'.
// Usually, only 1 or 0 rows of data should be queried by GVR.
func (dao *UpgradeV1alpha1) Get() (*types.NodeTaskRequest, error) {
	row, err := onlyOneUpgradeRowByGVR(dao.db, MetaGVRUpgradeV1alpha1.String())
	if err != nil {
		return nil, err
	}
	if row == nil || row.Value == "" {
		return nil, nil
	}
	var req types.NodeTaskRequest
	if err := json.Unmarshal([]byte(row.Value), &req); err != nil {
		return nil, fmt.Errorf("failed to unmarshal metav2 value to NodeTaskRequest, err: %v", err)
	}
	return &req, nil
}

// Save saves the NodeTaskRequest to metav2 table.
// A node will only retain one piece of upgrade data.
func (dao *UpgradeV1alpha1) Save(request *types.NodeTaskRequest) error {
	// Cleaning up historical nodeupgradejobtaskrequests data
	if err := dao.Delete(); err != nil {
		return err
	}

	buff, err := json.Marshal(request)
	if err != nil {
		return fmt.Errorf("failed to marshal NodeTaskRequest to json, err: %v", err)
	}
	meta := models.MetaV2{
		Key:                  dao.key(request.TaskID),
		Name:                 request.TaskID,
		Namespace:            models.NullNamespace,
		GroupVersionResource: MetaGVRUpgradeV1alpha1.String(),
		Value:                string(buff),
	}
	return NewMetaV2Service().InsertOrReplaceMetaV2(&meta)
}

// Delete deletes nodeupgradejobtaskrequests resources from metav2 table.
func (dao *UpgradeV1alpha1) Delete() error {
	row, err := onlyOneUpgradeRowByGVR(dao.db, MetaGVRUpgradeV1alpha1.String())
	if err != nil {
		return err
	}
	if row == nil {
		return nil
	}
	return NewMetaV2Service().DeleteByKey(row.Key)
}
