/*
Copyright 2020 The KubeEdge Authors.

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

package listener

import (
	"testing"

	"github.com/hashicorp/golang-lru"
	v1 "k8s.io/api/core/v1"
	metaV1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func TestCacheServiceList(t *testing.T) {
	fakeServiceList := make([]v1.Service, 0)
	fakeService1 := v1.Service{
		ObjectMeta: metaV1.ObjectMeta{
			Name: "svc1",
		},
	}
	fakeService2 := v1.Service{
		ObjectMeta: metaV1.ObjectMeta{
			Name: "svc2",
		},
	}
	fakeService3 := v1.Service{
		ObjectMeta: metaV1.ObjectMeta{
			Name: "svc3",
		},
	}
	fakeService4 := v1.Service{
		ObjectMeta: metaV1.ObjectMeta{
			Name: "svc4",
		},
	}
	fakeServiceList = append(fakeServiceList, fakeService1, fakeService2, fakeService3, fakeService4)
	// bug case
	testCache1, err := lru.New(10)
	if err != nil {
		t.Errorf("create testCache1 err: %v", err)
	}
	for _, svc := range fakeServiceList {
		testCache1.Add(svc.Name, &svc)
	}
	if v, ok := testCache1.Get("svc1"); !ok {
		t.Errorf("can't get svc1 from testCache1")
	} else {
		if svc, ok := v.(*v1.Service); !ok {
			t.Errorf("can't convert to *v1.Service")
		} else {
			if svc.Name != "svc4" {
				t.Errorf("svc name is not svc4")
			}
		}
	}
	// after fixing bug
	testCache2, err := lru.New(10)
	if err != nil {
		t.Errorf("create testCache2 err: %v", err)
	}
	for i := range fakeServiceList {
		testCache2.Add(fakeServiceList[i].Name, &fakeServiceList[i])
	}
	if v, ok := testCache2.Get("svc1"); !ok {
		t.Errorf("can't get svc1 from testCache2")
	} else {
		if svc, ok := v.(*v1.Service); !ok {
			t.Errorf("can't convert to *v1.Service")
		} else {
			if svc.Name != "svc1" {
				t.Errorf("svc name is not svc1")
			}
		}
	}
}
