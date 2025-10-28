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

package dao

import (
	"k8s.io/klog/v2"

	"github.com/kubeedge/kubeedge/edge/pkg/common/dbm"
)

const (
	EndpointUrlsName = "endpoint_urls"
)

type EndpointUrls struct {
	Endpoint string `orm:"column(endpoint);type(text);pk"`
	URL      string `orm:"column(url);type(text)"`
}

func InsertEpUrls(endpoint, url string) error {
	_, err := dbm.DBAccess.Raw(
		"INSERT INTO endpoint_urls (endpoint, url) VALUES (?, ?)", endpoint, url).Exec()
	klog.V(4).Infof("INSERT result %v", err)
	return err
}

func DeleteEpUrlsByKey(endpoint string) error {
	num, err := dbm.DBAccess.QueryTable(EndpointUrlsName).Filter("endpoint", endpoint).Delete()
	klog.V(4).Infof("Delete affected Num: %d, %v", num, err)
	return err
}

func IsTableEmpty() bool {
	var count int64
	if count, _ = dbm.DBAccess.QueryTable(EndpointUrlsName).Count(); count > 0 {
		return false
	}
	return true
}

func GetEpUrlsByKey(endpoint string) (*EndpointUrls, error) {
	targetUrls := new(EndpointUrls)
	if err := dbm.DBAccess.QueryTable(EndpointUrlsName).Filter("endpoint", endpoint).One(targetUrls); err != nil {
		return nil, err
	}
	return targetUrls, nil
}

func GetAllEpUrls() ([]*EndpointUrls, error) {
	var EpUrls []*EndpointUrls
	_, err := dbm.DBAccess.QueryTable(EndpointUrlsName).All(&EpUrls)
	if err != nil {
		return nil, err
	}
	return EpUrls, nil
}
