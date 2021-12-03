/*
Copyright 2021 The KubeEdge Authors.

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
	TargetUrlsName = "target_urls"
)

type TargetUrls struct {
	URL string `orm:"column(url);type(text);pk"`
}

// InsertUrls insert target_urls
func InsertUrls(url string) error {
	_, err := dbm.DBAccess.Raw("INSERT OR REPLACE INTO target_urls (url) VALUES (?)", url).Exec()
	klog.V(4).Infof("INSERT result %v", err)
	return err
}

// DeleteUrlsByKey delete target_urls by key
func DeleteUrlsByKey(key string) error {
	num, err := dbm.DBAccess.QueryTable(TargetUrlsName).Filter("url", key).Delete()
	klog.V(4).Infof("Delete affected Num: %d, %v", num, err)
	return err
}

func IsTableEmpty() bool {
	var count int64
	if count, _ = dbm.DBAccess.QueryTable(TargetUrlsName).Count(); count > 0 {
		return false
	}
	return true
}

func GetUrlsByKey(key string) (result *TargetUrls, err error) {
	targetUrls := new(TargetUrls)
	if err := dbm.DBAccess.QueryTable(TargetUrlsName).Filter("url", key).One(targetUrls); err != nil {
		return nil, err
	}
	return targetUrls, nil
}
