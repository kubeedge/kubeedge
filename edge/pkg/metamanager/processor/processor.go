package processor

import (
	"k8s.io/klog/v2"

	"github.com/kubeedge/beehive/pkg/core/model"
)

var (
	processors = make(map[queryKey]Processor)
)

type queryKey struct {
	resType   string
	operation string
}

type Processor interface {
	Process(message model.Message)
}

func Process(message model.Message) {
	op := message.GetOperation()
	_, resType, _ := parseResource(message.GetResource())

	p := processors[queryKey{resType: resType, operation: op}]
	if resType != "" && p == nil {
		p = processors[queryKey{operation: op}]
	}
	if p != nil {
		p.Process(message)
	} else {
		klog.Errorf("No processor to process message,op:%v,resType:%v", op, resType)
	}
}
