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

package dao

import (
	"k8s.io/klog/v2"

	"github.com/kubeedge/kubeedge/edge/pkg/common/dbm"
)

const (
	SubTopicsName = "sub_topics"
)

type SubTopics struct {
	Topic string `orm:"column(topic); type(text); pk"`
}

// InsertTopics insert sub_topics
func InsertTopics(topic string) error {
	_, err := dbm.DBAccess.Raw("INSERT OR REPLACE INTO sub_topics (topic) VALUES (?)", topic).Exec()
	klog.V(4).Infof("INSERT result %v", err)
	return err
}

// DeleteTopicsByKey delete sub_topics by key
func DeleteTopicsByKey(key string) error {
	num, err := dbm.DBAccess.QueryTable(SubTopicsName).Filter("topic", key).Delete()
	klog.V(4).Infof("Delete affected Num: %d, %v", num, err)
	return err
}

// QueryAllTopics return all sub_topics, if no error, SubTopics not null
func QueryAllTopics() (*[]string, error) {
	event := new([]SubTopics)
	_, err := dbm.DBAccess.QueryTable(SubTopicsName).All(event)
	if err != nil {
		return nil, err
	}
	var result []string
	for _, v := range *event {
		result = append(result, v.Topic)
	}
	return &result, nil
}
