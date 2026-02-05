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

package models

// Meta metadata object
type Meta struct {
	Key   string `gorm:"primaryKey;size:256"`
	Type  string `gorm:"size:32"`
	Value string `gorm:"type:text"`
}

// TableName sets the insert table name for this struct type
func (Meta) TableName() string {
	return MetaTableName
}

// MetaV2 record k8s api object
type MetaV2 struct {
	Key                  string `gorm:"column:key;primaryKey;size:256"`
	GroupVersionResource string `gorm:"column:group_version_resource;size:256"`
	Namespace            string `gorm:"column:namespace;size:256"`
	Name                 string `gorm:"column:name;size:256"`
	ResourceVersion      uint64 `gorm:"column:resource_version"`
	Value                string `gorm:"column:value;type:text"`
}

// TableName sets the insert table name for this struct type
func (MetaV2) TableName() string {
	return NewMetaTableName
}
