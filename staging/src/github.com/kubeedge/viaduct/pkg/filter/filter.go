/*
Copyright 2019 The KubeEdge Authors.

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

package filter

import (
	"k8s.io/klog"

	"github.com/kubeedge/beehive/pkg/core/model"
)

type FilterFunc func(*model.Message) error

type MessageFilter struct {
	Filters []FilterFunc
	Index   int
}

func (filter *MessageFilter) AddFilterFunc(filterFunc FilterFunc) {
	filter.Filters = append(filter.Filters, filterFunc)
}

func (filter *MessageFilter) ProcessFilter(msg *model.Message) error {
	for _, filterFunc := range filter.Filters {
		err := filterFunc(msg)
		if err != nil {
			klog.Warningf("the message(%s) have been filtered", msg.GetID())
			return err
		}
	}
	return nil
}
