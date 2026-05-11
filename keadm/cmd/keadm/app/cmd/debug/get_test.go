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

package debug

import (
	"testing"

	"github.com/kubeedge/beehive/pkg/core/model"
	"github.com/kubeedge/kubeedge/edge/pkg/metamanager/dao/models"
)

func TestFilterMetaRecords(t *testing.T) {
	records := []models.Meta{
		{Key: "default/" + model.ResourceTypePod + "/pod-a"},
		{Key: "default/" + model.ResourceTypePod + "/pod-b"},
		{Key: "kube-system/" + model.ResourceTypePod + "/pod-c"},
	}

	opts := &GetOptions{}
	got := opts.filterMetaRecords(records, "default", []string{"pod-a"})
	if len(got) != 1 || got[0].Key != records[0].Key {
		t.Fatalf("filterMetaRecords() = %#v, want only %q", got, records[0].Key)
	}

	opts.AllNamespace = true
	got = opts.filterMetaRecords(records, "default", nil)
	if len(got) != len(records) {
		t.Fatalf("filterMetaRecords() with all namespaces returned %d records, want %d", len(got), len(records))
	}
}

func TestFilterSelector(t *testing.T) {
	data := []models.Meta{
		{Key: "matched", Value: `{"metadata":{"labels":{"app":"edge","tier":"test"}}}`},
		{Key: "unlabeled", Value: `{"metadata":{}}`},
		{Key: "unmatched", Value: `{"metadata":{"labels":{"app":"cloud","tier":"test"}}}`},
	}

	got, err := FilterSelector(data, "app=edge,tier!=prod")
	if err != nil {
		t.Fatalf("FilterSelector() returned error: %v", err)
	}
	if len(got) != 2 {
		t.Fatalf("FilterSelector() returned %d records, want 2", len(got))
	}
	if got[0].Key != "matched" || got[1].Key != "unlabeled" {
		t.Fatalf("FilterSelector() = %#v, want matched and unlabeled records", got)
	}
}

func TestFilterSelectorReturnsErrorForMalformedLabels(t *testing.T) {
	data := []models.Meta{
		{Key: "malformed", Value: `{"metadata":{"labels":["app=edge"]}}`},
	}

	if _, err := FilterSelector(data, "app=edge"); err == nil {
		t.Fatal("FilterSelector() returned nil error for malformed labels")
	}
}
