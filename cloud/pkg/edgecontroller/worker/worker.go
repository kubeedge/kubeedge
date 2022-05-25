/*
Copyright 2022 The KubeEdge Authors.

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

package worker

import (
	"k8s.io/klog/v2"

	"github.com/kubeedge/beehive/pkg/core/context"
	"github.com/kubeedge/beehive/pkg/core/model"
)

// Worker defines interface for resource message process worker.
// It is an simple implementation of the producer-consumer pattern.
type Worker interface {
	Start()
	Consume()
	Produce(msg model.Message)
}

type msgHandler func(msg model.Message, resourceType string)

type WorkConfig struct {
	Workers      int32
	Buffer       int32
	Action       string
	ResourceType string
	Handler      msgHandler
}

type msgWorker struct {
	workers      int32
	handler      msgHandler
	action       string
	resourceType string
	msgQueue     chan model.Message
}

func NewWorker(config WorkConfig) Worker {
	msgQueue := make(chan model.Message, config.Buffer)
	return &msgWorker{
		workers:      config.Workers,
		handler:      config.Handler,
		msgQueue:     msgQueue,
		action:       config.Action,
		resourceType: config.ResourceType,
	}
}

func (w *msgWorker) Start() {
	for i := 0; i < int(w.workers); i++ {
		go w.Consume()
	}
}

func (w *msgWorker) Produce(message model.Message) {
	w.msgQueue <- message
}

func (w *msgWorker) Consume() {
	for {
		select {
		case <-context.Done():
			klog.Warningf("stop %s", w.action)
			return

		case msg := <-w.msgQueue:
			klog.V(4).InfoS("Get a upstream message",
				"message ID", msg.GetID(),
				"message operation", msg.GetOperation(),
				"message resource", msg.GetResource())

			w.handler(msg, w.resourceType)
		}
	}
}
