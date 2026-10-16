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
	"encoding/json"
	"fmt"
	"strings"

	"gorm.io/gorm"
	"k8s.io/apimachinery/pkg/runtime/schema"

	operationv1alpha2 "github.com/kubeedge/api/apis/operations/v1alpha2"
	"github.com/kubeedge/kubeedge/edge/pkg/metamanager/dao"
	"github.com/kubeedge/kubeedge/edge/pkg/metamanager/dao/models"
)

var MetaGVRConfigUpdate = schema.GroupVersionResource{
	Group:    operationv1alpha2.GroupName,
	Version:  operationv1alpha2.Version,
	Resource: "configupdatejobspecs",
}

// ConfigUpdate is a dao wrapper for config update job record.
// keadm config-update restarts EdgeCore before the Update action can report its result,
// so the record is used to report the result after EdgeCore is restarted.
type ConfigUpdate struct {
	db *gorm.DB
}

func NewConfigUpdate() *ConfigUpdate {
	return &ConfigUpdate{db: dao.GetDB()}
}

// Generates the key for meta_v2 table. The format is:
// /{group}/{version}/{resource}/{jobname}/nodes/{nodename}
func (dao *ConfigUpdate) key(jobname, nodename string) string {
	items := []string{
		"/" + MetaGVRConfigUpdate.Group, // Start with '/'
		MetaGVRConfigUpdate.Version,
		MetaGVRConfigUpdate.Resource,
		jobname,
		"nodes",
		nodename,
	}
	return strings.Join(items, "/")
}

// Get returns jobname, nodename and ConfigUpdateJobSpec that query by GVR.
// The return values jobname and nodename are parsed from 'meta_v2.key'.
// The ConfigUpdateJobSpec is parsed from 'meta_v2.value'.
// Usually, only 1 or 0 rows of data should be queried by GVR.
func (dao *ConfigUpdate) Get() (string, string, *operationv1alpha2.ConfigUpdateJobSpec, error) {
	row, err := onlyOneUpgradeRowByGVR(dao.db, MetaGVRConfigUpdate.String())
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
	var spec operationv1alpha2.ConfigUpdateJobSpec
	if err := json.Unmarshal([]byte(row.Value), &spec); err != nil {
		return jobname, nodename, nil, fmt.Errorf("failed to unmarshal metav2 value to ConfigUpdateJobSpec, err: %v", err)
	}
	return jobname, nodename, &spec, nil
}

// Save saves jobname, nodename and ConfigUpdateJobSpec to metav2 table.
// A node will only retain one piece of config update data.
func (dao *ConfigUpdate) Save(jobname, nodename string, spec *operationv1alpha2.ConfigUpdateJobSpec) error {
	// Cleaning up historical configupdatejobspecs data
	if err := dao.Delete(); err != nil {
		return err
	}

	buff, err := json.Marshal(spec)
	if err != nil {
		return fmt.Errorf("failed to marshal ConfigUpdateJobSpec to json, err: %v", err)
	}
	meta := models.MetaV2{
		Key:                  dao.key(jobname, nodename),
		Name:                 jobname,
		Namespace:            models.NullNamespace,
		GroupVersionResource: MetaGVRConfigUpdate.String(),
		Value:                string(buff),
	}
	return NewMetaV2Service().InsertOrReplaceMetaV2(&meta)
}

// Delete deletes configupdatejobspecs resources from metav2 table.
func (dao *ConfigUpdate) Delete() error {
	row, err := onlyOneUpgradeRowByGVR(dao.db, MetaGVRConfigUpdate.String())
	if err != nil {
		return err
	}
	if row == nil {
		return nil
	}
	return NewMetaV2Service().DeleteByKey(row.Key)
}
