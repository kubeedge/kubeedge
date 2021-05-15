package record

import (
	"context"
	"encoding/json"
	"fmt"

	v1 "k8s.io/api/core/v1"
	metaV1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/klog/v2"

	"github.com/kubeedge/kubeedge/cloud/pkg/dynamiccontroller/application"
)

var eventsKind = schema.GroupVersionKind{Group: "", Version: "v1", Kind: "Event"}

type SimpleEventSink struct {
	*application.Agent
}

func NewSimpleEventSink(nodeName string) *SimpleEventSink {
	return &SimpleEventSink{application.NewApplicationAgent(nodeName)}
}

func getKey(e *v1.Event) string {
	return fmt.Sprintf("/core/v1/events/%s/%s", e.GetNamespace(), e.GetName())
}

func (s *SimpleEventSink) Create(e *v1.Event) (*v1.Event, error) {
	e.SetGroupVersionKind(eventsKind)
	app := s.Agent.GenerateWithKey(context.Background(), application.Create, metaV1.CreateOptions{}, e, getKey(e))
	err := s.Agent.Apply(app)
	defer app.Close()
	if err != nil {
		klog.V(3).Infof("[simpleeventsink] failed to create event, %v", err)
		return nil, err
	}

	retObj := new(v1.Event)
	if err := json.Unmarshal(app.RespBody, retObj); err != nil {
		return nil, err
	}
	return retObj, nil
}

func (s *SimpleEventSink) Update(e *v1.Event) (*v1.Event, error) {
	e.SetGroupVersionKind(eventsKind)
	app := s.Agent.GenerateWithKey(context.Background(), application.Update, metaV1.UpdateOptions{}, e, getKey(e))
	err := s.Agent.Apply(app)
	defer app.Close()
	if err != nil {
		klog.V(3).Infof("[simpleeventsink] failed to update event, %v", err)
		return nil, err
	}

	retObj := new(v1.Event)
	if err := json.Unmarshal(app.RespBody, retObj); err != nil {
		return nil, err
	}
	return retObj, nil
}

func (s *SimpleEventSink) Patch(e *v1.Event, p []byte) (*v1.Event, error) {
	e.SetGroupVersionKind(eventsKind)
	pi := application.PatchInfo{
		Name:      e.Name,
		PatchType: types.StrategicMergePatchType,
		Data:      p,
	}
	app := s.Agent.GenerateWithKey(context.Background(), application.Patch, pi, nil, getKey(e))
	err := s.Agent.Apply(app)
	defer app.Close()
	if err != nil {
		klog.V(3).Infof("[simpleeventsink] failed to patch event, %v", err)
		return nil, err
	}

	retObj := new(v1.Event)
	if err := json.Unmarshal(app.RespBody, retObj); err != nil {
		return nil, err
	}
	return retObj, nil
}
