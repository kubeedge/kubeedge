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
