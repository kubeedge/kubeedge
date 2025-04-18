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

package v2

import (
	"encoding/json"
	"fmt"
	"strings"

	"github.com/beego/beego/v2/client/orm"
	"k8s.io/apimachinery/pkg/runtime/schema"

	operationv1alpha1 "github.com/kubeedge/api/apis/operations/v1alpha1"
	operationv1alpha2 "github.com/kubeedge/api/apis/operations/v1alpha2"
	"github.com/kubeedge/kubeedge/common/types"
	"github.com/kubeedge/kubeedge/edge/pkg/common/dbm"
)

var (
	MetaGVRUpgradeV1alpha1 = schema.GroupVersionResource{
		Group:    operationv1alpha1.GroupName,
		Version:  operationv1alpha1.Version,
		Resource: "nodeupgradejobtaskrequests",
	}

	MetaGVRUpgrade = schema.GroupVersionResource{
		Group:    operationv1alpha2.GroupName,
		Version:  operationv1alpha2.Version,
		Resource: "nodeupgradejobspecs",
	}
)

// UpgradeV1alpha1 is a dao wrapper for v1alpha1 node upgrade job
// Deprecated: For compatibility with v1alpha1 version, It will be removed in v1.23
type UpgradeV1alpha1 struct {
	db orm.Ormer
}

func NewUpgradeV1alpha1() *UpgradeV1alpha1 {
	return &UpgradeV1alpha1{
		db: dbm.DBAccess,
	}
}

func (dao *UpgradeV1alpha1) key(taskID string) string {
	items := []string{
		"/" + MetaGVRUpgradeV1alpha1.Group, // Start with '/'
		MetaGVRUpgradeV1alpha1.Version,
		MetaGVRUpgradeV1alpha1.Resource,
		taskID,
	}
	return strings.Join(items, "/")
}

func (dao *UpgradeV1alpha1) onlyOne() (*MetaV2, error) {
	var row MetaV2
	if err := dao.db.QueryTable(NewMetaTableName).
		Filter(GVR, MetaGVRUpgradeV1alpha1.String()).One(&row); err != nil {
		return nil, fmt.Errorf("failed to query metav2 by %s GVR, err: %v", MetaGVRUpgradeV1alpha1.String(), err)
	}
	return &row, nil
}

func (dao *UpgradeV1alpha1) Get() (*types.NodeTaskRequest, error) {
	row, err := dao.onlyOne()
	if err != nil {
		return nil, err
	}
	if row.Value == "" {
		return nil, nil
	}
	var req types.NodeTaskRequest
	if err := json.Unmarshal([]byte(row.Value), &req); err != nil {
		return nil, fmt.Errorf("failed to unmarshal metav2 value to NodeTaskRequest, err: %v", err)
	}
	return &req, nil
}

func (dao *UpgradeV1alpha1) Save(request *types.NodeTaskRequest) error {
	buff, err := json.Marshal(request)
	if err != nil {
		return fmt.Errorf("failed to marshal NodeTaskRequest to json, err: %v", err)
	}
	meta := MetaV2{
		Key:                  dao.key(request.TaskID),
		Name:                 request.TaskID,
		GroupVersionResource: MetaGVRUpgradeV1alpha1.String(),
		Value:                string(buff),
	}
	if _, err := dao.db.InsertOrUpdate(&meta); err != nil {
		return fmt.Errorf("failed to save NodeTaskRequest to metav2, err: %v", err)
	}
	return nil
}

func (dao *UpgradeV1alpha1) Delete() error {
	row, err := dao.onlyOne()
	if err != nil {
		return err
	}
	if _, err := dao.db.Delete(row); err != nil {
		return fmt.Errorf("failed to delete NodeTaskRequest by key %s, err: %v", row.Key, err)
	}
	return nil
}

// Upgrade is a dao wrapper for node upgrade job
type Upgrade struct {
	db orm.Ormer
}

func NewUpgrade() *Upgrade {
	return &Upgrade{
		db: dbm.DBAccess,
	}
}

func (dao *Upgrade) Get() (*operationv1alpha2.NodeUpgradeJobSpec, error) {
	return nil, nil
}

func (dao *Upgrade) Save(jobname, nodename string, spec *operationv1alpha2.NodeUpgradeJobSpec) error {
	return nil
}

func (dao *Upgrade) Delete() error {
	return nil
}
