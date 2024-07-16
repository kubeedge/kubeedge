/*
Copyright 2023 The KubeEdge Authors.

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

package testtools

import (
	"testing"

	"github.com/beego/beego/v2/client/orm"
	"github.com/golang/mock/gomock"

	"github.com/kubeedge/kubeedge/edge/mocks/beego"
	"github.com/kubeedge/kubeedge/edge/pkg/common/dbm"
)

func InitOrmerMock(t *testing.T) (*beego.MockOrmer, *beego.MockQuerySeter) {
	//Initialize Global Variables (Mocks)
	// ormerMock is mocked Ormer implementation
	var ormerMock *beego.MockOrmer
	// querySeterMock is mocked QuerySeter implementation
	var querySeterMock *beego.MockQuerySeter

	mockCtrl := gomock.NewController(t)
	defer mockCtrl.Finish()
	ormerMock = beego.NewMockOrmer(mockCtrl)
	querySeterMock = beego.NewMockQuerySeter(mockCtrl)
	dbm.DBAccess = ormerMock
	dbm.DefaultOrmFunc = func() orm.Ormer {
		return ormerMock
	}

	return ormerMock, querySeterMock
}
