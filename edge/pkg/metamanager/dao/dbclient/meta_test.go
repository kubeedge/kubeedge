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
	"fmt"
	"path/filepath"
	"strings"
	"testing"

	"gorm.io/driver/sqlite"
	"gorm.io/gorm"

	"github.com/kubeedge/kubeedge/edge/pkg/metamanager/dao/models"
)

func TestQueryAllMetaByKeyPrefix(t *testing.T) {
	service := newTestMetaService(t)
	key := "logical_%_Token"
	entries := []models.Meta{
		{Key: key, Type: "serviceaccounttoken", Value: "legacy"},
		{Key: key + "/100", Type: "serviceaccounttoken", Value: "first"},
		{Key: key + "/200", Type: "serviceaccounttoken", Value: "second"},
		{Key: key + "-other/300", Type: "serviceaccounttoken", Value: "prefix-collision"},
		{Key: "logicalXabcYToken/400", Type: "serviceaccounttoken", Value: "wildcard-collision"},
		{Key: strings.ToUpper(key) + "/500", Type: "serviceaccounttoken", Value: "case-collision"},
	}
	for i := range entries {
		if err := service.InsertOrUpdate(&entries[i]); err != nil {
			t.Fatalf("failed to insert meta %q: %v", entries[i].Key, err)
		}
	}

	got, err := service.QueryAllMetaByKeyPrefix(key)
	if err != nil {
		t.Fatalf("QueryAllMetaByKeyPrefix() returned error: %v", err)
	}
	gotKeys := make(map[string]struct{}, len(*got))
	for _, meta := range *got {
		gotKeys[meta.Key] = struct{}{}
	}
	wantKeys := []string{key, key + "/100", key + "/200"}
	if len(gotKeys) != len(wantKeys) {
		t.Fatalf("QueryAllMetaByKeyPrefix() returned keys %v, want %v", gotKeys, wantKeys)
	}
	for _, want := range wantKeys {
		if _, ok := gotKeys[want]; !ok {
			t.Errorf("QueryAllMetaByKeyPrefix() missing key %q", want)
		}
	}
}

func TestQueryAllMetaByKeyPrefixUsesIndex(t *testing.T) {
	service := newTestMetaService(t)
	key := "logical-token-key"
	generationStart := key + "/"
	generationEnd := key + "0"
	var plan []struct {
		Detail string
	}
	query := fmt.Sprintf("EXPLAIN QUERY PLAN SELECT * FROM %s WHERE %s", models.MetaTableName, metaKeyPrefixCondition)
	if err := service.db.Raw(query, key, generationStart, generationEnd).Scan(&plan).Error; err != nil {
		t.Fatalf("failed to explain prefix query: %v", err)
	}

	foundSearch := false
	for _, step := range plan {
		if strings.Contains(step.Detail, "SCAN "+models.MetaTableName) {
			t.Fatalf("prefix query performs a full table scan: %q", step.Detail)
		}
		if strings.Contains(step.Detail, "SEARCH "+models.MetaTableName) {
			foundSearch = true
		}
	}
	if !foundSearch {
		t.Fatalf("prefix query plan does not use an indexed search: %+v", plan)
	}
}

func newTestMetaService(t *testing.T) *MetaService {
	t.Helper()
	db, err := gorm.Open(sqlite.Open(filepath.Join(t.TempDir(), "meta.db")), &gorm.Config{})
	if err != nil {
		t.Fatalf("failed to open test database: %v", err)
	}
	if err := db.AutoMigrate(&models.Meta{}); err != nil {
		t.Fatalf("failed to migrate test database: %v", err)
	}
	return &MetaService{db: db}
}
